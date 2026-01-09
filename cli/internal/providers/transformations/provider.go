package transformations

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Provider wraps BaseProvider and adds transformations-specific functionality
type Provider struct {
	provider.Provider
	compositeHandler *CompositeTransformationsHandler
}

// NewProvider creates a new transformations provider with all resource handlers
func NewProvider(client *client.Client) provider.Provider {
	// Create the composite handler which manages both transformation and library handlers
	compositeHandler := NewCompositeHandler(client)

	return &Provider{
		Provider:         provider.NewBaseProvider(compositeHandler.Handlers()),
		compositeHandler: compositeHandler,
	}
}

// ConsolidateSync implements batch publishing of all transformations and libraries
// This is called after all Create/Update operations to publish changes in a single batch
func (p *Provider) ConsolidateSync(ctx context.Context, st *state.State) error {
	// Build batch publish request by extracting versions from state
	req, err := p.compositeHandler.BuildBatchPublishRequest(st)
	if err != nil {
		return err
	}

	// If no versions to publish, we're done
	if len(req.Transformations) == 0 && len(req.Libraries) == 0 {
		return nil
	}

	// Batch publish all versions
	if err := p.compositeHandler.BatchPublish(ctx, req); err != nil {
		return fmt.Errorf("batch publishing %d transformations and %d libraries: %w",
			len(req.Transformations), len(req.Libraries), err)
	}

	return nil
}
