# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A distributed event-driven platform for storing, replaying, and analyzing system events. Supports event sourcing, audit logging, and time-travel debugging. Currently in the blueprint/early-implementation phase.

## Commands

```bash
pnpm test          # Run unit tests (Vitest)
pnpm test:cov      # Run tests with coverage report
npx tsc            # TypeScript type check
```

Use `/commit`, `/test`, and `/pr` slash commands for guided workflows.

## Architecture

The platform is composed of five layers that flow in sequence:

```
Event Producers → Event Bus (Kafka/Redpanda) → Event Store (PostgreSQL, append-only)
                                                        ↓
                                          Query Layer (Elasticsearch)
                                                        ↓
                                             Replay Engine + Snapshots
```

**Storage tiers:** PostgreSQL for hot/active event storage, S3 for cold/archival storage.

**Key design constraints:**
- Event store is **append-only** and immutable — no updates or deletes
- Events must carry versioning metadata to support schema evolution
- Replay engine must handle duplicates, partial writes, and out-of-order delivery

## Source Structure

Code is organized by DDD bounded contexts:

```
src/contexts/<module>/
  application/      # Services, use cases (*.service.ts, *.service.spec.ts)
  domain/           # Aggregates, entities, domain events, value objects
  infrastructure/   # Repository implementations, Kafka adapters, Prisma
```

## Commit Conventions

Format: `type(scope): description`

- **scope** = module name derived from `src/contexts/<module>` (e.g., `events`, `replay`, `audit`)
- **types:** `feat`, `fix`, `refactor`, `test`, `chore`, `docs`
- Group changes by logical unit, not by file — max 5 commits per batch
- Never add `Co-Authored-By` in this repo

Examples:
```
feat(events): Add append-only event store schema
test(replay): Add unit tests for replay engine service
refactor(audit): Move immutability check into domain aggregate
```

## Testing

- Framework: **Vitest**
- Test files: `*.spec.ts` co-located with the source file they test
- Mock repositories, test business logic — happy path, validation errors, and edge cases
- Do not modify production code when writing or fixing tests
- Keep mocks consistent with DDD architecture (mock at the repository boundary)

## Pull Requests

PRs always target `main`. Before creating, ensure the branch is up-to-date with main. Branch naming convention: `type/ticket-description` (e.g., `feat/42-add-event-ingestion`).

PR title follows the same `type(scope): description` format as commits.

## Architect Agent

Use the `Architect` agent (invoked on request) for DDD code reviews. It reviews code for correct bounded context separation, aggregate design, and domain event modeling.
