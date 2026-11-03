# FinLink Roadmap

## Milestone 1 — Foundation
- [x] Create `dev` branch
- [x] Add Go API skeleton and health endpoint
- [x] Add local PostgreSQL, Redis, and Kafka infrastructure
- [x] Add configuration package
- [x] Add database migrations and connection/sync-run models
- [ ] Add structured logging

## Milestone 2 — Provider connectivity
- [x] Define provider adapter interface
- [x] Implement first mock bank provider
- [x] Implement second mock bank with a different credential and payload shape
- [x] Add connection creation and sync request API
- [x] Add repository boundary and PostgreSQL connection repository
- [x] Wire PostgreSQL driver and runtime database lifecycle

## Milestone 3 — Event-driven sync
- [x] Add durable sync-run repository and `sync.requested` event contract
- [x] Route HTTP sync requests through the durable sync-run boundary
- [x] Add sync-run status API and request idempotency coverage
- [x] Publish `sync.requested` events to Kafka
- [x] Add Kafka sync consumer and worker execution path
- [x] Persist sync lifecycle metadata and result counts
- [x] Add bounded retries and dead-letter handling

The worker consumes `sync.requested` with an explicit consumer group and commits Kafka offsets only after successful processing or confirmed DLQ publication. Processing uses a configurable bounded retry budget with backoff. Exhausted and malformed messages are published to a dedicated dead-letter topic with source coordinates, attempt count, and failure metadata before their source offsets are committed.

## Milestone 4 — Canonical financial data
- [x] Define and normalize canonical accounts
- [x] Define and normalize canonical transactions
- [x] Add PostgreSQL account/transaction schema and idempotent repository writes
- [x] Add transaction deduplication constraints and deterministic fallback fingerprints
- [x] Integrate canonical persistence into Kafka sync worker
- [x] Advance incremental sync cursors only after successful persistence

The production worker executes provider sync → canonical normalization → atomic page persistence → cursor advancement → sync-run completion. A failed canonical write never advances the connection cursor, while replay remains safe through idempotent account upserts and transaction deduplication.

## Milestone 5 — Delivery and reliability
- [ ] Add webhooks
- [ ] Add webhook retries and signatures
- [ ] Add rate limiting
- [ ] Add connection health monitoring
- [ ] Add audit logging

## Milestone 6 — Product surface
- [ ] Add React + TypeScript developer dashboard
- [ ] Add connection and sync status views
- [ ] Add transaction explorer

## Milestone 7 — Production engineering
- [ ] Add unit and integration test coverage
- [ ] Add CI
- [ ] Add metrics and tracing
- [ ] Add load testing
- [ ] Add deployment configuration
