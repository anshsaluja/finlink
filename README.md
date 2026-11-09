# FinLink

FinLink is a full-stack, event-driven financial data synchronization application. It connects to two **simulated banks** (Northstar and Harbor), normalizes account and transaction records, and presents balances, transaction history, and spending analytics in a React dashboard.

**Sandbox only:** FinLink does not connect to real banks or process real financial credentials.

## Features

- Connect to either mock institution and trigger asynchronous synchronization.
- Track sync status and view normalized accounts and transactions.
- Browse paginated transaction history and explore spending by month, category, and merchant.
- Retry failed dashboard requests and restore a previously selected connection.
- Deduplicate transactions, support idempotent sync requests, and preserve incremental sync cursors.
- Retry worker failures and route exhausted messages to a Kafka dead-letter topic.

## Architecture

```text
React + TypeScript (Vite)
          |
       Go REST API ---- PostgreSQL
          |
    Kafka sync.requested
          |
      Go worker -------- PostgreSQL
          |
   Northstar / Harbor
     mock adapters

Redis: available in the local infrastructure stack
```

The API creates durable sync runs and publishes Kafka events. The worker reads those events, fetches mock provider records, normalizes and persists them, and updates the run status. PostgreSQL stores connections, accounts, transactions, and sync state. Kafka decouples HTTP requests from processing. Redis is provisioned for local infrastructure; it is **not required for the currently verified sync flow**.

See [ARCHITECTURE.md](ARCHITECTURE.md) for implementation details.

## Technology

Go, PostgreSQL, Apache Kafka, Redis, Docker Compose, React, TypeScript, and Vite. GitHub Actions runs backend checks, frontend tests/builds, and Docker-based mock-bank integration tests.

## Run locally

**Prerequisites:** Docker Desktop (with Compose), Go (version specified in `go.mod`), Node.js 22+, npm, Python 3, and curl. Run the commands from the repository root unless otherwise noted.

1. Start the local infrastructure:

   ```bash
   docker compose up -d
   ```

2. Wait until PostgreSQL and Kafka are ready:

   ```bash
   docker compose exec -T postgres pg_isready -U finlink -d finlink
   docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list
   ```

   If either command fails, wait a few seconds and retry.

3. Apply the database migrations **in filename order** and create the Kafka topics:

   ```bash
   for migration in migrations/*.up.sql; do
     docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U finlink -d finlink < "$migration" || exit 1
   done

   docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic finlink.sync.requested --partitions 3 --replication-factor 1
   docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server localhost:9092 --create --if-not-exists --topic finlink.sync.dead-letter --partitions 1 --replication-factor 1
   ```

4. Start the API and worker in **separate terminals**:

   ```bash
   go run ./cmd/api
   ```

   ```bash
   go run ./cmd/worker
   ```

5. Start the dashboard in a third terminal:

   ```bash
   cd web
   npm install
   npm run dev
   ```

   Open the local URL printed by Vite (typically `http://localhost:5173`). The dashboard defaults to `http://localhost:8080` for the API; set `VITE_API_BASE_URL` before starting Vite if using a different API address.

6. Confirm the API is healthy:

   ```bash
   curl http://localhost:8080/health
   ```

Select Northstar or Harbor in the dashboard, connect using the mock flow, start a sync, and inspect the resulting accounts, transactions, and analytics.

**Note:** The Compose file runs PostgreSQL, Kafka, and Redis; the Go API, worker, and Vite dashboard run separately on your machine.

## Tests

```bash
go test ./...
go vet ./...
cd web && npm install && npm run typecheck && npm test && npm run build
```

For a full backend integration smoke test, first start the infrastructure, apply migrations, create Kafka topics, and run both Go processes as above. Then, from the repository root:

```bash
python3 scripts/smoke_mock_banks.py
```

The smoke test creates isolated **synthetic** connections for both mock institutions, waits for synchronization, and validates accounts, transaction pagination, and stable transaction identities. GitHub Actions executes this flow against Docker infrastructure on `dev`.

## Configuration and safety

The backend defaults to local-only development values: PostgreSQL at `localhost:5432`, Kafka at `localhost:9092`, and HTTP on port `8080`. Supported backend settings include `FINLINK_HTTP_ADDR`, `FINLINK_DATABASE_URL`, `FINLINK_KAFKA_BROKERS`, `FINLINK_KAFKA_SYNC_TOPIC`, `FINLINK_KAFKA_DLQ_TOPIC`, `FINLINK_KAFKA_CONSUMER_GROUP`, `FINLINK_SYNC_MAX_ATTEMPTS`, and `FINLINK_SYNC_RETRY_BACKOFF`.

The Compose configuration uses example local database credentials and exposes development ports. **Do not expose this stack directly to the public internet.** It is a mock-data engineering demo, not a production banking integration or an authenticated multi-tenant service.

## Troubleshooting

- **API cannot connect to PostgreSQL:** Confirm `docker compose ps` shows a healthy `postgres` service and that migrations ran successfully.
- **Sync stays queued:** Confirm the worker is running and the Kafka topics exist. Inspect the API and worker terminal logs.
- **Dashboard cannot reach API:** Check `curl http://localhost:8080/health` and the dashboard's `VITE_API_BASE_URL`.
- **Start fresh (deletes local database data):** `docker compose down -v`, then repeat setup. Do not run this if you need to preserve your local data.

## Project status

The application and automated mock-bank integration tests are implemented and passing on the `dev` branch. Manual visual verification on a developer machine is a separate acceptance step. No public deployment is required to run or evaluate the project.
