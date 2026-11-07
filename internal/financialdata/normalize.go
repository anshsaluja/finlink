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
	// Preserve persisted fingerprints for ordinary identifiers so a re-sync
	// does not insert duplicates after upgrading the fingerprint algorithm.
	// Only ambiguous delimiter-containing identities use the new encoding.
	identity := strings.Join(fields, "\x1f")
	if strings.ContainsRune(identity, '\x1f') {
		// A joined identity always contains separators. Check individual fields.
		for _, field := range fields {
			if strings.ContainsRune(field, '\x1f') {
				var encoded strings.Builder
				for _, value := range fields {
					encoded.WriteString(strconv.Itoa(len(value)))
					encoded.WriteByte(':')
					encoded.WriteString(value)
				}
				identity = encoded.String()
				break
			}
		}
	}
	sum := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(sum[:])
}
