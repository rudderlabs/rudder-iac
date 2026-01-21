package provider

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// TypeProvider defines the interface for providers to declare what resource types
// and spec kinds they support. This information is used to route operations
// to the appropriate provider based on type or kind.
type TypeProvider interface {
	// SupportedKinds returns the list of YAML spec kinds that this provider can load.
	// This is used to route spec loading to the appropriate provider.
	SupportedKinds() []string

	// SupportedTypes returns the list of resource types that this provider can manage.
	// This is used to route resource operations to the appropriate provider based on type.
	SupportedTypes() []string
}

type SpecLoader interface {
	// LoadSpec loads and processes a YAML spec from the given path.
	// It populates the provider's internal state with the resource definitions from the spec.
	//
	// NOTE: To avoid having stateful providers, it might be better to have this method
	// return the loaded resources as well as any parsed metadata. The loaded resources
	// (captured as part of the resource graph) already contain the same information that would
	// otherwise be captured in the internal provider states. This would avoid
	// the interface exposing multiple methods (load, parsing, getting resource graph) considering
	// that these are all steps of the same load spec process.
	LoadSpec(path string, s *specs.Spec) error

	// LoadLegacySpec loads and processes a legacy YAML spec (rudder/0.1) from the given path.
	// It populates the provider's internal state with the resource definitions from the legacy spec.
	// This is used during migration to load old specs before converting them.
	LoadLegacySpec(path string, s *specs.Spec) error

	// ParseSpec parses a specification without fully loading it into the provider's state.
	// It extracts metadata such as external IDs that may be referenced by other specs.
	// This allows for two-phase loading where references can be validated before full processing.
	//
	// NOTE: Assuming LoadSpec would return the resource graph that corresponds to the spec, this method could be replaced by
	// metadata of the resource graph returned by LoadSpec.
	ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error)

	// GetResourceGraph returns the complete resource graph built from all loaded specifications.
	// The graph contains all resources and their dependencies, ready for validation and planning.
	//
	// NOTE: Similar to ParseSpec, if LoadSpec returned the resource graph, this method could be eliminated.
	// However, the current design allows for loading multiple specs before retrieving the full graph.
	// An alternative approach would be for the [project.Project] to aggregate graphs from multiple providers,
	// during the loading phase.
	ResourceGraph() (*resources.Graph, error)
}

// Validator performs provider-specific validation on a resource graph.
// It ensures that resources conform to the provider's requirements and constraints.
type Validator interface {
	// Validate checks the resource graph for provider-specific errors.
	// Providers should validate their own resources but may leverage the full graph
	// for cross-resource validations (e.g., checking references, ensuring dependencies exist).
	// Returns an error if validation fails.
	//
	// NOTE: A possible improvement could be to have this method return a list of validation errors
	// instead of a single error. This would allow reporting multiple issues in one pass,
	// providing more comprehensive feedback to the user. The list could also include warnings for non-critical issues.
	Validate(graph *resources.Graph) error
}

// ManagedRemoteResourceLoader loads resources from a remote system that are currently
// managed by this tool (i.e., they exist in the local state).
type ManagedRemoteResourceLoader interface {
	// LoadResourcesFromRemote fetches all managed resources from a remote backend,
	// typically by calling the RudderStack API or other relevant service endpoints.
	// These resources represent the current state of the remote infrastructure and
	// are used for drift detection and synchronization planning
	// (by converting them into state and getting the resource graph),
	// as well as for directly interacting with the remote backend, e.g for listing resources.
	LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error)
}

// UnmanagedRemoteResourceLoader loads resources from a remote system that are not yet
// managed by this tool but are available for import.
type UnmanagedRemoteResourceLoader interface {
	// LoadImportable fetches resources from the remote system that exist but are not
	// currently managed in the local project. These resources can be imported into
	// the project's configuration.
	// The idNamer is used to generate unique IDs for the importable resources,
	// avoiding conflicts with existing resources.
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error)
}

// RemoteResourceLoader combines the ability to load both managed and unmanaged resources
// from a remote system. This is the complete interface for remote resource discovery.
//
// NOTE: In the future we could unify the two interfaces in one LoadRemoteResources method
// that accepts a parameter to indicate whether to load managed, unmanaged, or both types of resources.
// The majority of providers that implement both interfaces have very similar logic in both methods.
type RemoteResourceLoader interface {
	ManagedRemoteResourceLoader
	UnmanagedRemoteResourceLoader
}

// StateLoader converts remote resources into a state representation.
// State is used to track what resources are managed and their last known configuration.
type StateLoader interface {
	// MapRemoteToState transforms a collection of remote resources into a [state.State] format.
	MapRemoteToState(resources *resources.RemoteResources) (*state.State, error)
}

// LifecycleManager handles the creation, modification, deletion, and import of resources
// in the remote backend.
type LifecycleManager interface {
	CreateRaw(ctx context.Context, data *resources.Resource) (any, error)
	UpdateRaw(ctx context.Context, data *resources.Resource, oldData any, oldState any) (any, error)
	DeleteRaw(ctx context.Context, ID string, resourceType string, oldData any, oldState any) error
	ImportRaw(ctx context.Context, data *resources.Resource, remoteId string) (any, error)

	// Create provisions a new resource in the remote system.
	// Returns the actual resource data after creation, which may include system-generated
	// fields such as timestamps or auto-assigned identifiers.
	Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)

	// Update modifies an existing resource in the remote backend.
	// The 'data' parameter contains the desired configuration, while 'state' contains
	// the last known configuration. Returns the actual resource data after the update.
	Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)

	// Delete removes a resource from the remote backend.
	// The 'state' parameter contains the last known configuration, which may be needed
	// to properly delete the resource.
	Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error

	// Import brings an existing remote resource under management.
	// This operation associates an unmanaged remote resource (identified by remoteId)
	// with a local resource definition. Returns the actual resource data from the remote backend.
	//
	// NOTE: Current implementations assume the import is not only associating the resource,
	// but also updating it to match the local configuration. This should be clarified in the future,
	// potentially by separating the association and update into distinct steps, so as not repeat the update logic here.
	Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error)
}

// Exporter transforms resources into a format suitable for generating files.
// These are typically YAML specs but could include other formats as necessary, e.g SQL or JavaScript.
// This is used during import operations to create spec files, and other related files, from existing remote resources.
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
		collection *resources.RemoteResources,
		idNamer namer.Namer,
		resolver resolver.ReferenceResolver,
	) ([]writer.FormattableEntity, error)
}

// SpecMigrator handles migration of project specifications from one version to another.
type SpecMigrator interface {
	// MigrateSpec migrates project specifications from rudder/0.1 to rudder/1.
	// This method transforms the existing project configuration to the new spec version. s contains the spec to migrate.
	// Returns the migrated spec or an error.
	MigrateSpec(s *specs.Spec) (*specs.Spec, error)
}

// RuleProvider is an optional interface that providers can implement
// to contribute validation rules. Providers aggregate rules from their
// handlers (if using BaseProvider pattern) or define them directly.
//
// Rules are collected from providers and registered in the global Registry
// by the validation engine. Syntactic rules run before resource graph
// construction, while semantic rules run after.
type RuleProvider interface {
	// SyntacticRules returns rules that validate spec structure and format
	// before resource graph construction. These rules receive ValidationContext
	// with Graph set to nil.
	//
	// Rule IDs should follow convention: "<provider>/<kind>/<rule-name>"
	// Example: "datacatalog/properties/unique-name"
	SyntacticRules() []rules.Rule

	// SemanticRules returns rules that validate cross-resource relationships
	// and business logic after resource graph construction. These rules receive
	// ValidationContext with Graph populated.
	//
	// Rule IDs should follow convention: "<provider>/<kind>/<rule-name>"
	// Example: "datacatalog/events/valid-property-ref"
	SemanticRules() []rules.Rule
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
//   - Migration: Migrating project specifications from one version to another
//   - Rules: Providing syntactic and semantic validation rules for resources
//
// Providers act as adapters between the generic infrastructure management framework
// and specific resource types or backend systems.
type Provider interface {
	TypeProvider
	SpecLoader
	Validator
	RemoteResourceLoader
	StateLoader
	LifecycleManager
	Exporter
	SpecMigrator
	RuleProvider
}
