package accounts

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Provider wraps the base provider to give a concrete type for dependency
// injection. Accounts are a single collection kind (`kind: accounts`).
type Provider struct {
	provider.Provider
}

// NewProvider creates the accounts provider around the given account store.
func NewProvider(store AccountStore) *Provider {
	handlers := []provider.Handler{
		NewHandler(store),
	}
	return &Provider{
		Provider: provider.NewBaseProvider(handlers),
	}
}

// SupportedMatchPatterns declares the `accounts` kind so the gatekeeper rules
// (duplicate-urn, metadata-syntax-valid, manifest-inline-conflict) scope to it —
// without this they match nothing for the kind. Accounts is a new resource, so
// like the destination and data-graph providers it ships `rudder/v1` only (no
// legacy versions).
func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	return prules.V1VersionPatterns(HandlerMetadata.SpecKind)
}
