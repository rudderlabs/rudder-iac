package example

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/book"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/handlers/writer"
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
