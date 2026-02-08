package testorchestrator

import (
	"context"
	"fmt"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// StagingManager handles uploading transformations and libraries to the workspace as unpublished versions
type StagingManager struct {
	store transformations.TransformationStore
}

// NewStagingManager creates a new staging manager
func NewStagingManager(store transformations.TransformationStore) *StagingManager {
	return &StagingManager{
		store: store,
	}
}

// StageTransformation uploads a transformation to the workspace as an unpublished version.
// It creates a new transformation if it doesn't exist, or updates the existing transformation.
// Returns the versionID of the uploaded transformation.
func (s *StagingManager) StageTransformation(ctx context.Context, transformation *model.TransformationResource, remoteState *state.State) (string, error) {
	// Check if transformation exists in remote state
	transformationURN := fmt.Sprintf("%s::%s", transformation.ID, "transformation")
	remoteResource := remoteState.GetResource(transformationURN)

	if remoteResource == nil {
		// Transformation doesn't exist remotely - create new
		req := &transformations.CreateTransformationRequest{
			Name:        transformation.Name,
			Description: transformation.Description,
			Code:        transformation.Code,
			Language:    transformation.Language,
			ExternalID:  transformation.ID,
		}
		result, err := s.store.CreateTransformation(ctx, req, false) // publish=false for unpublished version
		if err != nil {
			return "", fmt.Errorf("creating transformation %s: %w", transformation.ID, err)
		}
		return result.VersionID, nil
	}

	// Transformation exists remotely - update it
	// Extract remote ID from state output
	remoteID, ok := remoteResource.Output["id"].(string)
	if !ok || remoteID == "" {
		return "", fmt.Errorf("transformation %s exists in remote state but has no valid remote ID", transformation.ID)
	}

	updateReq := &transformations.UpdateTransformationRequest{
		Name:        transformation.Name,
		Description: transformation.Description,
		Code:        transformation.Code,
		Language:    transformation.Language,
	}

	result, err := s.store.UpdateTransformation(ctx, remoteID, updateReq, false) // publish=false for unpublished version
	if err != nil {
		return "", fmt.Errorf("updating transformation %s: %w", transformation.ID, err)
	}

	return result.VersionID, nil
}

// StageLibrary uploads a library to the workspace as an unpublished version.
// It creates a new library if it doesn't exist, or updates the existing library.
// Returns the versionID of the uploaded library.
func (s *StagingManager) StageLibrary(ctx context.Context, library *model.LibraryResource, remoteState *state.State) (string, error) {
	// Check if library exists in remote state
	libraryURN := fmt.Sprintf("%s::%s", library.ID, "library")
	remoteResource := remoteState.GetResource(libraryURN)

	if remoteResource == nil {
		// Library doesn't exist remotely - create new
		req := &transformations.CreateLibraryRequest{
			Name:        library.Name,
			Description: library.Description,
			Code:        library.Code,
			Language:    library.Language,
			ExternalID:  library.ID,
		}
		result, err := s.store.CreateLibrary(ctx, req, false) // publish=false for unpublished version
		if err != nil {
			return "", fmt.Errorf("creating library %s: %w", library.ID, err)
		}
		return result.VersionID, nil
	}

	// Library exists remotely - update it
	// Extract remote ID from state output
	remoteID, ok := remoteResource.Output["id"].(string)
	if !ok || remoteID == "" {
		return "", fmt.Errorf("library %s exists in remote state but has no valid remote ID", library.ID)
	}

	updateReq := &transformations.UpdateLibraryRequest{
		Name:        library.Name,
		Description: library.Description,
		Code:        library.Code,
		Language:    library.Language,
	}

	result, err := s.store.UpdateLibrary(ctx, remoteID, updateReq, false) // publish=false for unpublished version
	if err != nil {
		return "", fmt.Errorf("updating library %s: %w", library.ID, err)
	}

	return result.VersionID, nil
}
