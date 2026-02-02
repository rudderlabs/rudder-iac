package testorchestrator

import (
	"context"
	"fmt"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// StagingManager handles uploading libraries to the workspace as unpublished versions
type StagingManager struct {
	store transformations.TransformationStore
}

// NewStagingManager creates a new staging manager
func NewStagingManager(store transformations.TransformationStore) *StagingManager {
	return &StagingManager{
		store: store,
	}
}

// UploadLibrary uploads a library to the workspace as an unpublished version.
// It creates a new library if it doesn't exist, or updates the existing library.
// Returns the versionID of the uploaded library.
func (s *StagingManager) UploadLibrary(ctx context.Context, library *model.LibraryResource, remoteState *state.State) (string, error) {
	// Check if library exists in remote state
	libraryURN := fmt.Sprintf("%s::%s", library.ID, "library")
	remoteResource := remoteState.GetResource(libraryURN)

	req := buildLibraryRequest(library)

	if remoteResource == nil {
		// Library doesn't exist remotely - create new
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

	// Convert CreateLibraryRequest to UpdateLibraryRequest (same fields)
	updateReq := &transformations.UpdateLibraryRequest{
		Name:        req.Name,
		Description: req.Description,
		Code:        req.Code,
		Language:    req.Language,
	}

	result, err := s.store.UpdateLibrary(ctx, remoteID, updateReq, false) // publish=false for unpublished version
	if err != nil {
		return "", fmt.Errorf("updating library %s: %w", library.ID, err)
	}

	return result.VersionID, nil
}

// buildLibraryRequest creates a CreateLibraryRequest from a LibraryResource
func buildLibraryRequest(library *model.LibraryResource) *transformations.CreateLibraryRequest {
	return &transformations.CreateLibraryRequest{
		Name:        library.Name,
		Description: library.Description,
		Code:        library.Code,
		Language:    library.Language,
		ExternalID:  library.ID, // Use project ID as external ID
	}
}
