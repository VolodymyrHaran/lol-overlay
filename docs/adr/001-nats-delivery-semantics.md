# ADR 001: NATS delivery semantics

## Status

Accepted

## Context

The application publishes room state notifications after mutations.

Current subjects:

- `room.current.changed`
- `room.updated`

PostgreSQL is the source of truth. NATS events notify consumers that
state should be reloaded. WebSocket clients also receive the latest
snapshot when connecting.

## Decision

Use Core NATS for transient room events.

Core NATS provides at-most-once delivery. A disconnected consumer can
miss an event, and messages are not persisted or replayed.

This is acceptable because:

- room events are transient UI notifications;
- the latest state remains available in PostgreSQL and Redis;
- consumers load state by room ID;
- WebSocket clients receive a snapshot when connecting;
- later events supersede earlier room updates.

Do not use queue groups for WebSocket consumers. Every application
instance must receive events to update its locally connected clients.

Use JetStream later for durable domain events such as:

- `game.started`
- `game.ended`

Durable consumers must support duplicate delivery and idempotent
processing.

## Consequences

Advantages:

- low latency;
- simple operation;
- no acknowledgement handling;
- no durable stream growth for per-second cooldown events.

Trade-offs:

- transient events may be lost;
- producer state changes and event publication are not atomic;
- temporary UI staleness is possible;
- critical events require a different delivery mechanism.

## Future work

- JetStream streams and durable consumers;
- explicit acknowledgements;
- retry and dead-letter strategy;
- idempotency keys;
- transactional outbox for critical database changes.