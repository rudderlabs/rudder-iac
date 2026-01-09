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

// CompositeTransformationsHandler provides unified access to transformation and library handlers
// Used by the provider for operations that span multiple handlers like ConsolidateSync
type CompositeTransformationsHandler struct {
	store                 transformations.TransformationStore
	libraryHandler        *library.LibraryHandler
	transformationHandler *transformation.TransformationHandler
}

// NewCompositeHandler creates a composite handler that manages both transformation and library handlers
func NewCompositeHandler(client *client.Client) *CompositeTransformationsHandler {
	// Create store once
	store := transformations.NewRudderTransformationStore(client)

	return &CompositeTransformationsHandler{
		store:                 store,
		libraryHandler:        library.NewHandler(store),        // Receives store
		transformationHandler: transformation.NewHandler(store), // Receives store
	}
}

// Handlers returns the individual handlers for registration with BaseProvider
// Libraries first (dependencies), then transformations
func (c *CompositeTransformationsHandler) Handlers() []provider.Handler {
	return []provider.Handler{
		c.libraryHandler,        // Libraries first (dependencies)
		c.transformationHandler, // Transformations second
	}
}

// BuildBatchPublishRequest builds the batch publish request by extracting version IDs from state
// This is used during ConsolidateSync to collect all versions that need to be published
func (c *CompositeTransformationsHandler) BuildBatchPublishRequest(st *state.State) (*transformations.BatchPublishRequest, error) {
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
				// TestInput can be added later when test support is implemented
			})
		}
	}

	return req, nil
}

// BatchPublish publishes all versions in a single batch operation
func (c *CompositeTransformationsHandler) BatchPublish(ctx context.Context, req *transformations.BatchPublishRequest) error {
	if len(req.Transformations) == 0 && len(req.Libraries) == 0 {
		return nil
	}

	return c.store.BatchPublish(ctx, req)
}
