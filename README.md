# Event Streaming & Audit Platform

A distributed event streaming platform built in Go with hexagonal architecture.
Events are stored append-only in PostgreSQL (source of truth), streamed via Kafka, and indexed into Elasticsearch for queries. Replay reads directly from PostgreSQL, guaranteeing integrity even if Kafka or Elasticsearch are stale.

---

## Architecture

```
HTTP POST /events
      │
      ▼
 ingest-api  ──► PostgreSQL (source of truth)
      │
      └──► Kafka  (best-effort, async publish)
                │
                ▼
       consumer-service
                │
          ┌─────┴─────┐
          ▼           ▼
  Elasticsearch    DLQ topic
  (read model)   (failed index)

HTTP GET /events/:streamID ──► Elasticsearch (eventually consistent)
gRPC Replay ─────────────────► PostgreSQL   (strongly consistent)
```

### Services

| Binary | Responsibility |
|---|---|
| `cmd/ingest-api` | HTTP API — ingest events, serve read model |
| `cmd/consumer-service` | Kafka consumer — index into Elasticsearch, route failures to DLQ |
| `cmd/replay-service` | gRPC server — replay ordered events from PostgreSQL |

---

## Prerequisites

- **Go 1.24+**
- **Docker + Docker Compose**
- `pg_isready`, `nc`, `curl` available in your PATH (standard on macOS/Linux)

---

## Getting Started

There are two modes depending on what you're doing:

| Mode | Command | Use when |
|---|---|---|
| **Dev** | `make dev` | Writing code — fast iteration, `go run` locally |
| **Full stack** | `make dev-full` | Integration testing, demos, or production-like validation |

---

### Mode 1 — Dev (recommended for day-to-day coding)

Infrastructure runs in Docker. Application services run locally with `go run` so you get fast reloads without rebuilding images.

**Terminal 1 — API:**

```bash
make dev
```

This starts Postgres, Kafka, Elasticsearch, and MinIO in Docker, waits for all healthchecks to pass, then launches `ingest-api` locally on `:8080`.

**Terminal 2 — Consumer** (needed for `GET /events` to return results):

```bash
make dev-consumer
```

Starts the same infra (no-op if already running) and launches `consumer-service` locally.

**Apply migrations** (first time, or after adding new migration files):

```bash
make migrate
```

> Migrations run against `localhost:5433` by default. Only pending files are applied — already-applied ones are skipped.

**Verify:**

```bash
make test-e2e
```

---

### Mode 2 — Full stack (production-like)

Everything runs in Docker — infrastructure and application services. No local Go needed after `make dev-full`.

```bash
make dev-full
```

Builds the Go binaries inside Docker and starts all services with live logs in the foreground. Services start in dependency order via healthcheck-gated `depends_on`.

To run in the background instead:

```bash
make up        # detached
make logs      # follow logs
make down      # stop everything
```

Ports exposed on the host:

| Service | Port | Notes |
|---|---|---|
| `ingest-api` | `8080` | HTTP API + Swagger UI |
| `postgres` | `5433` | 5432 is reserved for a local Postgres instance |
| `kafka` | `9094` | External listener for local tooling |
| `elasticsearch` | `9200` | REST API |
| `minio` | `9000` / `9001` | S3 API / Console UI |

> **Why port 5433?** Docker maps PostgreSQL to `5433` to avoid conflicts with a local Postgres on `5432`. Inside the Docker network services always connect to `postgres:5432`.

---

## Database Migrations

Migrations live in `db/migrations/` as numbered SQL files (`001_create_events.sql`, `002_...sql`, …). They are embedded into the `migrate` binary at build time.

```bash
# Apply all pending migrations
make migrate

# Against a custom DSN
POSTGRES_DSN=postgres://user:pass@host:5432/db make migrate
```

The runner creates a `schema_migrations` table to track which files have been applied. Each migration runs in its own transaction — if it fails, nothing is committed.

To add a new migration, create the next numbered file:

```bash
touch db/migrations/002_add_my_column.sql
# edit the file, then:
make migrate
```

---

## Verify It Works

Once the API is running (either mode), run the end-to-end smoke test:

```bash
make test-e2e
```

This validates the full pipeline: **POST /events → PostgreSQL → Kafka → consumer-service → Elasticsearch → GET /events/{streamID}**.

Or manually with curl:

```bash
# 1. Ingest an event
curl -X POST http://localhost:8080/events \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "stream_id": "order:1",
    "type":      "order.created",
    "source":    "orders-svc",
    "payload":   {"amount": 99.99, "currency": "USD"}
  }'

# 2. Wait ~3s for the Kafka → Elasticsearch pipeline, then query
sleep 3
curl http://localhost:8080/events/order:1 \
  -H "X-API-Key: dev-api-key"
```

> **Eventual consistency:** `GET /events/{streamID}` reads from Elasticsearch. Recently ingested events may not appear immediately — the consumer processes them asynchronously via Kafka. Events are always durable in PostgreSQL the moment `POST /events` returns `201`.

---

## Swagger UI

Interactive API docs are served by `ingest-api` at:

```
http://localhost:8080/swagger/index.html
```

To authenticate in the UI:

1. Click **Authorize** (top right)
2. Enter `dev-api-key` in the **ApiKeyAuth** field
3. Click **Authorize** → **Close**

All `/events` endpoints are now unlocked. No separate container needed.

---

## Authentication

All `/events` endpoints require authentication. `/health` and `/swagger` are public.

### Simple mode (default — dev)

Pass the API key via the `X-API-Key` header:

```bash
curl -X POST http://localhost:8080/events \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{"stream_id":"order:1","type":"order.created","source":"orders-svc","payload":{}}'
```

Configured via `AUTH_API_KEY` env var (default: `dev-api-key`).

### JWT mode (production / multi-tenant)

Set in `.env` or as env vars:

```bash
AUTH_MODE=jwt
AUTH_JWT_SECRET=change-me-use-at-least-32-random-characters
```

**Bootstrap a new tenant** (one-time per tenant):

```bash
curl -X POST http://localhost:8080/tenants \
  -H "X-Admin-Key: admin-secret" \
  -H "Content-Type: application/json" \
  -d '{"tenant_id": "acme", "subject_id": "acme-admin"}'
# → {"token": "eyJ...", "expires_at": "..."}
```

Use the returned JWT for all subsequent requests:

```bash
curl -X POST http://localhost:8080/events \
  -H "Authorization: Bearer eyJ..." \
  -H "Content-Type: application/json" \
  -d '{"stream_id":"order:1","type":"order.created","source":"orders-svc","payload":{}}'
```

Validate credentials and discover tenant scope:

```bash
curl -X POST http://localhost:8080/auth/token \
  -H "Authorization: Bearer eyJ..."
# → {"subject_id": "acme-admin", "tenant_id": "acme", "roles": ["writer", "reader"]}
```

---

## API Reference

### POST /events — Ingest

```
POST /events
X-API-Key: dev-api-key
Content-Type: application/json

{
  "stream_id": "order:1",
  "type":      "order.created",
  "source":    "orders-svc",
  "payload":   {"amount": 99.99},
  "metadata":  {"region": "us-east-1", "trace_id": "abc-123"}
}
```

Response `201`:

```json
{
  "id":          "01906c2e-4a3b-7000-8000-abc123def456",
  "tenant_id":   "default",
  "stream_id":   "order:1",
  "type":        "order.created",
  "source":      "orders-svc",
  "version":     1,
  "occurred_at": "2026-04-26T10:00:00Z",
  "payload":     {"amount": 99.99}
}
```

- `version` is assigned atomically by the store (`MAX + 1` per stream).
- Kafka publish is best-effort — a Kafka failure does **not** fail this request.

### GET /events/{streamID} — Query (read model)

```
GET /events/order:1?limit=20&offset=0
X-API-Key: dev-api-key
```

Response `200`:

```json
{
  "stream_id":  "order:1",
  "events":     [...],
  "total":      3,
  "limit":      20,
  "offset":     0,
  "read_model": "elasticsearch"
}
```

Response headers:
- `X-Read-Model: elasticsearch`
- `X-Data-Consistency: eventual`

### GET /health

```
GET /health  →  200 {"status": "ok"}
```

No authentication required.

---

## Configuration

All configuration is via environment variables. Defaults work out of the box with `docker-compose.yml`.

```bash
cp .env.example .env  # then edit as needed
```

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDR` | `:8080` | HTTP listen address |
| `GRPC_ADDR` | `:50051` | gRPC listen address |
| `POSTGRES_DSN` | `postgres://events:events@localhost:5433/events?sslmode=disable` | PostgreSQL DSN |
| `KAFKA_BROKERS` | `localhost:9094` | Comma-separated broker addresses |
| `KAFKA_TOPIC` | `events` | Main events topic |
| `KAFKA_DLQ_TOPIC` | `events-dlq` | Dead-letter queue topic |
| `KAFKA_GROUP_ID` | `consumer-service` | Consumer group ID |
| `ELASTICSEARCH_ADDRS` | `http://localhost:9200` | Comma-separated ES node addresses |
| `ELASTICSEARCH_INDEX` | `events` | ES index name |
| `AUTH_MODE` | `simple` | `simple` (API key) or `jwt` (Bearer token) |
| `AUTH_API_KEY` | `dev-api-key` | Static API key for `simple` mode |
| `AUTH_JWT_SECRET` | _(empty)_ | HMAC secret for `jwt` mode — required when `AUTH_MODE=jwt` |
| `ADMIN_KEY` | `admin-secret` | Protects `POST /tenants` (tenant bootstrap) |

---

## Make Reference

```bash
# Dev workflow
make dev             # infra in Docker + ingest-api locally (fast iteration)
make dev-consumer    # infra in Docker + consumer-service locally
make wait-infra      # start infra only and wait for healthchecks

# Full stack
make dev-full        # everything in Docker, logs in foreground (production-like)
make up              # everything in Docker, detached
make down            # stop all containers
make logs            # tail all logs

# Database
make migrate         # apply pending SQL migrations

# Testing
make test            # unit tests
make test-e2e        # end-to-end smoke test (requires running stack)

# Build & tooling
make build           # compile all binaries → bin/
make lint            # golangci-lint
make tidy            # go mod tidy
make swag            # regenerate Swagger docs from annotations
make proto           # regenerate gRPC stubs from .proto files
```

---

## Data Guarantees

| Guarantee | Details |
|---|---|
| Append-only | Events are never updated or deleted |
| Per-stream ordering | `version` is strictly increasing per `stream_id` |
| Atomic writes | Each `POST /events` is a single PostgreSQL transaction |
| At-least-once delivery | Kafka consumer may receive duplicates |
| Idempotent indexing | ES uses event UUID as document ID — re-indexing is safe |
| DLQ routing | ES indexing failures are routed to `events-dlq`, never silently lost |
| Replay integrity | Replay always reads from PostgreSQL, not Kafka or Elasticsearch |

---

## Tech Stack

| Concern | Technology |
|---|---|
| Event store | PostgreSQL 16 (append-only, `UNIQUE` per `stream_id + version`) |
| Event bus | Kafka 3.7 (KRaft mode, no ZooKeeper) |
| Read model | Elasticsearch 8 |
| Cold storage | MinIO (S3-compatible) |
| Language | Go 1.24 |
| HTTP router | chi v5 |
| gRPC | google.golang.org/grpc |
| Auth | Static API key / HMAC-HS256 JWT |
| API docs | Swaggo (OpenAPI 2.0, Swagger UI embedded) |

---

## Status

| Component | Status |
|---|---|
| Event Store (Postgres) | Done |
| Kafka Producer | Done |
| Kafka Consumer + DLQ | Done |
| Elasticsearch Indexer | Done |
| HTTP Ingest + Query API | Done |
| Auth (simple + JWT) | Done |
| Tenant bootstrap (`POST /tenants`) | Done |
| Replay Engine (gRPC) | Done |
| Swagger UI | Done |
| Database migrations | Done |
| Unit tests | Done |
| E2E smoke test | Done |
| Snapshots | Pending |
| S3/MinIO archival | Pending |
| Observability (metrics) | Pending |
