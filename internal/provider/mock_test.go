package provider

import (
	"context"
	"testing"
	"time"
)

func TestMockNorthstarProvidesStableMultiMonthHistory(t *testing.T) {
	adapter := MockNorthstar{}
	first, err := adapter.Sync(context.Background(), "ns-demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Accounts) != 1 || len(first.Transactions) < 15 {
		t.Fatalf("expected one account and realistic history, got %d accounts and %d transactions", len(first.Accounts), len(first.Transactions))
	}
	seen := make(map[string]bool)
	months := make(map[time.Month]bool)
	var transfer, refund, income bool
	for _, txn := range first.Transactions {
		if txn.ExternalID == "" || seen[txn.ExternalID] {
			t.Fatalf("duplicate or empty transaction ID: %q", txn.ExternalID)
		}
		seen[txn.ExternalID] = true
		if txn.AccountExternalID != first.Accounts[0].ExternalID {
			t.Fatalf("transaction %q references wrong account", txn.ExternalID)
		}
		if txn.PostedAt.Location() != time.UTC {
			t.Fatalf("transaction %q is not UTC", txn.ExternalID)
		}
		months[txn.PostedAt.Month()] = true
		switch txn.Description {
		case "Transfer to Savings":
			transfer = txn.AmountCents < 0
		case "Online Shop Refund":
			refund = txn.AmountCents > 0
		case "Employer Payroll":
			income = txn.AmountCents > 0
		}
	}
	if !months[time.August] || !months[time.September] || !months[time.October] {
		t.Fatalf("expected three months of transactions: %v", months)
	}
	if !transfer || !refund || !income {
		t.Fatal("mock history must include transfer, refund and income to test analytics exclusions")
	}
	replay, err := adapter.Sync(context.Background(), "ns-demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(replay.Transactions) != len(first.Transactions) || replay.NextCursor != first.NextCursor {
		t.Fatal("replaying first page must be deterministic")
	}
	for i := range first.Transactions {
		if replay.Transactions[i] != first.Transactions[i] {
			t.Fatalf("replayed transaction %d changed", i)
		}
	}
	next, err := adapter.Sync(context.Background(), "ns-demo", first.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Transactions) != 0 || len(next.Accounts) != 0 {
		t.Fatal("subsequent sync must not re-emit the first page")
	}
}
