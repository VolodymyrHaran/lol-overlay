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
- 📤 PostgreSQL transactional outbox with retry and leases
- 🔌 gRPC Champion Service backed by Riot Data Dragon

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
- gRPC
- Protocol Buffers
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
          ├── RoomService ── gRPC ── Champion Service ── Data Dragon
          │       │
          │       └── Core NATS ── RoomConsumer
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
                 PostgreSQL Outbox
                         │
                         ▼
                    Outbox Relay
                         │
                         ▼
                  NATS JetStream
                    GAME_EVENTS
                         │
                         ▼
                   GameConsumer
                         │
                         ▼
          Transactional Inbox / game_sessions
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

Game lifecycle production uses the PostgreSQL `outbox_events` table. Lifecycle
events are persisted before publication, then a background relay claims pending
rows with `FOR UPDATE SKIP LOCKED` and a 30-second lease. Failed publications
use exponential backoff from five seconds up to five minutes. Published rows
are retained for 30 days; pending and failed rows are never removed by cleanup.

Champion metadata is isolated behind a unary gRPC API. The main backend keeps
one long-lived HTTP/2 connection to Champion Service and propagates request
contexts with a two-second client timeout. Champion Service owns Data Dragon
loading and its in-memory catalog. `InvalidArgument`, `NotFound`,
`DeadlineExceeded` and `Unavailable` remain distinguishable gRPC status codes.
Because champion metadata is non-critical, temporary RPC failures fall back to
an `Unknown` champion instead of blocking room synchronization.

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

Before reaching JetStream, both event types are stored in `outbox_events` with
the same event ID and JSON payload. The relay marks a row with `published_at`
only after JetStream acknowledges publication. Repeated publication uses the
same message ID and remains safe for the idempotent consumer.

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

The `dlq-replay` command provides a controlled recovery path. Preview the most
recent message without changing it:

```bash
go run ./cmd/dlq-replay -latest
```

After inspecting the output, replay a specific sequence explicitly:

```bash
go run ./cmd/dlq-replay -sequence 17 -execute
```

Replay republishes the original subject and payload with a deterministic replay
message ID. The DLQ record is deleted only after JetStream acknowledges the
publication. `-latest -execute` is intentionally rejected so an operator must
inspect and explicitly select the message before a destructive replay.

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

Start Champion Service first:

```bash
go run ./cmd/champion-service
```

Then start the main backend. It uses `localhost:50051` by default:

```bash
go run ./cmd/server
```

Docker Compose builds separate runtime targets for both Go binaries and uses
the internal address `champion-service:50051`:

```bash
docker compose up -d --build
```

### Champion gRPC API

The protobuf contract is defined in
`api/proto/champion/v1/champion.proto`. Generated files under
`gen/champion/v1` must not be edited manually.

Available unary RPCs:

```text
champion.v1.ChampionService/GetChampion
champion.v1.ChampionService/ListChampions
```

Example PowerShell request:

```powershell
'{"championId":103}' |
    grpcurl `
        -plaintext `
        -import-path api/proto `
        -proto champion/v1/champion.proto `
        -d '@' `
        localhost:50051 `
        champion.v1.ChampionService/GetChampion
```

The service also registers the standard gRPC Health service. Unary client and
server interceptors log the full RPC method, status code and duration without
high-cardinality champion or player identifiers.

### Kubernetes with kind

The project can run in a local Kubernetes cluster created with kind. The kind
configuration pins the Kubernetes node image and maps the backend NodePort to
`http://localhost:18080`.

Create the cluster:

```powershell
kind create cluster `
    --name lol-overlay `
    --config deploy/kind/cluster.yaml `
    --wait 120s
```

Create the namespace and shared non-sensitive configuration:

```powershell
kubectl apply -f deploy/kubernetes/namespace.yaml
kubectl apply -f deploy/kubernetes/configmap.yaml
```

Create local secrets without committing credentials. The required keys are
documented in `deploy/kubernetes/secret.example.yaml`.

```powershell
kubectl create secret generic lol-overlay-secrets `
    --namespace lol-overlay `
    --from-literal=POSTGRES_USER=lol_timer `
    --from-literal=POSTGRES_PASSWORD=lol_timer `
    --from-literal=POSTGRES_DB=lol_timer `
    --from-literal=DATABASE_URL='postgres://lol_timer:lol_timer@postgres:5432/lol_timer?sslmode=disable' `
    --from-literal=REDIS_PASSWORD=''

kubectl create secret generic grafana-admin `
    --namespace lol-overlay `
    --from-literal=GF_SECURITY_ADMIN_USER=admin `
    --from-literal=GF_SECURITY_ADMIN_PASSWORD=admin
```

Build the project images and load them into kind:

```powershell
docker build --target champion-service --tag lol-overlay/champion-service:local .
docker build --target migrations --tag lol-overlay/migrations:local .
docker build --target app --tag lol-overlay/app:v1 .

kind load docker-image lol-overlay/champion-service:local --name lol-overlay
kind load docker-image lol-overlay/migrations:local --name lol-overlay
kind load docker-image lol-overlay/app:v1 --name lol-overlay
```

Apply the infrastructure, run the migrations, and start the application:

```powershell
kubectl apply -f deploy/kubernetes/postgres.yaml
kubectl apply -f deploy/kubernetes/redis.yaml
kubectl apply -f deploy/kubernetes/nats.yaml
kubectl apply -f deploy/kubernetes/champion-service.yaml
kubectl apply -f deploy/kubernetes/migrations-job.yaml

kubectl wait -n lol-overlay `
    --for=condition=complete `
    job/database-migrations `
    --timeout=120s

kubectl apply -f deploy/kubernetes/app.yaml
kubectl apply -f deploy/kubernetes/prometheus.yaml
kubectl apply -f deploy/kubernetes/grafana.yaml
```

Inspect the workload and wait for the backend rollout:

```powershell
kubectl get pods,services,jobs,pvc -n lol-overlay
kubectl rollout status deployment/app -n lol-overlay --timeout=120s
```

Prometheus and Grafana are available through local port forwarding:

```powershell
kubectl port-forward -n lol-overlay service/prometheus 19090:9090
kubectl port-forward -n lol-overlay service/grafana 13000:3000
```

The manifests use Deployments for stateless services, StatefulSets and
persistent volumes for stateful infrastructure, readiness and liveness probes,
resource requests and limits, two backend replicas, and rolling updates.

### Frontend

```bash
cd frontend

npm install

npm run dev
```

---

## Tests

Run unit tests and static analysis:

```bash
go test ./...
go vet ./...
```

Integration tests require PostgreSQL with applied migrations and NATS with
JetStream enabled.

PowerShell:

```powershell
$env:TEST_DATABASE_URL = $env:DATABASE_URL
$env:NATS_URL = "nats://localhost:4222"

go test -tags=integration ./... -count=1 -p=1
```

The integration suite covers JetStream deduplication and redelivery,
dead-letter routing, PostgreSQL repositories, and the complete lifecycle event
pipeline:

```text
outbox_events
      │
      ▼
OutboxRelayService
      │
      ▼
NATS JetStream
      │
      ▼
GameConsumer
      │
      ├── processed_events
      └── game_sessions
```

GitHub Actions automatically checks formatting, runs `go vet` and race-enabled
unit tests, starts PostgreSQL, Redis and NATS JetStream, applies migrations,
runs the integration suite, and builds the project.

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

Outbox metrics

```text
lol_timer_outbox_relay_events_total
lol_timer_outbox_relay_duration_seconds
lol_timer_outbox_cleanup_deleted_total
```

Relay outcomes include successful publication, scheduled retry, retry-state
errors and publication-finalization errors.

Dead-letter replay metric

```text
lol_timer_dead_letter_replay_outcomes_total
```

The `outcome` label reports `replayed` after publication and DLQ deletion both
succeed, or `replay_error` when any replay step fails. Event IDs and stream
sequences are intentionally excluded from labels.

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
- PostgreSQL outbox with leased multi-instance batch claiming
- Exponential outbox publication retry and published-event retention
- Dead-letter stream with deterministic transfer deduplication
- Controlled dead-letter preview and replay CLI
- Deterministic replay publication with publish-before-delete ordering
- Prometheus metrics for ACK, retry, outbox and dead-letter outcomes
- Unit and integration coverage for lifecycle, deduplication, redelivery and DLQ
- End-to-end outbox-to-inbox integration coverage
- GitHub Actions CI with PostgreSQL, Redis and NATS JetStream
- Versioned protobuf contract with generated Go client/server code
- Separate Champion Service with unary gRPC and standard health service
- Long-lived gRPC client connection with deadline propagation
- Graceful Champion Service shutdown and non-critical metadata fallback

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
- [x] Controlled dead-letter replay tooling
- [x] Transactional outbox
- [x] gRPC Champion Service
- [x] Protobuf code generation
- [x] Local Kubernetes deployment with kind
- [x] Kubernetes health probes and resource limits
- [x] Kubernetes Prometheus and Grafana deployment
- [ ] Helm chart
- [ ] ArgoCD GitOps deployment
- [ ] Electron desktop application
- [x] Riot Data Dragon integration
- [ ] Champion icons cache
- [ ] Settings window
- [ ] System tray
- [ ] Auto start with Windows
- [x] GitHub Actions CI
- [ ] Continuous deployment
- [ ] Releases
- [ ] Auto updater

---

## Screenshots

_Coming soon._

---
