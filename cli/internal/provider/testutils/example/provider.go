package example

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/book"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Provider wraps the base provider to provide a concrete type for dependency injection
type Provider struct {
	provider.Provider
}

// NewProvider creates a new example provider with all resource handlers
func NewProvider(backend *backend.Backend) *Provider {
	handlers := []provider.Handler{
		writer.NewHandler(backend),
		book.NewHandler(backend),
	}

	return &Provider{
		Provider: provider.NewBaseProvider(handlers),
	}
}

// SupportedMatchPatterns declares the example kinds so the gatekeeper rules
// (e.g. metadata-syntax-valid) scope to them — without this they match nothing.
func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	var patterns []rules.MatchPattern
	for _, kind := range []string{writer.HandlerMetadata.SpecKind, book.HandlerMetadata.SpecKind} {
		patterns = append(patterns, prules.LegacyVersionPatterns(kind)...)
		patterns = append(patterns, prules.V1VersionPatterns(kind)...)
	}
	return patterns
}
