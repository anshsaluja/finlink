package syncs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SQLRepository struct{ db *sql.DB }
func NewSQLRepository(db *sql.DB) *SQLRepository { return &SQLRepository{db:db} }
func (r *SQLRepository) Create(ctx context.Context, run Run) (Run,error) {
	err:=r.db.QueryRowContext(ctx,`INSERT INTO sync_runs (connection_id,idempotency_key,requested_at,status,cursor_before) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key=EXCLUDED.idempotency_key RETURNING id,connection_id,idempotency_key,requested_at,started_at,completed_at,status,cursor_before,cursor_after,account_count,transaction_count,COALESCE(error_message,'')`,run.ConnectionID,run.IdempotencyKey,run.RequestedAt,run.Status,run.CursorBefore).Scan(&run.ID,&run.ConnectionID,&run.IdempotencyKey,&run.RequestedAt,&run.StartedAt,&run.CompletedAt,&run.Status,&run.CursorBefore,&run.CursorAfter,&run.AccountCount,&run.TransactionCount,&run.ErrorMessage)
	if err!=nil{return Run{},fmt.Errorf("insert sync run: %w",err)}; return run,nil
}
func (r *SQLRepository) Get(ctx context.Context,id int64)(Run,error){ var run Run; err:=r.db.QueryRowContext(ctx,`SELECT id,connection_id,idempotency_key,requested_at,started_at,completed_at,status,cursor_before,cursor_after,account_count,transaction_count,COALESCE(error_message,'') FROM sync_runs WHERE id=$1`,id).Scan(&run.ID,&run.ConnectionID,&run.IdempotencyKey,&run.RequestedAt,&run.StartedAt,&run.CompletedAt,&run.Status,&run.CursorBefore,&run.CursorAfter,&run.AccountCount,&run.TransactionCount,&run.ErrorMessage); if errors.Is(err,sql.ErrNoRows){return Run{},ErrRunNotFound}; if err!=nil{return Run{},fmt.Errorf("get sync run: %w",err)}; return run,nil }
func (r *SQLRepository) MarkStarted(ctx context.Context,id int64,at time.Time) error { return r.update(ctx,`UPDATE sync_runs SET status='processing',started_at=$2,error_message=NULL WHERE id=$1`,id,at.UTC()) }
func (r *SQLRepository) MarkCompleted(ctx context.Context,id int64,cursor string,accounts,transactions int,at time.Time) error { return r.update(ctx,`UPDATE sync_runs SET status='succeeded',completed_at=$2,cursor_after=$3,account_count=$4,transaction_count=$5,error_message=NULL WHERE id=$1`,id,at.UTC(),cursor,accounts,transactions) }
func (r *SQLRepository) MarkFailed(ctx context.Context,id int64,message string,at time.Time) error { return r.update(ctx,`UPDATE sync_runs SET status='failed',completed_at=$2,error_message=$3 WHERE id=$1`,id,at.UTC(),message) }
func (r *SQLRepository) update(ctx context.Context,q string,args ...any) error { result,err:=r.db.ExecContext(ctx,q,args...); if err!=nil{return fmt.Errorf("update sync run: %w",err)}; rows,err:=result.RowsAffected(); if err!=nil{return fmt.Errorf("read sync run update result: %w",err)}; if rows==0{return ErrRunNotFound}; return nil }
