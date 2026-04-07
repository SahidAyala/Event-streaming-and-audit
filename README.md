# Event Streaming & Audit Platform

## 🧠 Overview

A distributed event-driven platform for storing, replaying, and analyzing system events.

Supports event sourcing, audit logging, and time-travel debugging.

---

## 🎯 Problem Statement

In distributed systems, it's difficult to:

* Debug historical state
* Ensure audit compliance
* Reconstruct system behavior

This project provides:

* Immutable event logs
* Event replay
* Full audit traceability

---

## 🏗️ Architecture

* Event Producers
* Event Bus (Kafka)
* Event Store
* Query Layer
* Replay Engine

---

## ⚙️ Features

* Append-only event store
* Event replay (full & partial)
* Time-travel debugging
* Event versioning
* Snapshot support
* Audit logs

---

## 🛠️ Tech Stack

* Kafka / Redpanda
* PostgreSQL (event store)
* S3 (cold storage)
* Elasticsearch (querying)
* Node.js / Go

---

## 🔥 Challenges

* Event ordering
* Schema evolution
* Storage optimization
* Replay performance

---

## 📊 Metrics

* Events/sec
* Replay time
* Storage growth
* Query latency

---

## 🧪 Failure Scenarios

* Event duplication
* Partial writes
* Out-of-order delivery

---

## 🚀 Roadmap

* [ ] Event schema design
* [ ] Event ingestion pipeline
* [ ] Storage layer
* [ ] Replay engine
* [ ] Query API
* [ ] Observability

---

## 💬 Pitch

Implemented an event-driven audit platform enabling full traceability and replay of system events, improving debugging and compliance capabilities.
