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
			ON CONFLICT (connection_id, fingerprint) DO NOTHING`, transaction.ConnectionID, accountID,
			transaction.ProviderExternalID, transaction.Description, transaction.AmountCents, transaction.Currency,
			transaction.PostedAt, transaction.Fingerprint)
		if err != nil { return fmt.Errorf("insert transaction %q: %w", transaction.ProviderExternalID, err) }
	}
	if err := tx.Commit(); err != nil { return fmt.Errorf("commit financial data: %w", err) }
	return nil
}
