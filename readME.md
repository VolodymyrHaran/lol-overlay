# LoL Timer

A backend service for tracking League of Legends summoner spell cooldowns in real time.

The application automatically synchronizes the current Champion Select session through the League Client API (LCU), creates game rooms, stores players and their summoner spells, broadcasts updates through WebSocket, and persists data in PostgreSQL with Redis caching.

LoL Timer is a production-oriented Go backend project demonstrating clean architecture, repository pattern, Redis caching, PostgreSQL persistence, WebSockets, dependency injection, graceful shutdown, structured logging, and comprehensive testing while tracking League of Legends summoner spell cooldowns in real time.

---

## Features

- Champion Select synchronization via League Client API
- Automatic room creation
- Automatic player synchronization
- Real-time spell cooldown tracking
- WebSocket updates
- REST API
- PostgreSQL persistence
- Redis cache (Cache-Aside pattern)
- Repository Pattern
- Dependency Injection
- Graceful Shutdown
- Structured logging (slog)
- Docker support
- Unit tests
- Integration tests
- Context propagation
- Automatic room cleanup

---

## Tech Stack

### Language

- Go 1.25+

### Database

- PostgreSQL

### Cache

- Redis

### API

- REST
- WebSocket

### Infrastructure

- Docker
- Docker Compose

### Architecture

- Repository Pattern
- Cached Repository Decorator
- Dependency Injection
- Context propagation

### Testing

- Unit Tests
- Integration Tests
- Race Detector

---

## Architecture

```
                    HTTP
                      │
                RoomHandler
                      │
                RoomService
                      │
             RoomRepository
                      │
      CachedRoomRepository
          │            │
          │            │
      Redis Cache   PostgreSQL
```

---

## Project Structure

```
cmd/
    server/

internal/
    app/
    cache/
    config/
    constants/
    database/
    dto/
    handlers/
    logger/
    middleware/
    metrics/
    models/
    repositories/
        postgres/
    services/
    websocket/

migrations/
docker/
```

---

## Running

### Clone

```bash
git clone https://github.com/<username>/lol-timer.git

cd lol-timer
```

### Environment

Create `.env`

```env
DATABASE_URL=postgres://lol_timer:lol_timer_password@localhost:5432/lol_timer?sslmode=disable

POSTGRES_USER=lol_timer
POSTGRES_PASSWORD=lol_timer_password
POSTGRES_DB=lol_timer
POSTGRES_PORT=5432

REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
REDIS_DATABASE=0
ROOM_CACHE_TTL=5m

HTTP_ADDRESS=:8080

LOG_LEVEL=debug
```

---

### Start containers

```bash
docker compose up -d
```

---

### Run migrations

```bash
go run ./cmd/migrate
```

---

### Run server

```bash
go run ./cmd/server
```

---

## REST API

### Get Room

```
GET /rooms/{roomId}
```

---

### Toggle Spell

```
POST /rooms/{roomId}/spells/toggle
```

Request

```json
{
  "gameName": "Player",
  "tagLine": "EUW",
  "spell": "Flash"
}
```

---

### WebSocket

```
ws://localhost:8080/ws?roomId=<roomId>
```

Clients automatically receive room updates whenever player state changes.

---

## Testing

Run all tests

```bash
go test ./...
```

Run race detector

```bash
go test -race ./...
```

Run integration tests

```bash
go test ./internal/repositories/...
```

---

## Redis Cache

The application uses the Cache-Aside pattern.

```
Request
    │
    ▼
Redis Cache
    │
 Cache Miss
    │
    ▼
PostgreSQL
    │
    ▼
Redis Update
    │
    ▼
Response
```

---

## Logging

The application uses:

- slog
- Recovery middleware
- HTTP request logging

Every request logs:

- method
- path
- status
- duration
- remote address
- user agent

---

## Current Status

Implemented

- PostgreSQL persistence
- Redis cache
- Champion Select synchronization
- WebSocket broadcasting
- REST API
- Repository Pattern
- Cached Repository
- Graceful Shutdown
- Context propagation
- Structured logging
- Docker support
- Unit tests
- Integration tests

Planned

- Prometheus
- Grafana
- gRPC API
- NATS / RabbitMQ
- Authentication
- OpenAPI / Swagger

---