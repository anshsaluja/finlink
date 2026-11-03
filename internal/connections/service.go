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

// SyncCursor implements the sync request service's connection reader without
// exposing provider credentials or external identifiers across package boundaries.
func (s *Service) SyncCursor(ctx context.Context, id string) (string, error) {
	connection, err := s.repository.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return connection.Cursor, nil
}

func (s *Service) Sync(ctx context.Context, id string) (SyncResult, error) {
	connection, err := s.repository.Get(ctx, id)
	if err != nil {
		return SyncResult{}, err
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
	if err := s.repository.UpdateSyncState(ctx, connection); err != nil {
		return SyncResult{}, err
	}
	return SyncResult{Connection: connection, Accounts: result.Accounts, Transactions: result.Transactions}, nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(b[:])
}
