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
 ┌────────┴─────────┐
 │                  │
 ▼                  ▼
Current Room WS   Room WS
 │                  │
 ▼                  ▼
 React UI       Live Updates
```

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

---

## Roadmap

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
