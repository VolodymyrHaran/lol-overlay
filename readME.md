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
 Champion Select Sync
          │
          ▼
     RoomService
          │
          │ EventPublisher
          ▼
       Core NATS
          │
          ▼
     RoomConsumer
          │
          ├── room.current.changed
          │         │
          │         ▼
          │   Current Room WebSocket
          │
          └── room.updated
                    │
                    ▼
              Room WebSocket
                    │
                    ▼
                 React UI
```

`RoomService` depends on an `EventPublisher` interface rather than directly
on the NATS connection or WebSocket Hub. `RoomConsumer` receives transient
events, loads the latest room state through the repository/cache layer, and
broadcasts it to connected WebSocket clients.

Core NATS is currently used with at-most-once delivery. Events are transient:
if an update is lost, clients recover the latest state when reconnecting.
JetStream is planned for scenarios requiring durable delivery, acknowledgements,
retries, and replay.

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

## Docker

```bash
docker compose up -d
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

---

## Roadmap
- [x] Core NATS integration
- [ ] NATS delivery semantics and queue groups
- [ ] NATS JetStream
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
