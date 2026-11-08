package financialdata

import (
	"testing"
	"time"
)

func TestSummarizeSpendingFiltersAndDeduplicates(t *testing.T) {
	january := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	tx := func(id, description, currency string, cents int64, date time.Time) Transaction {
	return Transaction{ConnectionID: "bank-1", AccountProviderExternalID: "checking", ProviderExternalID: id, Description: description, AmountCents: cents, Currency: currency, PostedAt: date}
	}
	transactions := []Transaction{
		tx("1", "Amazon", "USD", 2500, january),
		tx("1", "Amazon", "USD", 2500, january),
		tx("2", "Starbucks", "USD", 700, january),
		tx("3", "Transfer to savings", "USD", 9000, january),
		tx("4", "Amazon refund", "USD", -500, january),
		tx("5", "Amazon", "EUR", 800, january),
		tx("6", "Amazon", "USD", 800, january.AddDate(0, 1, 0)),
	}
	summary := SummarizeSpending(transactions, january, "usd")
	if summary.TotalCents != 3200 || summary.TransactionCount != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if len(summary.Categories) != 2 || summary.Categories[0].Name != "Shopping" || summary.Categories[0].AmountCents != 2500 {
		t.Fatalf("unexpected categories: %+v", summary.Categories)
	}
	if len(summary.Merchants) != 2 || summary.Merchants[0].Name != "Amazon" {
		t.Fatalf("unexpected merchants: %+v", summary.Merchants)
	}
}

func TestSummarizeSpendingEmptyMonth(t *testing.T) {
	summary := SummarizeSpending(nil, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "USD")
	if summary.TotalCents != 0 || len(summary.Categories) != 0 || len(summary.Merchants) != 0 {
		t.Fatalf("unexpected empty summary: %+v", summary)
	}
}

func TestSummarizeSpendingDistinctPurchasesWithoutProviderIDs(t *testing.T) {
	day := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	transactions := []Transaction{
		{ConnectionID: "c1", AccountProviderExternalID: "a1", Description: "  STARBUCKS  ", AmountCents: 500, Currency: "USD", PostedAt: day},
		{ConnectionID: "c1", AccountProviderExternalID: "a1", Description: "starbucks", AmountCents: 700, Currency: "USD", PostedAt: day},
		{ConnectionID: "c1", AccountProviderExternalID: "a1", Description: "Starbucks", AmountCents: 500, Currency: "USD", PostedAt: day},
	}
	summary := SummarizeSpending(transactions, day, "USD")
	if summary.TotalCents != 1700 || summary.TransactionCount != 3 {
		t.Fatalf("purchases without stable identifiers must not be deduplicated: %+v", summary)
	}
	if len(summary.Merchants) != 1 || summary.Merchants[0].Name != "Starbucks" || summary.Merchants[0].Count != 3 {
		t.Fatalf("merchant capitalization and whitespace must be normalized: %+v", summary.Merchants)
	}
}

func TestSummarizeSpendingDeduplicatesFingerprints(t *testing.T) {
	day := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	transactions := []Transaction{
		{ConnectionID: "c1", AccountProviderExternalID: "a1", Fingerprint: "same", Description: "Coffee", AmountCents: 500, Currency: "USD", PostedAt: day},
		{ConnectionID: "c1", AccountProviderExternalID: "a1", Fingerprint: "same", Description: "Coffee", AmountCents: 500, Currency: "USD", PostedAt: day},
		{ConnectionID: "c1", AccountProviderExternalID: "a2", Fingerprint: "same", Description: "Coffee", AmountCents: 500, Currency: "USD", PostedAt: day},
	}
	summary := SummarizeSpending(transactions, day, "USD")
	if summary.TotalCents != 1000 || summary.TransactionCount != 2 {
		t.Fatalf("fingerprints should deduplicate only within their connection and account: %+v", summary)
	}
}

func TestSummarizeSpendingIdentityTypesAndDelimiters(t *testing.T) {
	day := time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC)
	transactions := []Transaction{
		{ConnectionID: "bank", AccountProviderExternalID: "checking", ProviderExternalID: "same", Description: "Coffee", AmountCents: 500, Currency: "USD", PostedAt: day},
		{ConnectionID: "bank", AccountProviderExternalID: "checking", Fingerprint: "same", Description: "Coffee", AmountCents: 600, Currency: "USD", PostedAt: day},
		{ConnectionID: "bank|checking", AccountProviderExternalID: "a", ProviderExternalID: "same", Description: "Coffee", AmountCents: 700, Currency: "USD", PostedAt: day},
		{ConnectionID: "bank", AccountProviderExternalID: "checking|a", ProviderExternalID: "same", Description: "Coffee", AmountCents: 800, Currency: "USD", PostedAt: day},
		{ConnectionID: "bank", AccountProviderExternalID: "checking", ProviderExternalID: "same", Description: "Coffee", AmountCents: 500, Currency: "USD", PostedAt: day},
	}
	summary := SummarizeSpending(transactions, day, "USD")
	if summary.TotalCents != 2600 || summary.TransactionCount != 4 {
		t.Fatalf("identity type and account boundaries must remain distinct: %+v", summary)
	}
}

func TestSummarizeSpendingRejectsMissingPostedAt(t *testing.T) {
	month := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	transactions := []Transaction{
		{ConnectionID: "bank", AccountProviderExternalID: "checking", ProviderExternalID: "missing", Description: "Coffee", AmountCents: 500, Currency: "USD"},
		{ConnectionID: "bank", AccountProviderExternalID: "checking", ProviderExternalID: "valid", Description: "Coffee", AmountCents: 700, Currency: "USD", PostedAt: month.AddDate(0, 0, 1)},
	}
	summary := SummarizeSpending(transactions, month, "USD")
	if summary.TotalCents != 700 || summary.TransactionCount != 1 {
		t.Fatalf("missing timestamps must not be included in spending totals: %+v", summary)
	}
}

func TestSummarizeSpendingUsesUTCMonthAndCurrency(t *testing.T) {
	month := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	west := time.FixedZone("PDT", -7*60*60)
	east := time.FixedZone("JST", 9*60*60)
	transactions := []Transaction{
		{ProviderExternalID: "march-utc", Description: "Coffee", AmountCents: 100, Currency: "USD", PostedAt: time.Date(2026, 3, 31, 16, 59, 0, 0, west)},
		{ProviderExternalID: "april-utc", Description: "Coffee", AmountCents: 200, Currency: " usd ", PostedAt: time.Date(2026, 4, 1, 8, 0, 0, 0, east)},
		{ProviderExternalID: "may-utc", Description: "Coffee", AmountCents: 300, Currency: "USD", PostedAt: time.Date(2026, 4, 30, 18, 0, 0, 0, west)},
		{ProviderExternalID: "other-currency", Description: "Coffee", AmountCents: 400, Currency: "EUR", PostedAt: month},
	}
	summary := SummarizeSpending(transactions, month, " usd ")
	if summary.Month != "2026-04" || summary.Currency != "USD" || summary.TotalCents != 300 || summary.TransactionCount != 2 {
		t.Fatalf("UTC month boundaries and currency filtering failed: %+v", summary)
	}
}
