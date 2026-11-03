package syncs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type SQLRepository struct{ db *sql.DB }

func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

func (r *SQLRepository) Create(ctx context.Context, run Run) (Run, error) {
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO sync_runs (connection_id, idempotency_key, requested_at, status, cursor_before)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		RETURNING id, connection_id, idempotency_key, requested_at, started_at, completed_at, status,
		          cursor_before, cursor_after, account_count, transaction_count, COALESCE(error_message, '')`,
		run.ConnectionID, run.IdempotencyKey, run.RequestedAt, run.Status, run.CursorBefore,
	).Scan(&run.ID, &run.ConnectionID, &run.IdempotencyKey, &run.RequestedAt, &run.StartedAt, &run.CompletedAt,
		&run.Status, &run.CursorBefore, &run.CursorAfter, &run.AccountCount, &run.TransactionCount, &run.ErrorMessage)
	if err != nil {
		return Run{}, fmt.Errorf("insert sync run: %w", err)
	}
	return run, nil
}

func (r *SQLRepository) Get(ctx context.Context, id int64) (Run, error) {
	var run Run
	err := r.db.QueryRowContext(ctx, `
		SELECT id, connection_id, idempotency_key, requested_at, started_at, completed_at, status,
		       cursor_before, cursor_after, account_count, transaction_count, COALESCE(error_message, '')
		FROM sync_runs WHERE id = $1`, id).Scan(&run.ID, &run.ConnectionID, &run.IdempotencyKey, &run.RequestedAt,
		&run.StartedAt, &run.CompletedAt, &run.Status, &run.CursorBefore, &run.CursorAfter, &run.AccountCount,
		&run.TransactionCount, &run.ErrorMessage)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound
	}
	if err != nil {
		return Run{}, fmt.Errorf("get sync run: %w", err)
	}
	return run, nil
}
