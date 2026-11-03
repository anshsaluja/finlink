package syncs

import (
	"context"
	"sync"
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
