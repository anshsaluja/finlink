BEGIN;

CREATE TABLE connections (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    institution TEXT NOT NULL,
    external_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('connected', 'healthy', 'syncing', 'error', 'disconnected')),
    cursor TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_synced_at TIMESTAMPTZ,
    last_error TEXT,
    UNIQUE (provider, external_id)
);

CREATE INDEX connections_status_idx ON connections (status);
CREATE INDEX connections_provider_idx ON connections (provider);

CREATE TABLE sync_runs (
    id BIGSERIAL PRIMARY KEY,
    connection_id TEXT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    idempotency_key TEXT NOT NULL UNIQUE,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('requested', 'processing', 'succeeded', 'failed')),
    cursor_before TEXT NOT NULL DEFAULT '',
    cursor_after TEXT NOT NULL DEFAULT '',
    account_count INTEGER NOT NULL DEFAULT 0 CHECK (account_count >= 0),
    transaction_count INTEGER NOT NULL DEFAULT 0 CHECK (transaction_count >= 0),
    error_message TEXT
);

CREATE INDEX sync_runs_connection_requested_idx ON sync_runs (connection_id, requested_at DESC);
CREATE INDEX sync_runs_status_idx ON sync_runs (status);

COMMIT;
