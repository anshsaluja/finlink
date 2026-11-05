package provider

import (
	"context"
	"fmt"
	"time"
)

type MockNorthstar struct{}

func (MockNorthstar) Name() string { return "northstar" }

func (MockNorthstar) Connect(_ context.Context, credentials Credentials) (Connection, error) {
	if credentials["username"] == "" || credentials["password"] == "" {
		return Connection{}, ErrInvalidCredentials
	}
	return Connection{ExternalID: "ns-" + credentials["username"], Institution: "Northstar Bank"}, nil
}

func (MockNorthstar) Sync(_ context.Context, externalID, cursor string) (SyncResult, error) {
	if externalID == "" {
		return SyncResult{}, fmt.Errorf("external connection id is required")
	}
	if cursor != "" {
		return SyncResult{NextCursor: cursor}, nil
	}
	accountID := externalID + "-checking"
	// Deterministic, synthetic history across three months. Stable IDs make repeated
	// syncs safe to replay and let analytics demonstrate trends and recurring bills.
	entries := []struct {
		id, description string
		amount int64
		year int
		month time.Month
		day int
	}{
		{"txn-1", "Corner Coffee", -575, 2026, time.October, 31},
		{"txn-2", "Fresh Market Groceries", -6842, 2026, time.October, 28},
		{"txn-3", "Metro Transit", -275, 2026, time.October, 26},
		{"txn-4", "StreamFlix Subscription", -1599, 2026, time.October, 20},
		{"txn-5", "City Utilities", -8400, 2026, time.October, 15},
		{"txn-6", "Northside Rent", -120000, 2026, time.October, 1},
		{"txn-7", "Corner Coffee", -625, 2026, time.October, 17},
		{"txn-8", "Online Shop", -4250, 2026, time.October, 12},
		{"txn-9", "Employer Payroll", 240000, 2026, time.October, 15},
		{"txn-10", "Transfer to Savings", -30000, 2026, time.October, 10},
		{"txn-11", "Online Shop Refund", 1250, 2026, time.October, 18},
		{"txn-12", "Northside Rent", -120000, 2026, time.September, 1},
		{"txn-13", "StreamFlix Subscription", -1599, 2026, time.September, 20},
		{"txn-14", "Fresh Market Groceries", -7340, 2026, time.September, 24},
		{"txn-15", "Metro Transit", -275, 2026, time.September, 26},
		{"txn-16", "Corner Coffee", -575, 2026, time.September, 27},
		{"txn-17", "Online Shop", -8200, 2026, time.September, 12},
		{"txn-18", "Northside Rent", -120000, 2026, time.August, 1},
		{"txn-19", "StreamFlix Subscription", -1599, 2026, time.August, 20},
		{"txn-20", "Fresh Market Groceries", -5990, 2026, time.August, 22},
	}
	transactions := make([]Transaction, 0, len(entries))
	for _, entry := range entries {
		transactions = append(transactions, Transaction{
			ExternalID: externalID + "-" + entry.id,
			AccountExternalID: accountID,
			Description: entry.description,
			AmountCents: entry.amount,
			Currency: "USD",
			PostedAt: time.Date(entry.year, entry.month, entry.day, 12, 0, 0, 0, time.UTC),
		})
	}
	return SyncResult{
		Accounts: []Account{{ExternalID: accountID, Name: "Everyday Checking", Type: "checking", Currency: "USD", BalanceCents: 245075}},
		Transactions: transactions,
		NextCursor: "northstar:1",
	}, nil
}

type MockHarbor struct{}

func (MockHarbor) Name() string { return "harbor" }

func (MockHarbor) Connect(_ context.Context, credentials Credentials) (Connection, error) {
	if credentials["token"] == "" {
		return Connection{}, ErrInvalidCredentials
	}
	return Connection{ExternalID: "hb-" + credentials["token"], Institution: "Harbor Credit Union"}, nil
}

func (MockHarbor) Sync(_ context.Context, externalID, cursor string) (SyncResult, error) {
	if externalID == "" {
		return SyncResult{}, fmt.Errorf("external connection id is required")
	}
	if cursor != "" {
		return SyncResult{NextCursor: cursor}, nil
	}
	postedAt := time.Date(2026, 10, 30, 18, 30, 0, 0, time.UTC)
	return SyncResult{
		Accounts:     []Account{{ExternalID: "acct_" + externalID, Name: "Harbor Savings", Type: "savings", Currency: "USD", BalanceCents: 510000}},
		Transactions: []Transaction{{ExternalID: "entry_" + externalID + "_1", AccountExternalID: "acct_" + externalID, Description: "Payroll", AmountCents: 125000, Currency: "USD", PostedAt: postedAt}},
		NextCursor:   "harbor:1",
	}, nil
}
