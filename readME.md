# LoL Timer

Real-time League of Legends summoner spell tracker built with Go.

The application synchronizes with the League Client (LCU), tracks enemy summoner spell cooldowns, stores room state in PostgreSQL, caches data in Redis, and exposes Prometheus metrics with Grafana dashboards.

---

## Features

- Real-time cooldown tracking
- Automatic Champion Select synchronization
- REST API
- WebSocket updates
- PostgreSQL persistence
- Redis caching
- Prometheus metrics
- Grafana dashboard
- Swagger API documentation
- Docker Compose
- Health & Readiness endpoints
- GitHub Actions CI
- Unit tests
- Race detector support

---

## Architecture

```
                   League Client (LCU)
                           │
                           ▼
                  Champion Select Sync
                           │
                           ▼
                     Room Service
                           │
           ┌───────────────┴───────────────┐
           ▼                               ▼
     Redis Cache                    PostgreSQL
           │                               │
           └───────────────┬───────────────┘
                           ▼
                     REST API
                           │
                           ▼
                      WebSocket
                           │
                           ▼
                     React Frontend
```

---

## Tech Stack

### Backend

- Go 1.26
- net/http
- WebSocket
- PostgreSQL
- Redis

### Frontend

- React
- TypeScript
- TailwindCSS

### Infrastructure

- Docker
- Docker Compose
- Prometheus
- Grafana

### Documentation

- Swagger / OpenAPI

### Quality

- GitHub Actions
- Unit Tests
- Race Tests

---

## Project Structure

```
cmd/
    server/

internal/
    app/
    cache/
    config/
    database/
    dto/
    handlers/
    logger/
    metrics/
    middleware/
    models/
    repositories/
    services/
    websocket/

docs/

monitoring/

frontend/
```

---

## API

### Health

```
GET /health
```

### Readiness

```
GET /ready
```

### Room

```
GET /rooms/{roomId}
```

### Add player

```
POST /rooms/{roomId}/players
```

### Toggle spell

```
POST /rooms/{roomId}/spells/toggle
```

### Metrics

```
GET /metrics
```

### Swagger

```
GET /swagger/index.html
```

---

## Monitoring

Prometheus metrics include

- HTTP requests
- HTTP latency
- Active rooms
- Redis cache hits
- Redis cache misses
- Repository operations
- WebSocket connections
- Go runtime metrics

Grafana dashboard includes

- Active rooms
- WebSocket connections
- Cache hit ratio
- HTTP requests/sec
- HTTP latency
- Repository operations
- Memory usage
- CPU usage
- Goroutines

---

## Running locally

Clone repository

```bash
git clone https://github.com/YOUR_USERNAME/lol-timer.git

cd lol-timer
```

Create environment file

```env
DATABASE_URL=postgres://lol_timer:password@localhost:5432/lol_timer?sslmode=disable

REDIS_ADDRESS=localhost:6379

REDIS_PASSWORD=

REDIS_DATABASE=0

ROOM_CACHE_TTL=10m

HTTP_ADDRESS=:8080

LOG_LEVEL=info
```

Start infrastructure

```bash
docker compose up -d
```

Run application

```bash
go run ./cmd/server
```

or

```bash
docker compose up --build
```

---

## Testing

Run all tests

```bash
go test ./...
```

Race detector

```bash
go test -race ./...
```

Format

```bash
go fmt ./...
```

Vet

```bash
go vet ./...
```

---

## CI

Every push automatically runs

- gofmt
- go vet
- go test
- go test -race
- build verification

---

## Documentation

Swagger UI

```
http://localhost:8080/swagger/index.html
```

Prometheus

```
http://localhost:9090
```

Grafana

```
http://localhost:3000
```

---

## Roadmap

- Automatic room creation
- Automatic room cleanup on game end
- Match history
- Improved frontend
- Authentication
- Spectator mode

---
