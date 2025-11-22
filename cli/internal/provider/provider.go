package provider

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

type composable interface {
	// GetSupportedTypes returns the list of resource types that this provider can manage.
	// This is used to route resource operations to the appropriate provider based on type.
	GetSupportedTypes() []string
}

// SpecLoader handles loading and parsing of resource specifications from configuration files.
// It is responsible for transforming declarative configuration into an internal resource graph
// that can be used for planning and applying changes.
type SpecLoader interface {
	// GetSupportedKinds returns the list of specification kinds that this provider can load.
	// Kinds represent different types of configuration entities in spec files (e.g., resource definitions).
	// This is used to route spec loading to the appropriate provider.
	GetSupportedKinds() []string

	// LoadSpec loads and processes a specification from the given path.
	// It populates the provider's internal state with the resource definitions from the spec.
	// This method is called for each spec file during project loading.
	LoadSpec(path string, s *specs.Spec) error

	// ParseSpec parses a specification without fully loading it into the provider's state.
	// It extracts metadata such as external IDs that may be referenced by other specs.
	// This allows for two-phase loading where references can be validated before full processing.
	ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error)

	// GetResourceGraph returns the complete resource graph built from all loaded specifications.
	// The graph contains all resources and their dependencies, ready for validation and planning.
	GetResourceGraph() (*resources.Graph, error)
}

// Validator performs provider-specific validation on a resource graph.
// It ensures that resources conform to the provider's requirements and constraints.
type Validator interface {
	// Validate checks the resource graph for provider-specific errors.
	// Providers should validate their own resources but may leverage the full graph
	// for cross-resource validations (e.g., checking references, ensuring dependencies exist).
	// Returns an error if validation fails.
	Validate(graph *resources.Graph) error
}

// ManagedRemoteResourceLoader loads resources from a remote system that are currently
// managed by this tool (i.e., they exist in the local state).
type ManagedRemoteResourceLoader interface {
	// LoadResourcesFromRemote fetches all managed resources from the remote system.
	// These resources represent the current state of the remote infrastructure and
	// are used for drift detection and synchronization planning.
	LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error)
}

// UnmanagedRemoteResourceLoader loads resources from a remote system that are not yet
// managed by this tool but are available for import.
type UnmanagedRemoteResourceLoader interface {
	// LoadImportable fetches resources from the remote system that exist but are not
	// currently managed in the local project. These resources can be imported into
	// the project's configuration.
	// The idNamer is used to generate unique IDs for the importable resources,
	// avoiding conflicts with existing resources.
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error)
}

// RemoteResourceLoader combines the ability to load both managed and unmanaged resources
// from a remote system. This is the complete interface for remote resource discovery.
type RemoteResourceLoader interface {
	ManagedRemoteResourceLoader
	UnmanagedRemoteResourceLoader
}

// StateLoader converts remote resources into a state representation.
// State is used to track what resources are managed and their last known configuration.
type StateLoader interface {
	// LoadStateFromResources transforms a collection of remote resources into state format.
	// The state contains normalized resource data including IDs, types, inputs, and dependencies.
	// This state is persisted and used for planning future changes.
	LoadStateFromResources(ctx context.Context, resources *resources.ResourceCollection) (*state.State, error)
}

// LifecycleManager handles the creation, modification, deletion, and import of resources
// in the remote system. It executes the planned operations against the actual infrastructure.
type LifecycleManager interface {
	// Create provisions a new resource in the remote system.
	// Returns the actual resource data after creation, which may include system-generated
	// fields such as timestamps or auto-assigned identifiers.
	Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)

	// Update modifies an existing resource in the remote system.
	// The 'data' parameter contains the desired configuration, while 'state' contains
	// the last known configuration. Returns the actual resource data after the update.
	Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)

	// Delete removes a resource from the remote system.
	// The 'state' parameter contains the last known configuration, which may be needed
	// to properly delete the resource.
	Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error

	// Import brings an existing remote resource under management.
	// This operation associates an unmanaged remote resource (identified by remoteId)
	// with a local resource definition. The workspaceId provides context for multi-workspace
	// scenarios. Returns the actual resource data from the remote system.
	Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error)
}

// Exporter transforms resources into a format suitable for generating configuration files.
// This is used during import operations to create spec files from existing remote resources.
type Exporter interface {
	// FormatForExport converts a resource collection into formattable entities that can be
	// written as configuration files.
	//
	// The idNamer generates human-readable identifiers for resources in the generated files.
	// The resolver handles reference resolution, converting remote IDs into local references
	// where appropriate (e.g., referencing other imported or existing resources).
	//
	// Returns a slice of entities that can be formatted and written to disk.
	FormatForExport(
		ctx context.Context,
		collection *resources.ResourceCollection,
		idNamer namer.Namer,
		resolver resolver.ReferenceResolver,
	) ([]writer.FormattableEntity, error)
}

// Provider is the complete interface that all providers must implement.
// It combines all the individual capabilities required for full resource lifecycle management:
//
//   - Discovery: Identifying what resource types and spec kinds are supported
//   - Loading: Reading and parsing configuration files into resource graphs
//   - Validation: Ensuring resource configurations meet provider-specific requirements
//   - Remote interaction: Fetching both managed and importable resources from remote systems
//   - State management: Converting remote resources into state format for tracking
//   - Lifecycle: Creating, updating, deleting, and importing resources in the remote system
//   - Export: Generating configuration files from existing remote resources
//
// Providers act as adapters between the generic infrastructure management framework
// and specific resource types or backend systems.
type Provider interface {
	composable
	SpecLoader
	Validator
	RemoteResourceLoader
	StateLoader
	LifecycleManager
	Exporter
}
