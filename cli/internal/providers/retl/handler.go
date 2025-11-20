package retl

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// resourceHandler defines the interface for type-specific resource handlers.
// Each resource type (e.g., SQL Model) must implement this interface to be
// managed by the RETL provider.
type resourceHandler interface {
	// ParseSpec parses the spec generically for the resource type
	// and returns the data
	ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error)

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
	// The hasExternalId parameter is used to filter the resources by external ID.
	// The returned resources will be added to the resource graph for
	// dependency resolution and state management.
	List(ctx context.Context, hasExternalId *bool) ([]resources.ResourceData, error)

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
	Preview(ctx context.Context, ID string, data resources.ResourceData, limit int) ([]map[string]any, error)

	// LoadResourcesFromRemote loads all RETL resources from remote
	// Returns a collection of resources or an error if loading fails.
	LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error)

	// LoadStateFromResources reconstructs RETL state from loaded resources
	// Returns a state or an error if loading fails.
	LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error)

	// LoadImportable loads all importable resources from remote
	// The idNamer is used to generate unique IDs for the resources.
	// Returns a collection of resources or an error if loading fails.
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error)

	// FormatForExport formats the resources for export
	// The idNamer is used to generate unique IDs for the resources.
	// The inputResolver is used to resolve references to other resources.
	// Returns a list of importable entities or an error if formatting fails.
	FormatForExport(ctx context.Context, collection *resources.ResourceCollection, idNamer namer.Namer, inputResolver resolver.ReferenceResolver) ([]importremote.FormattableEntity, error)
}
