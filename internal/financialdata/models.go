package financialdata

import "time"

// Account is FinLink's provider-independent account representation.
type Account struct {
	ConnectionID       string    `json:"connection_id"`
	ProviderExternalID string    `json:"provider_external_id"`
	Name               string    `json:"name"`
	Type               string    `json:"type"`
	Currency           string    `json:"currency"`
	BalanceCents       int64     `json:"balance_cents"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Transaction is FinLink's canonical transaction representation. ProviderExternalID is
// the primary deduplication identity within a connection; Fingerprint provides a stable
// fallback for providers that do not supply transaction IDs.
type Transaction struct {
	ConnectionID             string    `json:"connection_id"`
	ProviderExternalID       string    `json:"provider_external_id"`
	AccountProviderExternalID string   `json:"account_provider_external_id"`
	Description              string    `json:"description"`
	AmountCents              int64     `json:"amount_cents"`
	Currency                 string    `json:"currency"`
	PostedAt                 time.Time `json:"posted_at"`
	Fingerprint              string    `json:"fingerprint"`
}
