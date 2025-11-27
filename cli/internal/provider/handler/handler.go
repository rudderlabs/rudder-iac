package handler

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// HandlerImpl defines the resource-specific operations that each handler implementation must
// provide to work with BaseHandler. It separates resource-specific business logic from the
// common handler infrastructure, following the strategy pattern.
//
// Implementations must provide:
//   - Spec lifecycle: creation, validation, and resource extraction from configuration files
//   - Resource validation: ensuring resources are valid within the dependency graph
//   - Remote operations: loading resources from remote APIs, both for sync and import scenarios
//   - State mapping: converting remote API responses to local resource and state representations
//   - CRUD operations: creating, updating, importing, and deleting resources via API calls
//
// The MapRemoteToState method can return (nil, nil, nil) to skip resources that should not
// be included in state (e.g., resources without external IDs). The urnResolver parameter
// enables resolving cross-resource references during state mapping.
type HandlerImpl[Spec any, Res any, State any, Remote RemoteResource] interface {
	// NewSpec creates a new instance of the Spec type with zero values.
	// This factory method enables BaseHandler to instantiate spec objects
	// during configuration file parsing without knowing the concrete type.
	NewSpec() *Spec

	// ValidateSpec checks that the parsed specification contains valid values
	// and required fields. It should return descriptive errors for any validation
	// failures. This is called after decoding the YAML/JSON spec but before
	// extracting resources from it.
	ValidateSpec(spec *Spec) error

	// ExtractResourcesFromSpec parses a validated spec and extracts individual
	// resource instances from it, returning them as a map keyed by resource ID.
	// The path parameter provides the file path for error reporting and context.
	// Multiple resources may be extracted from a single spec file (e.g., a spec
	// containing multiple event definitions).
	//
	// NOTE: path should be part of the spec
	ExtractResourcesFromSpec(path string, spec *Spec) (map[string]*Res, error)

	// ValidateResource performs validation on a single resource within the context
	// of the full dependency graph. This enables cross-resource validation such as
	// checking that referenced resources exist (e.g., verifying a tracking plan
	// URN references an actual tracking plan resource). The graph provides access
	// to all loaded resources across all types.
	ValidateResource(resource *Res, graph *resources.Graph) error

	// LoadRemoteResources fetches all resources of this type from the remote API
	// that have external IDs (i.e., resources managed by this IaC system).
	// This is used during sync operations to build the current remote state.
	// Resources without external IDs should be filtered out as they are not
	// managed by the configuration files.
	LoadRemoteResources(ctx context.Context) ([]*Remote, error)

	// LoadImportableResources fetches all resources of this type from the remote API,
	// including those without external IDs. This is used during import operations
	// to discover resources that can be brought under IaC management. The returned
	// resources will be presented to users as candidates for import.
	LoadImportableResources(ctx context.Context) ([]*Remote, error)

	// MapRemoteToState converts a remote API resource into the corresponding Res
	// (input/configuration) and State (output) representations used internally.
	// The urnResolver enables resolving references to other resources by their IDs.
	//
	// Return (nil, nil, nil) to skip a resource (e.g., when it lacks required fields
	// like external IDs). This is a convention for filtering resources during state
	// loading without treating it as an error.
	MapRemoteToState(remote *Remote, urnResolver URNResolver) (*Res, *State, error)

	// Create provisions a new resource in the remote system using the provided
	// configuration data. It returns the state (output) data from the API response,
	// which typically includes server-assigned IDs, timestamps, and computed fields.
	// This is called when a resource exists in configuration but not in remote state.
	Create(ctx context.Context, data *Res) (*State, error)

	// Update modifies an existing remote resource to match the new configuration.
	// It receives both the new desired state (newData) and the current state
	// (oldData, oldState) to enable delta-based updates or conditional logic.
	// Returns the updated state from the API response.
	Update(ctx context.Context, newData *Res, oldData *Res, oldState *State) (*State, error)

	// Import associates an existing remote resource with a local configuration,
	// bringing it under IaC management. The remoteId identifies the resource in
	// the remote system. This typically sets the external ID on the remote resource
	// and returns its state. Import is used when users want to manage pre-existing
	// resources through configuration files.
	Import(ctx context.Context, data *Res, remoteId string) (*State, error)

	// Delete removes a resource from the remote system. It receives the resource ID,
	// the last known configuration (oldData), and state (oldState) to enable cleanup
	// operations that may depend on the resource's current configuration. Some
	// implementations may need state information to properly delete dependent resources
	// or perform cascading deletes.
	Delete(ctx context.Context, id string, oldData *Res, oldState *State) error

	FormatForExport(
		ctx context.Context,
		collection *resources.ResourceCollection,
		idNamer namer.Namer,
		inputResolver resolver.ReferenceResolver,
	) ([]writer.FormattableEntity, error)
}

// URNResolver provides URN (Uniform Resource Name) resolution from remote resource IDs.
// Implementations (such as ResourceCollection) use this interface to resolve cross-resource
// references by translating a resource type and ID into a URN that can be used to reference
// the resource in configuration files and state. This enables resources to reference each
// other using stable identifiers.
type URNResolver interface {
	GetURNByID(resourceType string, id string) (string, error)
}

// RemoteResourceMetadata contains the standard metadata fields for a resource fetched from
// a remote API. The ID represents the remote system's internal identifier, ExternalID is the
// user-facing identifier used in configuration files, WorkspaceID provides the workspace
// context, and Name is a human-readable descriptor. This structure standardizes how remote
// resources expose their identifying information across different resource types.
type RemoteResourceMetadata struct {
	ID          string
	ExternalID  string
	WorkspaceID string
	Name        string
}

// RemoteResource defines the minimal interface that any remote resource type must implement
// to be compatible with BaseHandler's generic type system. By requiring only metadata access,
// this interface allows API client types (e.g., EventStreamSource, RETLSource) to work with
// BaseHandler without extensive modification, while ensuring consistent access to identifying
// information needed for resource management operations.
type RemoteResource interface {
	GetResourceMetadata() RemoteResourceMetadata
}
