package financialdata

import (
	"testing"
	"time"

	"github.com/anshsaluja/finlink/internal/provider"
)

func TestNormalizeCanonicalizesAndFingerprints(t *testing.T) {
	posted := time.Date(2026, 11, 1, 12, 30, 0, 0, time.FixedZone("EST", -5*60*60))
	now := time.Date(2026, 11, 3, 7, 0, 0, 0, time.UTC)
	input := provider.SyncResult{
		Accounts: []provider.Account{{ExternalID: "acct-1", Name: " Checking ", Type: "CHECKING", Currency: "usd", BalanceCents: 12500}},
		Transactions: []provider.Transaction{{ExternalID: "txn-1", AccountExternalID: "acct-1", Description: " Coffee ", AmountCents: -525, Currency: "usd", PostedAt: posted}},
	}
	accounts, transactions := Normalize("conn-1", input, now)
	if len(accounts) != 1 || accounts[0].Name != "Checking" || accounts[0].Type != "checking" || accounts[0].Currency != "USD" {
		t.Fatalf("unexpected account: %+v", accounts)
	}
	if len(transactions) != 1 || transactions[0].Description != "Coffee" || transactions[0].Currency != "USD" || len(transactions[0].Fingerprint) != 64 {
		t.Fatalf("unexpected transaction: %+v", transactions)
	}
	_, repeated := Normalize("conn-1", input, now.Add(time.Hour))
	if repeated[0].Fingerprint != transactions[0].Fingerprint {
		t.Fatalf("fingerprint changed: %q != %q", repeated[0].Fingerprint, transactions[0].Fingerprint)
	}
	input.Transactions[0].ExternalID = "provider-reissued-id"
	_, reissued := Normalize("conn-1", input, now)
	if reissued[0].Fingerprint != transactions[0].Fingerprint {
		t.Fatal("fallback fingerprint must not depend on provider transaction id")
	}
}

func TestFingerprintChangesWithFinancialIdentity(t *testing.T) {
	posted := time.Date(2026, 11, 1, 12, 30, 0, 0, time.UTC)
	base := transactionFingerprint("conn", "acct", "Coffee", -525, "USD", posted)
	changed := transactionFingerprint("conn", "acct", "Coffee", -625, "USD", posted)
	if base == changed {
		t.Fatal("amount change must change fingerprint")
	}
}
