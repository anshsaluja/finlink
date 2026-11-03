package connections

import (
	"context"
	"testing"
	"time"

	"github.com/anshsaluja/finlink/internal/provider"
)

func TestMemoryRepositoryLifecycle(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	created := Connection{ID: "conn-1", Provider: "northstar", Institution: "Northstar Bank", Status: "connected", CreatedAt: time.Now().UTC(), externalID: "remote-1"}

	if err := repo.Create(ctx, created); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.externalID != created.externalID || got.Status != "connected" {
		t.Fatalf("unexpected stored connection: %#v", got)
	}

	now := time.Now().UTC()
	got.Status = "healthy"
	got.Cursor = "cursor-2"
	got.LastSyncedAt = &now
	if err := repo.UpdateSyncState(ctx, got); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Cursor != "cursor-2" || updated.Status != "healthy" || updated.LastSyncedAt == nil {
		t.Fatalf("sync state not persisted: %#v", updated)
	}
}

func TestMemoryRepositoryMissingConnection(t *testing.T) {
	repo := NewMemoryRepository()
	if _, err := repo.Get(context.Background(), "missing"); err != ErrConnectionNotFound {
		t.Fatalf("expected ErrConnectionNotFound, got %v", err)
	}
}

func TestReconnectReusesExistingConnectionAndSyncState(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	svc := NewServiceWithRepository(provider.NewRegistry(provider.MockNorthstar{}), repo)
	credentials := provider.Credentials{"username": "demo", "password": "demo"}
	first, err := svc.Create(ctx, "northstar", credentials)
	if err != nil { t.Fatal(err) }
	at := time.Now().UTC()
	if err := svc.MarkSynced(ctx, first.ID, "northstar:1", at); err != nil { t.Fatal(err) }
	again, err := svc.Create(ctx, "northstar", credentials)
	if err != nil { t.Fatal(err) }
	if again.ID != first.ID || again.Cursor != "northstar:1" || again.Status != "healthy" || again.LastSyncedAt == nil {
		t.Fatalf("reconnect should preserve connection and sync state: first=%+v again=%+v", first, again)
	}
}
