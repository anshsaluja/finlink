# FinLink

FinLink is an event-driven financial data connectivity platform that aggregates, normalizes, and synchronizes account and transaction data across financial institutions.

## Current stack

- Go
- PostgreSQL
- Kafka
- Redis
- Docker Compose
- React + TypeScript (planned dashboard)

## Local development

Start infrastructure:

```bash
docker compose up -d
```

Run the API:

```bash
go run ./cmd/api
```

Health check:

```bash
curl http://localhost:8080/health
```

See `ARCHITECTURE.md` for the system design and `ROADMAP.md` for active milestones.
