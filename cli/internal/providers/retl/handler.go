package retl

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// resourceHandler defines the interface for type-specific resource handlers.
// Each resource type (e.g., SQL Model) must implement this interface to be
// managed by the RETL provider.
type resourceHandler interface {
	// LoadSpec loads and validates a resource specification from a file.
	// The path parameter specifies the location of the spec file, and s contains
	// the parsed spec data. Returns an error if the spec is invalid or cannot be loaded.
	LoadSpec(path string, s *specs.Spec) error

	// Validate performs validation of all loaded specs for this resource type.
	// This is called after all specs are loaded to ensure consistency and
	// validate cross-references between resources.
	Validate() error

	// GetResources returns all resources managed by this handler.
	// The returned resources will be added to the resource graph for
	// dependency resolution and state management.
	GetResources() ([]*resources.Resource, error)

	// Create creates a new resource with the given ID and data.
	// Returns the created resource's data or an error if creation fails.
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)

	// Update updates an existing resource identified by ID with new data.
	// The state parameter contains the current state of the resource.
	// Returns the updated resource's data or an error if update fails.
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)

	// Delete deletes an existing resource identified by ID.
	// The state parameter contains the current state of the resource.
	// Returns an error if deletion fails.
	Delete(ctx context.Context, ID string, state resources.ResourceData) error

	// List lists all resources managed by this handler.
	// The returned resources will be added to the resource graph for
	// dependency resolution and state management.
	List(ctx context.Context) ([]resources.ResourceData, error)

	// FetchImportData retrieves data for multiple resources to be imported.
	// This method fetches remote resources based on the provided import arguments
	// and prepares them for local import. It handles resource discovery and metadata collection.
	// Returns a list of import data structures or an error if fetching fails.
	FetchImportData(ctx context.Context, args importremote.ImportArgs) ([]importremote.ImportData, error)

	// Import updates the remote state to match the resource defined in YAML projects.
	// This method takes the local ID, resource data from YAML definitions, and import metadata
	// to align the remote resource with the local configuration.
	// Returns the processed resource data or an error if import fails.
	Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error)

	// Preview returns the preview results for a resource.
	// Returns:
	// - []string: column names
	// - map[string]any: contains result data with keys: "errorMessage", "rows", "rowCount", and "columns" (array of column info)
	// - error: any error that occurred
	Preview(ctx context.Context, ID string, data resources.ResourceData, limit int) ([]string, []map[string]any, error)
}
