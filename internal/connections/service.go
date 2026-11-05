package connections

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
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
	registry   *provider.Registry
	repository Repository
}

func NewService(registry *provider.Registry) *Service {
	return NewServiceWithRepository(registry, NewMemoryRepository())
}

func NewServiceWithRepository(registry *provider.Registry, repository Repository) *Service {
	return &Service{registry: registry, repository: repository}
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
	if err := s.repository.Create(ctx, connection); err != nil {
		return Connection{}, err
	}
	return connection, nil
}

// Get returns the durable, public connection state without exposing provider credentials
// or the provider's internal connection identifier.
func (s *Service) Get(ctx context.Context, id string) (Connection, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) SyncCursor(ctx context.Context, id string) (string, error) {
	connection, err := s.repository.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return connection.Cursor, nil
}

// FetchSyncPage executes a provider sync without mutating durable connection state.
// Workers call MarkSynced only after canonical data has been persisted successfully.
func (s *Service) FetchSyncPage(ctx context.Context, id, cursor string) (provider.SyncResult, error) {
	connection, err := s.repository.Get(ctx, id)
	if err != nil {
		return provider.SyncResult{}, err
	}
	adapter, ok := s.registry.Get(connection.Provider)
	if !ok {
		return provider.SyncResult{}, ErrProviderNotFound
	}
	return adapter.Sync(ctx, connection.externalID, cursor)
}

func (s *Service) MarkSynced(ctx context.Context, id, cursor string, at time.Time) error {
	connection, err := s.repository.Get(ctx, id)
	if err != nil {
		return err
	}
	at = at.UTC()
	connection.LastSyncedAt = &at
	connection.Cursor = cursor
	connection.Status = "healthy"
	return s.repository.UpdateSyncState(ctx, connection)
}

// Sync remains useful for direct/local callers. The asynchronous production path uses
// FetchSyncPage + persistence + MarkSynced so cursors never advance before data storage.
func (s *Service) Sync(ctx context.Context, id string) (SyncResult, error) {
	connection, err := s.repository.Get(ctx, id)
	if err != nil {
		return SyncResult{}, err
	}
	result, err := s.FetchSyncPage(ctx, id, connection.Cursor)
	if err != nil {
		return SyncResult{}, err
	}
	now := time.Now().UTC()
	if err := s.MarkSynced(ctx, id, result.NextCursor, now); err != nil {
		return SyncResult{}, err
	}
	connection.LastSyncedAt = &now
	connection.Cursor = result.NextCursor
	connection.Status = "healthy"
	return SyncResult{Connection: connection, Accounts: result.Accounts, Transactions: result.Transactions}, nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
