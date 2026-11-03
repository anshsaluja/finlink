package syncs

import (
	"context"
	"testing"
)

type fakeConnections struct{ cursor string }

func (f fakeConnections) SyncCursor(context.Context, string) (string, error) { return f.cursor, nil }

func TestRequestPersistsAndPublishes(t *testing.T) {
	repo := NewMemoryRepository()
	publisher := &MemoryPublisher{}
	service := NewService(repo, fakeConnections{cursor: "cursor-7"}, publisher)
	run, err := service.Request(context.Background(), "conn-1", "request-1")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "requested" || run.CursorBefore != "cursor-7" {
		t.Fatalf("unexpected run: %+v", run)
	}
	if len(publisher.Events) != 1 || publisher.Events[0].RunID != run.ID {
		t.Fatalf("events=%+v", publisher.Events)
	}
}

func TestMemoryRepositoryIdempotency(t *testing.T) {
	repo := NewMemoryRepository()
	first, _ := repo.Create(context.Background(), Run{IdempotencyKey: "same"})
	second, _ := repo.Create(context.Background(), Run{IdempotencyKey: "same"})
	if first.ID != second.ID {
		t.Fatalf("ids differ: %d %d", first.ID, second.ID)
	}
}
