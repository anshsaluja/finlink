package syncs

import (
	"context"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu     sync.RWMutex
	nextID int64
	runs   map[int64]Run
	keys   map[string]int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{nextID: 1, runs: map[int64]Run{}, keys: map[string]int64{}}
}
func (r *MemoryRepository) Create(_ context.Context, run Run) (Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.keys[run.IdempotencyKey]; ok {
		return r.runs[id], nil
	}
	run.ID = r.nextID
	r.nextID++
	r.runs[run.ID] = run
	r.keys[run.IdempotencyKey] = run.ID
	return run, nil
}
func (r *MemoryRepository) Get(_ context.Context, id int64) (Run, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return Run{}, ErrRunNotFound
	}
	return run, nil
}
func (r *MemoryRepository) MarkStarted(_ context.Context, id int64, at time.Time) error {
	return r.change(id, func(run *Run) { run.Status = "processing"; at = at.UTC(); run.StartedAt = &at; run.ErrorMessage = "" })
}
func (r *MemoryRepository) MarkCompleted(_ context.Context, id int64, cursor string, accounts, transactions int, at time.Time) error {
	return r.change(id, func(run *Run) {
		run.Status = "succeeded"
		at = at.UTC()
		run.CompletedAt = &at
		run.CursorAfter = cursor
		run.AccountCount = accounts
		run.TransactionCount = transactions
		run.ErrorMessage = ""
	})
}
func (r *MemoryRepository) MarkFailed(_ context.Context, id int64, message string, at time.Time) error {
	return r.change(id, func(run *Run) {
		run.Status = "failed"
		at = at.UTC()
		run.CompletedAt = &at
		run.ErrorMessage = message
	})
}
func (r *MemoryRepository) change(id int64, fn func(*Run)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.runs[id]
	if !ok {
		return ErrRunNotFound
	}
	fn(&run)
	r.runs[id] = run
	return nil
}

type MemoryPublisher struct {
	mu     sync.Mutex
	Events []RequestedEvent
}

func (p *MemoryPublisher) PublishSyncRequested(_ context.Context, event RequestedEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Events = append(p.Events, event)
	return nil
}
