# ADR 002: Extract champion metadata behind gRPC

## Status

Accepted.

## Context

The main backend previously loaded Riot Data Dragon directly and kept champion
metadata in an in-memory map. Champion lookup is read-only, independent of room
and cooldown persistence, and has no dependency on PostgreSQL, Redis or NATS.
This makes it a small boundary for learning and applying gRPC without moving
critical game or room logic out of the existing backend.

## Decision

Run champion catalog ownership in a separate `champion-service` process. The
service loads the latest Data Dragon catalog before opening its listener and
exposes the versioned `champion.v1.ChampionService` protobuf contract.

The API contains two unary RPCs:

- `GetChampion` for lookup by numeric champion ID;
- `ListChampions` for the complete catalog and Data Dragon version.

The main backend depends on a `ChampionCatalog` interface. Its production
implementation is a gRPC client; tests may use the existing in-memory catalog
or a stub. A single `grpc.ClientConn` is created during application assembly,
reused across RPCs and closed during application shutdown.

Each client call derives a two-second timeout from the caller's context. An
earlier parent deadline or cancellation therefore remains authoritative.
Champion Service maps invalid IDs to `InvalidArgument` and missing IDs to
`NotFound`; transport and deadline failures remain available as `Unavailable`
and `DeadlineExceeded`.

Champion metadata is not required for room persistence. If an RPC fails while
the parent synchronization context is still valid, `RoomService` logs the
failure and stores an `Unknown` champion with an empty image URL. If the parent
context is cancelled, synchronization stops without partially replacing room
players.

Champion Service registers the standard gRPC Health service. It reports
`SERVING` only after Data Dragon has loaded. Shutdown first changes health to
`NOT_SERVING`, then calls `GracefulStop`, with a bounded fallback to `Stop`.

Unary client and server interceptors log the RPC method, gRPC status code and
duration. Champion IDs and other high-cardinality request values are excluded
from persistent log fields and future metric labels.

The current local and Docker network uses plaintext transport. TLS is required
before exposing this RPC outside a trusted network.

## Consequences

Advantages:

- a small, versioned and language-neutral service contract;
- generated client and server types instead of handwritten transport DTOs;
- independent ownership and deployment of Data Dragon loading;
- explicit status codes, deadlines and cancellation propagation;
- connection reuse and HTTP/2 multiplexing;
- graceful degradation when non-critical metadata is unavailable;
- standard health checking and graceful shutdown.

Trade-offs:

- local development now runs a second Go process;
- champion lookup includes a network hop;
- service discovery and startup ordering must be configured;
- plaintext transport is currently limited to local or container networking;
- the generated protobuf files must stay synchronized with the `.proto` source.

## Verification

The implementation is covered by:

- catalog lookup, fallback, ordering and copy-isolation unit tests;
- direct server tests for protobuf mapping and gRPC status codes;
- in-memory gRPC transport tests using `bufconn`;
- client deadline and cancellation tests;
- connection reuse verification across multiple unary RPCs;
- `RoomService` fallback and parent-cancellation tests;
- manual TCP verification with `grpcurl`;
- race detector and `go vet` checks.

## Future work

- TLS or service-mesh transport security outside local environments;
- optional client-side cache if per-player RPC latency becomes material;
- Prometheus metrics for RPC latency and status codes;
- OpenTelemetry propagation across HTTP and gRPC boundaries.
