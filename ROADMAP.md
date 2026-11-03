# FinLink Roadmap

## Milestone 1 — Foundation
- [x] Create `dev` branch
- [x] Add Go API skeleton and health endpoint
- [x] Add local PostgreSQL, Redis, and Kafka infrastructure
- [x] Add configuration package
- [x] Add database migrations and connection/sync-run models
- [ ] Add structured logging

The initial PostgreSQL schema models durable provider connections and sync runs, including incremental cursors, sync status, idempotency keys, counts, timestamps, and failure metadata. Configuration for the API, database, and Kafka is centralized under `internal/config`.

## Milestone 2 — Provider connectivity
- [x] Define provider adapter interface
- [x] Implement first mock bank provider
- [x] Implement second mock bank with a different credential and payload shape
- [x] Add connection creation and sync request API
- [x] Add repository boundary and PostgreSQL connection repository
- [ ] Wire PostgreSQL driver and runtime database lifecycle

Connection state now sits behind a repository interface instead of being owned by the service. A PostgreSQL implementation persists provider external IDs, status, cursors, and last-sync timestamps against the migration schema; the in-memory implementation remains available for isolated tests. Runtime database initialization is the remaining persistence wiring step.

## Milestone 3 — Event-driven sync
- [x] Add durable sync-run repository and `sync.requested` event contract
- [ ] Publish `sync.requested` events to Kafka
- [ ] Add Kafka sync consumer
- [ ] Persist raw sync metadata
- [ ] Add retry and dead-letter handling

Sync requests now have a dedicated service boundary that creates idempotent sync-run records before publishing a typed `sync.requested` event. Both in-memory and PostgreSQL repositories implement the same contract. The next product slice is Kafka producer wiring plus runtime PostgreSQL initialization, followed by a worker consumer that performs provider syncs asynchronously.

## Milestone 4 — Canonical financial data
- [ ] Normalize accounts
- [ ] Normalize transactions
- [ ] Add idempotent writes
- [ ] Add transaction deduplication
- [ ] Add incremental sync cursors

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
