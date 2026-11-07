package financialdata

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/anshsaluja/finlink/internal/provider"
)

// Normalize converts provider-specific sync output into FinLink's canonical data model.
func Normalize(connectionID string, result provider.SyncResult, now time.Time) ([]Account, []Transaction) {
	accounts := make([]Account, 0, len(result.Accounts))
	for _, account := range result.Accounts {
		accounts = append(accounts, Account{
			ConnectionID: connectionID, ProviderExternalID: account.ExternalID,
			Name: strings.TrimSpace(account.Name), Type: strings.ToLower(strings.TrimSpace(account.Type)),
			Currency: strings.ToUpper(strings.TrimSpace(account.Currency)), BalanceCents: account.BalanceCents,
			UpdatedAt: now.UTC(),
		})
	}
	transactions := make([]Transaction, 0, len(result.Transactions))
	for _, transaction := range result.Transactions {
		description := strings.TrimSpace(transaction.Description)
		currency := strings.ToUpper(strings.TrimSpace(transaction.Currency))
		postedAt := transaction.PostedAt.UTC()
		transactions = append(transactions, Transaction{
			ConnectionID: connectionID, ProviderExternalID: transaction.ExternalID,
			AccountProviderExternalID: transaction.AccountExternalID, Description: description,
			AmountCents: transaction.AmountCents, Currency: currency, PostedAt: postedAt,
			Fingerprint: transactionFingerprint(connectionID, transaction.AccountExternalID, description, transaction.AmountCents, currency, postedAt),
		})
	}
	return accounts, transactions
}

func transactionFingerprint(connectionID, accountID, description string, amount int64, currency string, postedAt time.Time) string {
	fields := []string{connectionID, accountID, description, strconv.FormatInt(amount, 10), currency, postedAt.UTC().Format(time.RFC3339Nano)}
	var identity strings.Builder
	for _, field := range fields {
		identity.WriteString(strconv.Itoa(len(field)))
		identity.WriteByte(':')
		identity.WriteString(field)
	}
	sum := sha256.Sum256([]byte(identity.String()))
	return hex.EncodeToString(sum[:])
}
