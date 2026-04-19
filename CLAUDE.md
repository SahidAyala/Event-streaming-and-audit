# Claude Context — Event Streaming & Audit Platform

## 🧠 System Purpose

A distributed event streaming and audit platform designed to:

* Capture events reliably (append-only)
* Provide a consistent audit trail
* Enable event replay for debugging and recovery
* Support scalable read models via indexing

This system prioritizes **correctness, durability, and traceability over convenience**.

---

## 🧱 Architecture

* Hexagonal Architecture (Ports & Adapters)

### Layers

* **Domain**

  * Pure business logic
  * No external dependencies
  * Defines:

    * Event entity (immutable)
    * Repository interfaces (Store, Publisher, Indexer)

* **Application**

  * Use cases:

    * Ingest events
    * Consume events
    * Query events (to be implemented)
    * Replay events (to be implemented)

* **Infrastructure**

  * PostgreSQL → Event Store
  * Kafka → Event Bus
  * Elasticsearch → Read Model
  * HTTP → Ingest API
  * gRPC → Replay API (planned)

---

## 📊 Data Flow (Source of Truth Model)

1. HTTP POST /events
2. Application layer validates event
3. Event is appended to PostgreSQL (source of truth)
4. Event is published to Kafka (async)
5. Consumer reads from Kafka
6. Event is indexed into Elasticsearch

### Important

* PostgreSQL is the **single source of truth**
* Kafka is **not authoritative**
* Elasticsearch is **eventually consistent**

---

## ⚙️ System Guarantees

* Append-only event store
* Per-stream versioning (`stream_id`, `version`)
* At-least-once delivery (Kafka consumer)
* Idempotent indexing (Elasticsearch)
* Immutable event data
* No in-place updates

---

## 🚫 Non-Goals (Current Scope)

* No exactly-once delivery
* No distributed transactions
* No strong consistency across services
* No schema registry (yet)
* No multi-region support

---

## ⚠️ Current Limitations

* No Query API (GET /events/{streamID})
* No Replay Engine
* No Snapshots
* No DLQ (dead-letter queue) for Kafka
* No time-travel debugging
* No S3/MinIO archival
* No test coverage

---

## 🎯 Design Principles

* Favor **durability over latency**
* Favor **simplicity over premature optimization**
* Prefer **idempotent operations**
* Keep domain layer isolated
* Avoid leaking infrastructure concerns into application logic
* Treat event ordering as critical per stream

---

## 🔐 Data Integrity Rules

* Version must be strictly increasing per stream
* Writes must be atomic (PostgreSQL transaction)
* Consumers must tolerate duplicate events
* Indexing must be idempotent
* Replay must always read from PostgreSQL

---

## 🧩 Key Concepts

* **Event**

  * Immutable record
  * Contains:

    * id
    * stream_id
    * version
    * type
    * payload
    * timestamp

* **Stream**

  * Logical grouping of events
  * Ordering guaranteed per stream only

* **Read Model**

  * Derived projection stored in Elasticsearch
  * Can be rebuilt from event store

---

## 🛠️ Implementation Rules (STRICT)

* Do NOT bypass application layer
* Do NOT query PostgreSQL directly from HTTP layer
* Do NOT treat Kafka as source of truth
* Do NOT mutate events after persistence
* Do NOT introduce shared state outside DB

---

## 📦 Current Components Status

| Component              | Status       |
| ---------------------- | ------------ |
| Event Store (Postgres) | ✅ Complete   |
| Kafka Producer         | ✅ Complete   |
| Kafka Consumer         | ✅ Functional |
| Elasticsearch Indexer  | ⚠️ Partial   |
| HTTP Ingest API        | ✅ Complete   |
| Query API              | ❌ Missing    |
| Replay Engine          | ❌ Missing    |
| Tests                  | ❌ Missing    |

---

## 🚀 MVP Roadmap (Execution Order)

1. Fix build + dependencies ✅ (done)
2. Query API (GET /events/{streamID})
3. Basic Replay Engine (gRPC)
4. Add test coverage (core services)
5. Kafka DLQ + retry strategy
6. Observability (logs + metrics)
7. S3/MinIO archival
8. Snapshots
9. Time-travel debugging

---

## ⚡ Performance Expectations (Target)

* Ingest latency: < 50ms (without Kafka dependency)
* Kafka throughput: scalable horizontally
* Replay: linear per stream
* Indexing: eventual consistency (< seconds)

---

## 🧪 Testing Strategy (Planned)

* Unit tests for application layer
* Mock ports (Store, Publisher, Indexer)
* Integration tests with Postgres + Kafka (later)
* Replay correctness tests (ordering + idempotency)

---

## 🔍 Observability (Planned)

* Structured logging (slog)
* Correlation IDs per request
* Metrics:

  * ingest rate
  * consumer lag
  * indexing latency
  * replay duration

---

## ⚠️ Critical Risks

* Race conditions in Kafka consumer (no DLQ yet)
* Event duplication due to at-least-once delivery
* Elasticsearch inconsistency vs Postgres
* Missing replay → limits audit/debug capability

---

## 🧠 Guidance for AI (Claude)

When modifying or extending the system:

* Always respect hexagonal boundaries
* Never introduce tight coupling between layers
* Prioritize correctness over performance
* Ask before introducing new infrastructure
* Prefer minimal, production-correct solutions

---

## 📌 Definition of Done (MVP)

The system is considered MVP-complete when:

* Events can be ingested reliably
* Events can be queried by stream (ordered)
* Events can be replayed from Postgres
* System handles duplicates safely
* Basic test coverage exists

---
