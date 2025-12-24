package transformations

import (
	"context"
	"fmt"

	transformationsClient "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// Provider implements the provider interface for transformation resources
type Provider struct {
	*provider.BaseProvider
	client transformationsClient.TransformationStore
}

// New creates a new transformations provider instance
func New(client transformationsClient.TransformationStore) *Provider {
	handlers := []provider.Handler{
		NewHandler(client),
	}

	return &Provider{
		BaseProvider: provider.NewBaseProvider(handlers),
		client:       client,
	}
}

// ConsolidateSync performs batch publishing after successful execution
// This is called after all transformations have been created/updated with publish=false
func (p *Provider) ConsolidateSync(ctx context.Context, s *state.State) error {
	// Collect all transformations with versionIDs from state
	var transformationVersions []transformationsClient.TransformationVersionInput

	for _, resState := range s.Resources {
		// Only process transformation resources
		if resState.Type != HandlerMetadata.ResourceType {
			continue
		}

		// Extract versionID from outputRaw
		state, ok := resState.OutputRaw.(*TransformationState)
		if !ok || state == nil {
			continue
		}

		if state.VersionID == "" {
			continue // Skip resources without versionID
		}

		transformationVersions = append(transformationVersions, transformationsClient.TransformationVersionInput{
			VersionID: state.VersionID,
			TestInput: nil, // Use default test events
		})
	}

	// If nothing to publish, return early
	if len(transformationVersions) == 0 {
		return nil
	}

	// Call batch publish API
	publishReq := transformationsClient.BatchPublishRequest{
		Transformations: transformationVersions,
		Libraries:       []transformationsClient.LibraryVersionInput{}, // No libraries for now
	}

	resp, err := p.client.BatchPublish(ctx, publishReq)
	if err != nil {
		return fmt.Errorf("batch publish failed: %w", err)
	}

	if !resp.Published {
		return fmt.Errorf("batch publish returned published=false")
	}

	return nil
}
