package connections

import (
	"context"
	"testing"
	"time"
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
