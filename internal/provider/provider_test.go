package provider

import (
	"context"
	"errors"
	"testing"
)

func TestMockProvidersExposeDifferentCredentialFlows(t *testing.T) {
	ctx := context.Background()
	northstar := MockNorthstar{}
	if _, err := northstar.Connect(ctx, Credentials{"username": "demo", "password": "secret"}); err != nil {
		t.Fatalf("northstar connect: %v", err)
	}
	if _, err := northstar.Connect(ctx, Credentials{"token": "wrong-shape"}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
	harbor := MockHarbor{}
	if _, err := harbor.Connect(ctx, Credentials{"token": "sandbox-token"}); err != nil {
		t.Fatalf("harbor connect: %v", err)
	}
}

func TestMockProviderIncrementalCursorReturnsNoDuplicatePage(t *testing.T) {
	result, err := (MockNorthstar{}).Sync(context.Background(), "ns-demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Transactions) != 1 || result.NextCursor == "" {
		t.Fatalf("unexpected first page: %+v", result)
	}
	next, err := (MockNorthstar{}).Sync(context.Background(), "ns-demo", result.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Transactions) != 0 {
		t.Fatalf("expected no duplicate transactions, got %d", len(next.Transactions))
	}
}
