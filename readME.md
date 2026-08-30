# LoL Group Helper

A real-time League of Legends summoner spell tracker built with Go, React and WebSockets.

The application automatically detects the current Champion Select session, synchronizes players from the League Client (LCU API), and allows teammates to track enemy summoner spell cooldowns in real time.

---

## Features

- 🎮 Automatic Champion Select detection
- 👥 Automatic room creation
- 🔄 Real-time synchronization via WebSocket
- ⚡ Summoner spell cooldown tracking
- 📡 League Client (LCU) integration
- 🗄 PostgreSQL persistence
- 🚀 Redis cache
- 📊 Prometheus metrics
- 📖 Swagger API
- ❤️ Health & Readiness endpoints
- 🐳 Docker support
- 🎨 React + TypeScript frontend
- 🌙 Modern UI built with Tailwind CSS + shadcn/ui
- 📨 Event-driven room updates via NATS
- ⏱ Event-driven cooldown synchronization without database polling
- 📦 Durable game lifecycle events via NATS JetStream
- 🚨 Dead-letter handling for failed game events

---

## Tech Stack

### Backend

- Go
- Gorilla WebSocket
- PostgreSQL
- Redis
- Docker
- Prometheus
- Swagger
- Repository Pattern
- NATS
- NATS JetStream
- Event-driven architecture

### Frontend

- React
- TypeScript
- Vite
- Tailwind CSS
- shadcn/ui

---

## Architecture

```
League Client (LCU)
          │
          ▼
 Champion Select / Gameflow Sync
          │
          ▼
          ├── RoomService ── Core NATS ── RoomConsumer
          │                         │
          │                         ├── room.current.changed
          │                         └── room.updated
          │                                  │
          │                                  ▼
          │                              WebSockets
          │                                  │
          │                                  ▼
          │                               React UI
          │
          └── GameLifecycleService
                         │
                         ▼
                  NATS JetStream
                    GAME_EVENTS
                         │
                         ▼
                   GameConsumer
```

`RoomService` depends on an `EventPublisher` interface rather than directly
on the NATS connection or WebSocket Hub. `RoomConsumer` receives transient
events, loads the latest room state through the repository/cache layer, and
broadcasts it to connected WebSocket clients.

Core NATS is used for transient room notifications with at-most-once delivery.
If an update is lost, clients recover the latest state from PostgreSQL/Redis when
reconnecting or receiving a later update.

NATS JetStream is used for durable game lifecycle events. The `GAME_EVENTS`
stream stores `game.started` and `game.ended`. Events include a unique event ID,
occurrence time and schema version. Publishers use the event ID as a JetStream
message ID, allowing duplicate publications to be deduplicated. The durable
consumer uses explicit acknowledgements and delayed negative acknowledgements,
providing at-least-once delivery. Consumers must therefore process these events
idempotently. After five failed deliveries, events are moved to the
`GAME_EVENTS_DLQ` stream. If the DLQ is unavailable, delivery continues so the
source event is not silently lost.

Game lifecycle consumption uses a transactional inbox. The `processed_events`
marker and the corresponding `game_sessions` insert or update are committed in
one PostgreSQL transaction. A failed business change rolls back the marker, so
JetStream redelivery can retry the complete operation safely.

---

## Project Structure

```
backend
│
├── cmd/
├── internal/
│   ├── app/
│   ├── cache/
│   ├── config/
│   ├── consumers/
│   ├── messaging/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── repositories/
│   ├── services/
│   └── websocket/
│
├── docs/
└── docker/

frontend
│
├── src/
│   ├── components/
│   ├── config/
│   ├── hooks/
│   ├── types/
│   └── ui/
```

---


## API

### Health

```
GET /health
```

### Ready

```
GET /ready
```

### Swagger

```
GET /swagger/index.html
```

### Toggle spell

```
POST /rooms/{roomId}/spells/toggle
```

### Metrics

```
GET /metrics
```
## NATS Events

### Current room changed

Subject:

```text
room.current.changed
```
```json
{
  "roomId": "7961620711-1"
}
```

An empty `roomId` means that the current Champion Select session has ended.

### Room updated

Subject:

```text
room.updated
```

This transient notification tells `RoomConsumer` to load the latest room state
and broadcast it to connected WebSocket clients.

### Game started and ended

JetStream subjects:

```text
game.started
game.ended
```

Example:

```json
{
  "eventId": "550e8400-e29b-41d4-a716-446655440000",
  "occurredAt": "2026-08-26T15:00:00Z",
  "version": 1,
  "gameId": 123456789,
  "roomId": "7961620711-1"
}
```

These events are stored in the `GAME_EVENTS` stream and processed by the
`game-events-processor` durable consumer. `game.started` creates or updates a
row in `game_sessions`; `game.ended` records its completion time. The room ID is
stored as historical data without a foreign key because temporary rooms may be
deleted after Champion Select.

### Dead-letter events

Subject and stream:

```text
subject: dead.game
stream:  GAME_EVENTS_DLQ
```

After five failed processing attempts, the original subject and payload,
processing error, delivery count and source JetStream metadata are stored in
the DLQ for inspection or controlled replay. A successful transfer terminates
the source message with `TermWithReason`. Transfers are deduplicated by source
stream, sequence and consumer.

--- 

## WebSocket API

### Current room

```
/ws/current-room
```

Message

```json
{
  "type": "current_room",
  "roomId": "7941931125-1"
}
```

---

### Room updates

```
/ws?roomId={roomId}
```

Message

```json
{
  "type": "room_update",
  "room": {}
}
```

---

## Running locally

### Infrastructure

Start PostgreSQL, Redis, NATS JetStream, Prometheus and Grafana:

```bash
docker compose up -d
```

### Backend

```bash
go run ./cmd/server
```

### Frontend

```bash
cd frontend

npm install

npm run dev
```

---

## Tests

```bash
go test ./...
go vet ./...
```

JetStream integration tests require the NATS service to be running:

```bash
go test -tags=integration ./internal/messaging -v
```

---

## Monitoring

Prometheus metrics

```
/metrics
```

Health

```
/health
```

Ready

```
/ready
```

Swagger

```
/swagger/index.html
```

NATS monitoring

```text
http://localhost:8222
```

Game-event delivery metric

```text
lol_timer_game_event_delivery_outcomes_total
```

The `outcome` label reports `acked`, `ack_error`, `retried`, `retry_error` or
`dead_lettered`. Event IDs and game IDs are intentionally excluded from labels
to avoid high-cardinality Prometheus metrics.

---

## Current Features

- Automatic League Client detection
- Champion Select synchronization
- Automatic room creation
- Real-time room updates
- Automatic reconnect
- Redis room cache
- PostgreSQL repository
- Cooldown calculation
- Cooldown persistence during Champion Select
- WebSocket-based frontend synchronization
- Core NATS event publishing and consumption
- Event-driven current-room synchronization
- Event-driven room and cooldown updates
- Room update deduplication
- WebSocket updates without database polling
- Automatic game lifecycle detection from LCU gameflow phases
- Durable `game.started` and `game.ended` events
- JetStream publisher deduplication by event ID
- Durable consumer with explicit ACK, delayed NAK and redelivery
- Transactional inbox with atomic `game_sessions` persistence
- Persistent game start and end timestamps
- Dead-letter stream with deterministic transfer deduplication
- Prometheus metrics for ACK, retry and dead-letter outcomes
- Unit and integration coverage for lifecycle, deduplication, redelivery and DLQ

---

## Roadmap
- [x] Core NATS integration
- [x] NATS delivery semantics documented
- [x] NATS JetStream
- [x] Durable game lifecycle publisher and consumer
- [x] Idempotent game event processing
- [x] Transactional inbox processing
- [x] Game session persistence
- [x] Processed event retention and cleanup
- [x] Dead-letter strategy
- [ ] Transactional outbox
- [ ] Electron desktop application
- [ ] Riot Data Dragon integration
- [ ] Champion icons cache
- [ ] Settings window
- [ ] System tray
- [ ] Auto start with Windows
- [ ] GitHub Actions CI/CD
- [ ] Releases
- [ ] Auto updater

---

## Screenshots

_Coming soon._

---
