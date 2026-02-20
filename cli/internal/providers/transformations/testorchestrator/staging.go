package testorchestrator

import (
	"context"
	"fmt"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// StageTransformation uploads a transformation to the workspace as an unpublished version.
// It creates a new transformation if it doesn't exist, or updates the existing transformation.
// Returns the versionID of the uploaded transformation.
func StageTransformation(
	ctx context.Context,
	store transformations.TransformationStore,
	transformation *model.TransformationResource,
	remoteResource *state.ResourceState,
) (string, error) {
	if remoteResource == nil {
		// Transformation doesn't exist remotely - create new
		req := &transformations.CreateTransformationRequest{
			Name:       transformation.Name,
			Code:       transformation.Code,
			Language:   transformation.Language,
			ExternalID: transformation.ID,
		}
		result, err := store.CreateTransformation(ctx, req, false)
		if err != nil {
			return "", fmt.Errorf("creating transformation %s: %w", transformation.ID, err)
		}
		return result.VersionID, nil
	}

	// Transformation exists remotely - update it
	// Extract remote ID from state output
	transState, ok := remoteResource.OutputRaw.(*model.TransformationState)
	if !ok || transState.ID == "" {
		return "", fmt.Errorf("transformation %s exists in remote state but has no valid remote ID", transformation.ID)
	}

	updateReq := &transformations.UpdateTransformationRequest{
		Name:     transformation.Name,
		Code:     transformation.Code,
		Language: transformation.Language,
	}

	result, err := store.UpdateTransformation(ctx, transState.ID, updateReq, false)
	if err != nil {
		return "", fmt.Errorf("updating transformation %s: %w", transformation.ID, err)
	}

	return result.VersionID, nil
}

// StageLibrary uploads a library to the workspace as an unpublished version.
// It creates a new library if it doesn't exist, or updates the existing library.
// Returns the versionID of the uploaded library.
func StageLibrary(
	ctx context.Context,
	store transformations.TransformationStore,
	library *model.LibraryResource,
	remote *state.ResourceState,
) (string, error) {

	if remote == nil {
		// Library doesn't exist remotely - create new
		req := &transformations.CreateLibraryRequest{
			Name:        library.Name,
			Description: library.Description,
			Code:        library.Code,
			Language:    library.Language,
			ExternalID:  library.ID,
		}
		result, err := store.CreateLibrary(ctx, req, false)
		if err != nil {
			return "", fmt.Errorf("creating library %s: %w", library.ID, err)
		}
		return result.VersionID, nil
	}

	// Library exists remotely - update it
	libState, ok := remote.OutputRaw.(*model.LibraryState)
	if !ok || libState.ID == "" {
		return "", fmt.Errorf("library %s exists in remote state but has no valid remote ID", library.ID)
	}

	updateReq := &transformations.UpdateLibraryRequest{
		Name:     library.Name,
		Code:     library.Code,
		Language: library.Language,
	}

	result, err := store.UpdateLibrary(ctx, libState.ID, updateReq, false)
	if err != nil {
		return "", fmt.Errorf("updating library %s: %w", library.ID, err)
	}

	return result.VersionID, nil
}
