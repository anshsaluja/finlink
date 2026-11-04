package syncworker

import (
	"context"
	"errors"
	"github.com/anshsaluja/finlink/internal/financialdata"
	"github.com/anshsaluja/finlink/internal/provider"
	"github.com/anshsaluja/finlink/internal/syncs"
	"testing"
	"time"
)

type fakeProvider struct {
	result provider.SyncResult
	marked bool
}

func (f *fakeProvider) FetchSyncPage(context.Context, string, string) (provider.SyncResult, error) {
	return f.result, nil
}
func (f *fakeProvider) MarkSynced(context.Context, string, string, time.Time) error {
	f.marked = true
	return nil
}

type fakeData struct {
	err    error
	stored bool
}

func (f *fakeData) Store(context.Context, []financialdata.Account, []financialdata.Transaction) error {
	f.stored = true
	return f.err
}

type fakeRuns struct{ status string }

func (f *fakeRuns) MarkStarted(context.Context, int64, time.Time) error {
	f.status = "processing"
	return nil
}
func (f *fakeRuns) MarkCompleted(context.Context, int64, string, int, int, time.Time) error {
	f.status = "succeeded"
	return nil
}
func (f *fakeRuns) MarkFailed(context.Context, int64, string, time.Time) error {
	f.status = "failed"
	return nil
}
func TestProcessPersistsBeforeAdvancingCursor(t *testing.T) {
	p := &fakeProvider{result: provider.SyncResult{NextCursor: "next"}}
	d := &fakeData{}
	r := &fakeRuns{}
	w := New(p, d, r)
	if err := w.Process(context.Background(), syncs.RequestedEvent{RunID: 1, ConnectionID: "c"}); err != nil {
		t.Fatal(err)
	}
	if !d.stored || !p.marked || r.status != "succeeded" {
		t.Fatalf("stored=%v marked=%v status=%s", d.stored, p.marked, r.status)
	}
}
func TestProcessDoesNotAdvanceCursorWhenPersistenceFails(t *testing.T) {
	p := &fakeProvider{result: provider.SyncResult{NextCursor: "next"}}
	d := &fakeData{err: errors.New("db unavailable")}
	r := &fakeRuns{}
	w := New(p, d, r)
	if err := w.Process(context.Background(), syncs.RequestedEvent{RunID: 1, ConnectionID: "c"}); err == nil {
		t.Fatal("expected error")
	}
	if p.marked {
		t.Fatal("cursor advanced after failed persistence")
	}
	if r.status != "failed" {
		t.Fatalf("status=%s", r.status)
	}
}
