package financialdata

import (
 "sort"
 "strconv"
 "strings"
 "time"
)

// SpendingSummary aggregates posted debit transactions in one currency and UTC month.
// Positive amounts represent expenditure; negative amounts are credits/refunds.
// Transfers are excluded, and duplicate provider IDs or fingerprints are counted once.
type SpendingSummary struct {
 Month string `json:"month"`
 Currency string `json:"currency"`
 TotalCents int64 `json:"total_cents"`
 TransactionCount int `json:"transaction_count"`
 Categories []SpendingGroup `json:"categories"`
 Merchants []SpendingGroup `json:"merchants"`
}
type SpendingGroup struct {
 Name string `json:"name"`
 AmountCents int64 `json:"amount_cents"`
 Count int `json:"count"`
}

func SummarizeSpending(transactions []Transaction, month time.Time, currency string) SpendingSummary {
 month = month.UTC()
 currency = strings.ToUpper(strings.TrimSpace(currency))
 result := SpendingSummary{Month: month.Format("2006-01"), Currency: currency, Categories: []SpendingGroup{}, Merchants: []SpendingGroup{}}
 categories := map[string]*SpendingGroup{}
 merchants := map[string]*SpendingGroup{}
 seen := map[string]bool{}
 for _, tx := range transactions {
  if tx.PostedAt.UTC().Format("2006-01") != result.Month || strings.ToUpper(strings.TrimSpace(tx.Currency)) != currency || tx.AmountCents <= 0 { continue }
  description := strings.ToLower(strings.TrimSpace(tx.Description))
  if isTransfer(description) { continue }
  identity := strings.TrimSpace(tx.ProviderExternalID)
  if identity == "" { identity = strings.TrimSpace(tx.Fingerprint) }
  if identity == "" { identity = tx.PostedAt.UTC().Format(time.RFC3339Nano) + "|" + description + "|" + tx.AccountProviderExternalID + "|" + strconv.FormatInt(tx.AmountCents, 10) }
  key := tx.ConnectionID + "|" + tx.AccountProviderExternalID + "|" + identity
  if seen[key] { continue }
  seen[key] = true
  category := categorize(description)
  merchant := normalizeMerchant(tx.Description)
  if merchant == "" { merchant = "Unknown merchant" }
  addGroup(categories, category, tx.AmountCents)
  addGroup(merchants, merchant, tx.AmountCents)
  result.TotalCents += tx.AmountCents
  result.TransactionCount++
 }
 result.Categories = sortedGroups(categories)
 result.Merchants = sortedGroups(merchants)
 return result
}
// normalizeMerchant groups casing and repeated whitespace without inventing merchant identities.
func normalizeMerchant(description string) string {
 return strings.ToLower(strings.Join(strings.Fields(description), " "))
}
func addGroup(groups map[string]*SpendingGroup, name string, cents int64) {
 if groups[name] == nil { groups[name] = &SpendingGroup{Name:name} }
 groups[name].AmountCents += cents
 groups[name].Count++
}
func sortedGroups(groups map[string]*SpendingGroup) []SpendingGroup {
 out := make([]SpendingGroup, 0, len(groups))
 for _, g := range groups { out = append(out, *g) }
 sort.Slice(out, func(i,j int) bool {
  if out[i].AmountCents != out[j].AmountCents { return out[i].AmountCents > out[j].AmountCents }
  return out[i].Name < out[j].Name
 })
 return out
}
func isTransfer(description string) bool {
 for _, term := range []string{"transfer to ", "transfer from ", "internal transfer", "account transfer", "zelle transfer"} {
  if strings.Contains(description, term) { return true }
 }
 return false
}
func categorize(description string) string {
 switch {
 case containsAny(description, "amazon", "target", "walmart", "shopping", "store"): return "Shopping"
 case containsAny(description, "starbucks", "restaurant", "cafe", "doordash", "uber eats", "grocery", "whole foods"): return "Food & Dining"
 case containsAny(description, "rent", "electric", "utility", "internet", "phone bill"): return "Bills"
 case containsAny(description, "uber", "lyft", "metro", "transit", "gas station"): return "Transport"
 case containsAny(description, "netflix", "spotify", "subscription"): return "Subscriptions"
 default: return "Other"
 }
}
func containsAny(s string, terms ...string) bool {
 for _, term := range terms { if strings.Contains(s,term) { return true } }
 return false
}
