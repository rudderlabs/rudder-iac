package helpers

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/transformations"
)

const (
	TransformationResourceType = "transformation"
	LibraryResourceType        = "transformation-library"
)

var _ UpstreamAdapter = &TransformationAdapter{}

// TransformationAdapter wraps the transformations.TransformationStore client
// and implements the UpstreamAdapter interface.
type TransformationAdapter struct {
	store transformations.TransformationStore
}

// NewTransformationAdapter creates a new TransformationAdapter instance
func NewTransformationAdapter(store transformations.TransformationStore) *TransformationAdapter {
	return &TransformationAdapter{
		store: store,
	}
}

func (a *TransformationAdapter) RemoteIDs(ctx context.Context) (map[string]string, error) {
	resourceIDs := make(map[string]string)

	// Fetch all transformations with external IDs (managed by CLI)
	allTransformations, err := a.store.ListTransformations(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing transformations: %w", err)
	}
	for _, t := range allTransformations {
		if t.ExternalID != "" {
			resourceIDs[urn(TransformationResourceType, t.ExternalID)] = t.ID
		}
	}

	// Fetch all libraries with external IDs (managed by CLI)
	allLibraries, err := a.store.ListLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing libraries: %w", err)
	}
	for _, lib := range allLibraries {
		if lib.ExternalID != "" {
			resourceIDs[urn(LibraryResourceType, lib.ExternalID)] = lib.ID
		}
	}

	return resourceIDs, nil
}

// FetchResource fetches a transformation or library by type and ID
func (a *TransformationAdapter) FetchResource(ctx context.Context, resourceType, resourceID string) (any, error) {
	switch resourceType {
	case TransformationResourceType:
		return a.store.GetTransformation(ctx, resourceID)
	case LibraryResourceType:
		return a.store.GetLibrary(ctx, resourceID)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}
