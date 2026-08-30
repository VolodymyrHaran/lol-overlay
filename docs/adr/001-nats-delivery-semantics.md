# ADR 001: NATS delivery semantics

## Status

Accepted and implemented

## Context

The application publishes two categories of events with different delivery
requirements.

Transient room events:

- `room.current.changed`
- `room.updated`

Durable game lifecycle events:

- `game.started`
- `game.ended`

PostgreSQL is the source of truth for room state. Room events only notify
consumers that the latest state should be loaded and broadcast to WebSocket
clients.

Game lifecycle events persist game sessions and may later trigger statistics,
analytics and other business operations. Losing these events is not
acceptable.

## Decision

### Transient room events

Use Core NATS for room notifications.

Core NATS provides at-most-once delivery. A disconnected consumer may miss an
event, and messages are not persisted or replayed.

This is acceptable because:

- room events are transient UI notifications;
- PostgreSQL and Redis retain the latest room state;
- consumers load room state by room ID;
- WebSocket clients receive a snapshot when connecting;
- later room updates supersede earlier updates.

Queue groups are not used for room WebSocket consumers. Every application
instance must receive events for its locally connected clients.

### Durable game events

Use NATS JetStream for game lifecycle events.

The `GAME_EVENTS` stream stores:

- `game.started`
- `game.ended`

Each event contains:

- a UUID event ID;
- occurrence time;
- schema version;
- game ID;
- room ID.

The event ID is also used as the JetStream message ID. Repeated publication
with the same message ID is deduplicated within the configured duplicate
window.

The `game-events-processor` durable consumer uses:

- explicit acknowledgements;
- confirmed acknowledgements with `DoubleAck`;
- delayed negative acknowledgements after processing errors;
- application-controlled retry attempts;
- at-least-once delivery semantics.

### Idempotent consumption

JetStream may deliver the same event more than once. The consumer uses the
PostgreSQL `processed_events` inbox table to detect duplicates.

The primary key is:

```text
consumer_name + event_id
```

This allows different consumers to process the same event independently while
preventing one consumer from processing it repeatedly.

A duplicate is treated as successfully handled and acknowledged. A PostgreSQL
error is returned to the JetStream callback, causing negative acknowledgement
and redelivery.

Inbox claiming and game-session persistence are executed in one PostgreSQL
transaction. `game.started` inserts or updates `game_sessions`, while
`game.ended` sets the end time of an existing session. If the business change
fails, the inbox marker is rolled back as part of the same transaction. A
redelivered message can therefore retry the complete operation safely.

The consumer does not call a standalone `TryMarkProcessed` operation. This
prevents the failure window in which a marker could be committed before the
corresponding business change.

Processed event markers are retained for 30 days. A background cleanup runs
once per day and removes older markers. The retention period is longer than
the seven-day maximum age of the `GAME_EVENTS` stream.

### Transactional outbox

Game lifecycle transitions are persisted to the PostgreSQL `outbox_events`
table before they are published to JetStream. The outbox row contains the event
ID, subject, JSON payload, creation time, next available time, publication time,
attempt count and last publication error.

The event ID is shared by the outbox row, event payload and JetStream message
ID. If enqueueing temporarily fails, `GameLifecycleService` retains the same
event in memory and retries the identical ID and payload on the next
observation.

A background relay claims pending rows in batches using
`FOR UPDATE SKIP LOCKED`. Claiming moves `available_at` to a 30-second lease
deadline. Multiple application instances can therefore run relays without
claiming the same row concurrently. If an instance stops after claiming a row,
the event becomes available again when the lease expires.

Publication failures are recorded in the outbox and retried with exponential
backoff starting at five seconds and capped at five minutes. Successful
publication sets `published_at` and clears the last error. If JetStream accepts
an event but the database update fails, the lease eventually expires and the
relay publishes the same message ID again. JetStream deduplication and the
idempotent consumer make this failure mode safe.

Published outbox rows are retained for 30 days and removed by a daily cleanup.
Pending and failed rows are never removed by retention cleanup.

Relay behavior is exposed through:

- `lol_timer_outbox_relay_events_total`;
- `lol_timer_outbox_relay_duration_seconds`;
- `lol_timer_outbox_cleanup_deleted_total`.

### Dead-letter handling

Processing failures are retried with delayed negative acknowledgements. After
the fifth failed delivery, the original subject, payload, processing error and
JetStream metadata are published to the `GAME_EVENTS_DLQ` stream on the
`dead.game` subject.

After the dead-letter publication succeeds, the source message is terminated
with `TermWithReason`. The DLQ message ID is derived from the source stream,
stream sequence and consumer, so repeating the transfer does not create
duplicate dead-letter records.

The JetStream consumer itself has unlimited delivery attempts. This is
intentional: if the DLQ stream is temporarily unavailable, the source message
continues to be retried instead of being silently stranded after the fifth
attempt. Dead-letter records are retained for 30 days.

Delivery outcomes are exposed through the Prometheus counter
`lol_timer_game_event_delivery_outcomes_total`, labelled by subject and one of
the following outcomes:

- `acked`;
- `ack_error`;
- `retried`;
- `retry_error`;
- `dead_lettered`.

## Consequences

Advantages:

- low-latency transient room updates;
- durable game lifecycle events;
- explicit acknowledgement and retry behavior;
- publisher-side deduplication;
- durable producer-side handoff through PostgreSQL outbox;
- safe multi-instance claiming with leases and `SKIP LOCKED`;
- bounded exponential publication retry;
- consumer-side idempotency;
- atomic inbox claiming and game-session persistence;
- durable dead-letter storage for poison messages;
- observable delivery outcomes;
- bounded inbox-table growth;
- independent delivery semantics for different event categories.

Trade-offs:

- Core NATS room notifications may be lost;
- JetStream provides at-least-once rather than exactly-once delivery;
- consumers must remain idempotent;
- the relay may publish an event more than once if finalizing its outbox row
  fails;
- dead-letter messages currently require manual inspection and replay.

## Verification

The implementation is covered by:

- unit tests for game lifecycle transitions;
- unit tests for publisher retry with the same event ID;
- JetStream publication deduplication integration tests;
- JetStream unacknowledged-message redelivery integration tests;
- JetStream dead-letter routing integration tests;
- consumer duplicate and repository-error tests;
- PostgreSQL transactional game-event repository integration tests;
- rollback verification for failed game-session changes;
- processed-event retention tests;
- PostgreSQL outbox enqueue, lease, retry, publication and cleanup integration
  tests;
- outbox relay and exponential backoff unit tests;
- manual end-to-end verification from outbox enqueue through game-session
  persistence.

## Future work

- alerts and dashboards for retry and dead-letter metrics;
- automated end-to-end outbox pipeline integration tests;
- controlled replay tooling for dead-letter events.
