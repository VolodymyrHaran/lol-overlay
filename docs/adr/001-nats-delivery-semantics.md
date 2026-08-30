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

Game lifecycle events may later trigger statistics, analytics and other
business operations. Losing these events is not acceptable.

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

Processed event markers are retained for 30 days. A background cleanup runs
once per day and removes older markers. The retention period is longer than
the seven-day maximum age of the `GAME_EVENTS` stream.

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
- consumer-side idempotency;
- durable dead-letter storage for poison messages;
- observable delivery outcomes;
- bounded inbox-table growth;
- independent delivery semantics for different event categories.

Trade-offs:

- Core NATS room notifications may be lost;
- JetStream provides at-least-once rather than exactly-once delivery;
- consumers must remain idempotent;
- producer database changes and event publication are not atomic;
- inbox markers and future business changes must share a transaction;
- dead-letter messages currently require manual inspection and replay.

## Verification

The implementation is covered by:

- unit tests for game lifecycle transitions;
- unit tests for publisher retry with the same event ID;
- JetStream publication deduplication integration tests;
- JetStream unacknowledged-message redelivery integration tests;
- JetStream dead-letter routing integration tests;
- consumer duplicate and repository-error tests;
- PostgreSQL inbox repository integration tests;
- processed-event retention tests.

## Future work

- transactional inbox processing with business changes;
- transactional outbox for atomic database changes and event publication;
- alerts and dashboards for retry and dead-letter metrics;
- controlled replay tooling for dead-letter events.
