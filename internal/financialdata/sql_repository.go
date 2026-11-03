package financialdata

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLRepository struct{ db *sql.DB }

func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

// Store atomically upserts accounts and transactions from one provider page. Replaying
// the same page is safe: provider IDs and deterministic fingerprints enforce deduplication.
func (r *SQLRepository) Store(ctx context.Context, accounts []Account, transactions []Transaction) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return fmt.Errorf("begin financial data transaction: %w", err) }
	defer tx.Rollback()

	accountIDs := make(map[string]int64, len(accounts))
	for _, account := range accounts {
		var id int64
		err = tx.QueryRowContext(ctx, `
			INSERT INTO accounts (connection_id, provider_external_id, name, type, currency, balance_cents, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (connection_id, provider_external_id) DO UPDATE SET
				name=EXCLUDED.name, type=EXCLUDED.type, currency=EXCLUDED.currency,
				balance_cents=EXCLUDED.balance_cents, updated_at=EXCLUDED.updated_at
			RETURNING id`, account.ConnectionID, account.ProviderExternalID, account.Name, account.Type,
			account.Currency, account.BalanceCents, account.UpdatedAt).Scan(&id)
		if err != nil { return fmt.Errorf("upsert account %q: %w", account.ProviderExternalID, err) }
		accountIDs[account.ProviderExternalID] = id
	}

	for _, transaction := range transactions {
		accountID, ok := accountIDs[transaction.AccountProviderExternalID]
		if !ok {
			err = tx.QueryRowContext(ctx, `SELECT id FROM accounts WHERE connection_id=$1 AND provider_external_id=$2`, transaction.ConnectionID, transaction.AccountProviderExternalID).Scan(&accountID)
			if err != nil { return fmt.Errorf("resolve transaction account %q: %w", transaction.AccountProviderExternalID, err) }
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO transactions (connection_id, account_id, provider_external_id, description, amount_cents, currency, posted_at, fingerprint)
			VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8)
			ON CONFLICT DO NOTHING`, transaction.ConnectionID, accountID,
			transaction.ProviderExternalID, transaction.Description, transaction.AmountCents, transaction.Currency,
			transaction.PostedAt, transaction.Fingerprint)
		if err != nil { return fmt.Errorf("insert transaction %q: %w", transaction.ProviderExternalID, err) }
	}
	if err := tx.Commit(); err != nil { return fmt.Errorf("commit financial data: %w", err) }
	return nil
}

// AccountsByConnection returns canonical accounts for the dashboard/read API.
func (r *SQLRepository) AccountsByConnection(ctx context.Context, connectionID string) ([]Account, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT connection_id, provider_external_id, name, type, currency, balance_cents, updated_at
		FROM accounts WHERE connection_id=$1 ORDER BY name, provider_external_id`, connectionID)
	if err != nil { return nil, fmt.Errorf("list accounts: %w", err) }
	defer rows.Close()
	accounts := make([]Account, 0)
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ConnectionID, &a.ProviderExternalID, &a.Name, &a.Type, &a.Currency, &a.BalanceCents, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate accounts: %w", err) }
	return accounts, nil
}

// TransactionsByConnection returns the most recent canonical transactions with a bounded
// limit so a dashboard request cannot accidentally scan an unbounded ledger.
func (r *SQLRepository) TransactionsByConnection(ctx context.Context, connectionID string, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 200 { limit = 100 }
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.connection_id, COALESCE(t.provider_external_id,''), a.provider_external_id,
		       t.description, t.amount_cents, t.currency, t.posted_at, t.fingerprint
		FROM transactions t JOIN accounts a ON a.id=t.account_id
		WHERE t.connection_id=$1 ORDER BY t.posted_at DESC, t.id DESC LIMIT $2`, connectionID, limit)
	if err != nil { return nil, fmt.Errorf("list transactions: %w", err) }
	defer rows.Close()
	transactions := make([]Transaction, 0)
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ConnectionID, &t.ProviderExternalID, &t.AccountProviderExternalID, &t.Description, &t.AmountCents, &t.Currency, &t.PostedAt, &t.Fingerprint); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil { return nil, fmt.Errorf("iterate transactions: %w", err) }
	return transactions, nil
}
