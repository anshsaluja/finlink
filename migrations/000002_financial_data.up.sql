CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    connection_id TEXT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    provider_external_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    currency TEXT NOT NULL CHECK (char_length(currency) = 3),
    balance_cents BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    UNIQUE (connection_id, provider_external_id)
);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,
    connection_id TEXT NOT NULL REFERENCES connections(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider_external_id TEXT,
    description TEXT NOT NULL,
    amount_cents BIGINT NOT NULL,
    currency TEXT NOT NULL CHECK (char_length(currency) = 3),
    posted_at TIMESTAMPTZ NOT NULL,
    fingerprint TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (connection_id, provider_external_id),
    UNIQUE (connection_id, fingerprint)
);

CREATE INDEX idx_accounts_connection_id ON accounts(connection_id);
CREATE INDEX idx_transactions_account_posted_at ON transactions(account_id, posted_at DESC);
CREATE INDEX idx_transactions_connection_posted_at ON transactions(connection_id, posted_at DESC);
