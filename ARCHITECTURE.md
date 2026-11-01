# FinLink Architecture

FinLink is an event-driven financial data connectivity platform. It presents one normalized API over multiple financial institution providers while isolating provider-specific formats behind adapters.

## Initial data flow

1. A client creates a financial institution connection.
2. The API publishes a `sync.requested` event to Kafka.
3. A provider-specific sync worker fetches account and transaction data.
4. Raw provider payloads are normalized into FinLink's canonical models.
5. Idempotency and deduplication prevent duplicate writes.
6. PostgreSQL stores connections, accounts, transactions, and sync state.
7. Redis supports ephemeral state, caching, and rate limiting.
8. Downstream events trigger webhook delivery and future reconciliation workflows.

## Core principles

- Provider isolation through adapters
- Event-driven asynchronous processing
- Idempotent consumers and retry-safe APIs
- Financial data correctness over raw throughput
- Explicit observability for syncs and failures
- Deterministic, testable normalization

## Planned Kafka topics

- `connection.created`
- `sync.requested`
- `transactions.ingested`
- `transactions.normalized`
- `webhook.delivery.requested`
- `webhook.delivery.failed`
