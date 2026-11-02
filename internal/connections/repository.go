package connections

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Repository owns durable connection state. Implementations must preserve the
// provider external ID because it is required for subsequent incremental syncs.
type Repository interface {
	Create(context.Context, Connection) error
	Get(context.Context, string) (Connection, error)
	UpdateSyncState(context.Context, Connection) error
}

type SQLRepository struct {
	db *sql.DB
}

func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db: db} }

func (r *SQLRepository) Create(ctx context.Context, c Connection) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO connections (id, provider, institution, external_id, status, cursor, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		c.ID, c.Provider, c.Institution, c.externalID, c.Status, c.Cursor, c.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert connection: %w", err)
	}
	return nil
}

func (r *SQLRepository) Get(ctx context.Context, id string) (Connection, error) {
	var c Connection
	err := r.db.QueryRowContext(ctx, `
		SELECT id, provider, institution, status, created_at, last_synced_at, cursor, external_id
		FROM connections WHERE id = $1`, id).Scan(
		&c.ID, &c.Provider, &c.Institution, &c.Status, &c.CreatedAt, &c.LastSyncedAt, &c.Cursor, &c.externalID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrConnectionNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("get connection: %w", err)
	}
	return c, nil
}

func (r *SQLRepository) UpdateSyncState(ctx context.Context, c Connection) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE connections
		SET status = $2, cursor = $3, last_synced_at = $4, updated_at = NOW(), last_error = NULL
		WHERE id = $1`, c.ID, c.Status, c.Cursor, c.LastSyncedAt)
	if err != nil {
		return fmt.Errorf("update connection sync state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read update result: %w", err)
	}
	if rows == 0 {
		return ErrConnectionNotFound
	}
	return nil
}
