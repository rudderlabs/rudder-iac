package datagraph

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
)

// AccountNameResolver resolves human-readable names for accounts by ID
type AccountNameResolver interface {
	GetAccountName(ctx context.Context, accountID string) (string, error)
}

// AccountGetter abstracts the account retrieval API for testability
type AccountGetter interface {
	Get(ctx context.Context, id string) (*client.Account, error)
}

// accountNameResolver implements AccountNameResolver using the accounts API
type accountNameResolver struct {
	accounts AccountGetter
}

// NewAccountNameResolver creates a new AccountNameResolver from an AccountGetter
func NewAccountNameResolver(accounts AccountGetter) AccountNameResolver {
	return &accountNameResolver{accounts: accounts}
}

func (r *accountNameResolver) GetAccountName(ctx context.Context, accountID string) (string, error) {
	account, err := r.accounts.Get(ctx, accountID)
	if err != nil {
		return "", fmt.Errorf("fetching account %s: %w", accountID, err)
	}

	if account.Name != "" {
		return account.Name, nil
	}
	if account.Definition.Type != "" {
		return account.Definition.Type, nil
	}

	return "", fmt.Errorf("account %s has no name or definition type", accountID)
}
