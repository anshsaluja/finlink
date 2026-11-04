package provider

import (
	"context"
	"errors"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid provider credentials")

type Credentials map[string]string

type Connection struct {
	ExternalID  string
	Institution string
}

type Account struct {
	ExternalID   string
	Name         string
	Type         string
	Currency     string
	BalanceCents int64
}

type Transaction struct {
	ExternalID        string
	AccountExternalID string
	Description       string
	AmountCents       int64
	Currency          string
	PostedAt          time.Time
}

type SyncResult struct {
	Accounts     []Account
	Transactions []Transaction
	NextCursor   string
}

type Adapter interface {
	Name() string
	Connect(context.Context, Credentials) (Connection, error)
	Sync(context.Context, string, string) (SyncResult, error)
}

type Registry struct {
	adapters map[string]Adapter
}

func NewRegistry(adapters ...Adapter) *Registry {
	r := &Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		r.adapters[adapter.Name()] = adapter
	}
	return r
}

func (r *Registry) Get(name string) (Adapter, bool) {
	adapter, ok := r.adapters[name]
	return adapter, ok
}
