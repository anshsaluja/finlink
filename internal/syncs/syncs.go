package syncs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

var ErrRunNotFound = errors.New("sync run not found")

type Run struct {
	ID int64 `json:"id"`
	ConnectionID string `json:"connection_id"`
	IdempotencyKey string `json:"idempotency_key"`
	RequestedAt time.Time `json:"requested_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status string `json:"status"`
	CursorBefore string `json:"cursor_before"`
	CursorAfter string `json:"cursor_after"`
	AccountCount int `json:"account_count"`
	TransactionCount int `json:"transaction_count"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type RequestedEvent struct {
	RunID int64 `json:"run_id"`
	ConnectionID string `json:"connection_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Cursor string `json:"cursor"`
	RequestedAt time.Time `json:"requested_at"`
}

type Repository interface {
	Create(context.Context, Run) (Run, error)
	Get(context.Context, int64) (Run, error)
	MarkStarted(context.Context, int64, time.Time) error
	MarkCompleted(context.Context, int64, string, int, int, time.Time) error
	MarkFailed(context.Context, int64, string, time.Time) error
}

type Publisher interface { PublishSyncRequested(context.Context, RequestedEvent) error }
type ConnectionReader interface { SyncCursor(context.Context, string) (string, error) }

type Service struct { runs Repository; connections ConnectionReader; publisher Publisher }
func NewService(runs Repository, connections ConnectionReader, publisher Publisher) *Service { return &Service{runs:runs, connections:connections, publisher:publisher} }

func (s *Service) Request(ctx context.Context, connectionID, idempotencyKey string) (Run, error) {
	if idempotencyKey == "" { idempotencyKey = newKey() }
	cursor, err := s.connections.SyncCursor(ctx, connectionID); if err != nil { return Run{}, err }
	run, err := s.runs.Create(ctx, Run{ConnectionID:connectionID, IdempotencyKey:idempotencyKey, RequestedAt:time.Now().UTC(), Status:"requested", CursorBefore:cursor})
	if err != nil { return Run{}, fmt.Errorf("create sync run: %w", err) }
	event := RequestedEvent{RunID:run.ID, ConnectionID:connectionID, IdempotencyKey:run.IdempotencyKey, Cursor:run.CursorBefore, RequestedAt:run.RequestedAt}
	if err := s.publisher.PublishSyncRequested(ctx, event); err != nil { return Run{}, fmt.Errorf("publish sync.requested: %w", err) }
	return run, nil
}
func (s *Service) Get(ctx context.Context, id int64) (Run,error) { return s.runs.Get(ctx,id) }
func newKey() string { var b [16]byte; if _,err:=rand.Read(b[:]); err!=nil { return fmt.Sprintf("sync-%d",time.Now().UTC().UnixNano()) }; return hex.EncodeToString(b[:]) }
