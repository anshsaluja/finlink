# FinLink Roadmap

## Milestone 1 — Foundation
- [x] Create `dev` branch
- [x] Add Go API skeleton and health endpoint
- [x] Add local PostgreSQL, Redis, and Kafka infrastructure
- [ ] Add configuration package and structured logging
- [ ] Add database migrations and connection models

## Milestone 2 — Provider connectivity
- [ ] Define provider adapter interface
- [ ] Implement first mock bank provider
- [ ] Implement second mock bank with a different payload shape
- [ ] Add connection creation and sync request API

## Milestone 3 — Event-driven sync
- [ ] Publish `sync.requested` events
- [ ] Add Kafka sync consumer
- [ ] Persist raw sync metadata
- [ ] Add retry and dead-letter handling

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
