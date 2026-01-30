package datagraph

import (
	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
)

// Provider wraps the base provider to provide a concrete type for dependency injection
type Provider struct {
	provider.Provider
}

// NewProvider creates a new data graph provider instance
func NewProvider(client dgClient.DataGraphStore) *Provider {
	handlers := []provider.Handler{
		datagraph.NewHandler(client),
	}

	return &Provider{
		Provider: provider.NewBaseProvider(handlers),
	}
}
