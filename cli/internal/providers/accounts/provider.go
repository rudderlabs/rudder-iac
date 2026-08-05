package accounts

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Provider wraps BaseProvider with the single account handler — the same shape
// as the destination provider.
type Provider struct {
	*provider.BaseProvider
}

// NewProvider constructs the accounts provider around the given store.
func NewProvider(store AccountStore) *Provider {
	return &Provider{
		BaseProvider: provider.NewBaseProvider([]provider.Handler{
			NewHandler(store),
		}),
	}
}

// SupportedMatchPatterns declares the account kind for rudder/v1 only (accounts
// are new — no legacy versions), scoping the gatekeeper rules to it.
func (p *Provider) SupportedMatchPatterns() []vrules.MatchPattern {
	return prules.V1VersionPatterns(AccountSpecKind)
}
