# Architecture

FinLink is a mock-bank financial data synchronization platform built around an asynchronous Go API and worker.

## Components

| Component | Responsibility |
| --- | --- |
| React + TypeScript dashboard | Mock institution connection, sync status, account/transaction exploration, spending analytics |
| Go HTTP API | Connection and sync-run endpoints, financial-data reads, Kafka event publishing |
| Go sync worker | Consume sync requests, normalize provider records, persist data, update sync state |
| PostgreSQL | Durable connections, sync runs, canonical accounts and transactions |
| Apache Kafka | Sync request queue and dead-letter handling |
| Redis | Local infrastructure service; not on the currently verified core synchronization path |
| Northstar and Harbor | Deterministic mock provider adapters; no real bank access |

## Synchronization flow

1. The dashboard creates a mock-bank connection through the Go API.
2. A sync request is recorded with an idempotency key and published to `finlink.sync.requested`.
3. The Kafka worker consumes the request and invokes the matching mock provider adapter.
4. Provider-specific accounts and transactions are normalized into canonical records.
5. PostgreSQL persists account updates and deduplicated transactions; the incremental cursor advances only after successful persistence.
6. The worker records completion or failure. The dashboard polls sync status and refreshes financial data on success.

The worker retries failed processing with bounded backoff. Messages that exhaust retries, or cannot be decoded, are written to `finlink.sync.dead-letter` before their source offsets are committed. This allows failed messages to be inspected without silently discarding them.

## Data correctness

- Idempotency keys prevent repeated requests from creating redundant sync runs.
- Account upserts and transaction deduplication make replay safe.
- Incremental cursors advance only after successful data persistence.
- Transaction reads use bounded offset pagination.
- The dashboard checks transaction page metadata, identities, amounts, and timestamps before trusting spending totals.
- Spending analytics aggregate normalized transactions by month, category, and merchant.

## Local execution and verification

Docker Compose runs PostgreSQL, Kafka, and Redis. The Go API, Go worker, and Vite dashboard run as separate local processes. See [README.md](README.md) for the full setup commands.

GitHub Actions runs Go formatting, unit tests and vet, React typechecking/tests/build, and a Docker integration job. The integration job starts infrastructure, applies migrations, creates Kafka topics, runs both Go processes, and exercises the Northstar and Harbor mock-bank flows.

## Scope and limitations

FinLink is a **local mock-data portfolio application**, not a production banking connector. It has no real institution integrations, customer authentication, or hardened public deployment configuration. Local Compose credentials are development examples only. Redis is included in the stack but should not be described as essential to the verified synchronization path. No real financial credentials or customer data are required.
