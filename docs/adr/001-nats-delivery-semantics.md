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
- limited redelivery;
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

## Consequences

Advantages:

- low-latency transient room updates;
- durable game lifecycle events;
- explicit acknowledgement and retry behavior;
- publisher-side deduplication;
- consumer-side idempotency;
- bounded inbox-table growth;
- independent delivery semantics for different event categories.

Trade-offs:

- Core NATS room notifications may be lost;
- JetStream provides at-least-once rather than exactly-once delivery;
- consumers must remain idempotent;
- producer database changes and event publication are not atomic;
- inbox markers and future business changes must share a transaction;
- poison messages still require a dead-letter strategy.

## Verification

The implementation is covered by:

- unit tests for game lifecycle transitions;
- unit tests for publisher retry with the same event ID;
- JetStream publication deduplication integration tests;
- JetStream unacknowledged-message redelivery integration tests;
- consumer duplicate and repository-error tests;
- PostgreSQL inbox repository integration tests;
- processed-event retention tests.

## Future work

- transactional inbox processing with business changes;
- dead-letter handling after maximum delivery attempts;
- transactional outbox for atomic database changes and event publication;
- metrics and alerts for redelivery and dead-letter events.
