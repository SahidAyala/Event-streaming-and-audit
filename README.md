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

## Quick Start

One command starts everything — infrastructure **and** application services:

```bash
git clone <repo> && cd event-streaming-and-audit
make up
```

Docker Compose builds the Go binaries, starts all six services in dependency order, and waits for each healthcheck to pass before the next service launches. No race conditions, no manual ordering, no extra terminals.

Services and their exposed ports:

| Service | Port | Notes |
|---|---|---|
| `ingest-api` | `8080` | HTTP API + Swagger UI |
| `postgres` | `5433` | Host port (5432 reserved for a local Postgres) |
| `kafka` | `9094` | External listener for local tooling |
| `elasticsearch` | `9200` | REST API |
| `minio` | `9000` / `9001` | S3 API / Console UI |

Follow logs:

```bash
make logs
```

Stop everything:

```bash
make down
```

> **Why port 5433?** Docker maps PostgreSQL to `5433` on the host to avoid conflicts if you already have a local PostgreSQL instance running on the default `5432`. Inside the Docker network, services always use `postgres:5432`.

---

## Verify It Works

Once `make up` completes, run the end-to-end smoke test:

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

> **Eventual consistency:** `GET /events/{streamID}` reads from Elasticsearch. Recently ingested events may not appear immediately — the consumer processes them asynchronously via Kafka. Events are always durable in PostgreSQL from the moment `POST /events` returns `201`.

---

## Swagger UI

Interactive API docs are served by the running `ingest-api`:

```
http://localhost:8080/swagger/index.html
```

To authenticate in the UI:

1. Click **Authorize** (top right)
2. Enter `dev-api-key` in the **ApiKeyAuth** field
3. Click **Authorize** → **Close**

All `/events` endpoints are now unlocked. No separate Swagger container needed.

---

## Authentication

All `/events` endpoints require authentication. `/health` and `/swagger` are public.

### Simple mode (default — local/dev)

Pass the API key in the `X-API-Key` header:

```bash
curl -X POST http://localhost:8080/events \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{"stream_id":"order:1","type":"order.created","source":"orders-svc","payload":{}}'
```

Configured via `AUTH_API_KEY` env var (default: `dev-api-key`).

### JWT mode (production / multi-tenant)

Set in `.env` or via env vars:

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
  "occurred_at": "2026-04-21T10:00:00Z",
  "payload":     {"amount": 99.99}
}
```

- `version` is assigned atomically by the store (`MAX + 1` per stream).
- Kafka publish is best-effort — a Kafka failure does **not** fail this request. The event is already durable in PostgreSQL.

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

Results are **eventually consistent** with PostgreSQL. The `total` field reflects indexed events, which may lag by a few seconds.

### GET /health

```
GET /health  →  200 {"status": "ok"}
```

No authentication required.

---

## Configuration

All configuration is via environment variables. The defaults match `docker-compose.yml` and work out of the box.

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
| `AUTH_JWT_SECRET` | _(empty)_ | HMAC secret for `jwt` mode (required if `AUTH_MODE=jwt`) |
| `ADMIN_KEY` | `admin-secret` | Protects `POST /tenants` (tenant bootstrap) |

Copy `.env.example` to `.env` to customise:

```bash
cp .env.example .env
```

---

## Local Development

Run services locally against the Docker infrastructure (infra-only via `make up`):

```bash
make run            # ingest-api on :8080
make run-consumer   # consumer-service
make run-replay     # replay-service on :50051
```

Other tasks:

```bash
make test       # Unit tests
make test-e2e   # End-to-end smoke test
make build      # Compile all binaries → bin/
make lint       # golangci-lint
make tidy       # go mod tidy
make swag       # Regenerate Swagger docs from annotations
make proto      # Regenerate gRPC stubs from .proto files
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
| Unit tests | Done |
| E2E smoke test | Done |
| Snapshots | Pending |
| S3/MinIO archival | Pending |
| Observability (metrics) | Pending |
