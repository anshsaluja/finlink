package connections

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/anshsaluja/finlink/internal/provider"
)

var ErrProviderNotFound = errors.New("provider not found")
var ErrConnectionNotFound = errors.New("connection not found")

type Connection struct {
	ID           string     `json:"id"`
	Provider     string     `json:"provider"`
	Institution  string     `json:"institution"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	Cursor       string     `json:"cursor,omitempty"`
	externalID   string
}

type SyncResult struct {
	Connection   Connection             `json:"connection"`
	Accounts     []provider.Account     `json:"accounts"`
	Transactions []provider.Transaction `json:"transactions"`
}

type Service struct {
	registry    *provider.Registry
	mu          sync.RWMutex
	connections map[string]Connection
}

func NewService(registry *provider.Registry) *Service {
	return &Service{registry: registry, connections: make(map[string]Connection)}
}

func (s *Service) Create(ctx context.Context, providerName string, credentials provider.Credentials) (Connection, error) {
	adapter, ok := s.registry.Get(providerName)
	if !ok {
		return Connection{}, ErrProviderNotFound
	}
	remote, err := adapter.Connect(ctx, credentials)
	if err != nil {
		return Connection{}, err
	}
	connection := Connection{ID: newID(), Provider: providerName, Institution: remote.Institution, Status: "connected", CreatedAt: time.Now().UTC(), externalID: remote.ExternalID}
	s.mu.Lock()
	s.connections[connection.ID] = connection
	s.mu.Unlock()
	return connection, nil
}

func (s *Service) Sync(ctx context.Context, id string) (SyncResult, error) {
	s.mu.RLock()
	connection, ok := s.connections[id]
	s.mu.RUnlock()
	if !ok {
		return SyncResult{}, ErrConnectionNotFound
	}
	adapter, ok := s.registry.Get(connection.Provider)
	if !ok {
		return SyncResult{}, ErrProviderNotFound
	}
	result, err := adapter.Sync(ctx, connection.externalID, connection.Cursor)
	if err != nil {
		return SyncResult{}, err
	}
	now := time.Now().UTC()
	connection.LastSyncedAt = &now
	connection.Cursor = result.NextCursor
	connection.Status = "healthy"
	s.mu.Lock()
	s.connections[id] = connection
	s.mu.Unlock()
	return SyncResult{Connection: connection, Accounts: result.Accounts, Transactions: result.Transactions}, nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
