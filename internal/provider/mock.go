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
	postedAt := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	return SyncResult{
		Accounts: []Account{{ExternalID: externalID + "-checking", Name: "Everyday Checking", Type: "checking", Currency: "USD", BalanceCents: 245075}},
		Transactions: []Transaction{{ExternalID: externalID + "-txn-1", AccountExternalID: externalID + "-checking", Description: "Corner Coffee", AmountCents: -575, Currency: "USD", PostedAt: postedAt}},
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
		Accounts: []Account{{ExternalID: "acct_" + externalID, Name: "Harbor Savings", Type: "savings", Currency: "USD", BalanceCents: 510000}},
		Transactions: []Transaction{{ExternalID: "entry_" + externalID + "_1", AccountExternalID: "acct_" + externalID, Description: "Payroll", AmountCents: 125000, Currency: "USD", PostedAt: postedAt}},
		NextCursor: "harbor:1",
	}, nil
}
