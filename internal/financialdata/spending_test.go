package financialdata

import (
 "testing"
 "time"
)

func TestSummarizeSpendingFiltersAndDeduplicates(t *testing.T) {
 january := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
 tx := func(id, description, currency string, cents int64, date time.Time) Transaction {
  return Transaction{ConnectionID:"bank-1", AccountProviderExternalID:"checking", ProviderExternalID:id, Description:description, AmountCents:cents, Currency:currency, PostedAt:date}
 }
 transactions := []Transaction{
  tx("1","Amazon", "USD", 2500,january),
  tx("1","Amazon", "USD", 2500,january),
  tx("2","Starbucks", "USD", 700,january),
  tx("3","Transfer to savings", "USD", 9000,january),
  tx("4","Amazon refund", "USD", -500,january),
  tx("5","Amazon", "EUR", 800,january),
  tx("6","Amazon", "USD", 800,january.AddDate(0,1,0)),
 }
 summary := SummarizeSpending(transactions, january, "usd")
 if summary.TotalCents != 3200 || summary.TransactionCount != 2 { t.Fatalf("unexpected summary: %+v",summary) }
 if len(summary.Categories) != 2 || summary.Categories[0].Name != "Shopping" || summary.Categories[0].AmountCents != 2500 { t.Fatalf("unexpected categories: %+v",summary.Categories) }
 if len(summary.Merchants) != 2 || summary.Merchants[0].Name != "Amazon" { t.Fatalf("unexpected merchants: %+v",summary.Merchants) }
}

func TestSummarizeSpendingEmptyMonth(t *testing.T) {
 summary := SummarizeSpending(nil,time.Date(2026,2,1,0,0,0,0,time.UTC),"USD")
 if summary.TotalCents != 0 || len(summary.Categories) != 0 || len(summary.Merchants) != 0 { t.Fatalf("unexpected empty summary: %+v",summary) }
}
