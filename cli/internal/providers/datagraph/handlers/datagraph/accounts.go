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

// NewAccountNameResolver creates a new AccountNameResolver from an AccountGetter.
// Results are cached for the lifetime of the resolver to avoid duplicate API calls
// when multiple data graphs share the same account.
func NewAccountNameResolver(accounts AccountGetter) AccountNameResolver {
	return &cachedAccountNameResolver{
		inner: &accountNameResolver{accounts: accounts},
		cache: make(map[string]string),
	}
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

	return "", nil
}

// cachedAccountNameResolver wraps an AccountNameResolver with an in-memory cache
type cachedAccountNameResolver struct {
	inner AccountNameResolver
	cache map[string]string
}

func (r *cachedAccountNameResolver) GetAccountName(ctx context.Context, accountID string) (string, error) {
	if name, ok := r.cache[accountID]; ok {
		return name, nil
	}
	name, err := r.inner.GetAccountName(ctx, accountID)
	if err != nil {
		return "", err
	}
	r.cache[accountID] = name
	return name, nil
}
