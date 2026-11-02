package connections

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	connections map[string]Connection
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{connections: make(map[string]Connection)}
}

func (r *MemoryRepository) Create(_ context.Context, c Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[c.ID] = c
	return nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (Connection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.connections[id]
	if !ok {
		return Connection{}, ErrConnectionNotFound
	}
	return c, nil
}

func (r *MemoryRepository) UpdateSyncState(_ context.Context, c Connection) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.connections[c.ID]; !ok {
		return ErrConnectionNotFound
	}
	r.connections[c.ID] = c
	return nil
}
