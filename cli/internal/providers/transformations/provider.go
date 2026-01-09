package transformations

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Provider wraps BaseProvider and adds transformations-specific functionality
type Provider struct {
	provider.Provider
	store transformations.TransformationStore
}

// NewProvider creates a new transformations provider with all resource handlers
func NewProvider(client *client.Client) provider.Provider {
	// Create the transformation store
	store := transformations.NewRudderTransformationStore(client)

	// Register handlers - libraries first (dependencies), then transformations
	handlers := []provider.Handler{
		library.NewHandler(store),
		transformation.NewHandler(store),
	}

	return &Provider{
		Provider: provider.NewBaseProvider(handlers),
		store:    store,
	}
}

// ConsolidateSync implements batch publishing of all transformations and libraries
// This is called after all Create/Update operations to publish changes in a single batch
func (p *Provider) ConsolidateSync(ctx context.Context, st *state.State) error {
	// Build batch publish request by extracting versions from state
	req, err := p.buildBatchPublishRequest(st)
	if err != nil {
		return err
	}

	// If no versions to publish, we're done
	if len(req.Transformations) == 0 && len(req.Libraries) == 0 {
		return nil
	}

	// Batch publish all versions
	if err := p.store.BatchPublish(ctx, req); err != nil {
		return fmt.Errorf("batch publishing %d transformations and %d libraries: %w",
			len(req.Transformations), len(req.Libraries), err)
	}

	return nil
}

// buildBatchPublishRequest builds the batch publish request by extracting version IDs from state
func (p *Provider) buildBatchPublishRequest(st *state.State) (*transformations.BatchPublishRequest, error) {
	req := &transformations.BatchPublishRequest{
		Transformations: []transformations.BatchPublishTransformation{},
		Libraries:       []transformations.BatchPublishLibrary{},
	}

	for urn, resource := range st.Resources {
		switch resource.Type {
		case library.HandlerMetadata.ResourceType:
			libState, ok := resource.OutputRaw.(*model.LibraryState)
			if !ok {
				return nil, fmt.Errorf("resource %s has invalid OutputRaw type for library", urn)
			}
			if libState.VersionID == "" {
				return nil, fmt.Errorf("resource %s has empty version ID", urn)
			}

			req.Libraries = append(req.Libraries, transformations.BatchPublishLibrary{
				VersionID: libState.VersionID,
			})

		case transformation.HandlerMetadata.ResourceType:
			transState, ok := resource.OutputRaw.(*model.TransformationState)
			if !ok {
				return nil, fmt.Errorf("resource %s has invalid OutputRaw type for transformation", urn)
			}
			if transState.VersionID == "" {
				return nil, fmt.Errorf("resource %s has empty version ID", urn)
			}

			req.Transformations = append(req.Transformations, transformations.BatchPublishTransformation{
				VersionID: transState.VersionID,
			})
		}
	}

	return req, nil
}
