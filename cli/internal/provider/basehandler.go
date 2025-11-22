package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

// ImportResourceInfo tracks metadata for resources that were imported from a remote system.
// It associates a local resource with its workspace context and remote identifier,
// enabling the system to maintain the relationship between local configuration and
// remote resources during import operations.
type ImportResourceInfo struct {
	WorkspaceId string
	RemoteId    string
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
	GetResourceMetadata() *RemoteResourceMetadata
}

// BaseHandler provides a generic, reusable foundation for implementing resource handlers
// across different resource types. It abstracts common lifecycle operations (load, validate,
// create, update, delete, import) while delegating resource-specific logic to the HandlerImpl.
//
// Type parameters:
//   - Spec: The configuration specification type (e.g., EventStreamSpec)
//   - Res: The resource data type representing user-defined configuration (e.g., EventStreamResource)
//   - State: The state type representing API response data (e.g., EventStreamState)
//   - Remote: The remote API response type implementing RemoteResource (e.g., *EventStreamSource)
//
// BaseHandler manages resource collections, import metadata, and provides type-safe CRUD
// operations with runtime type assertions. It bridges the gap between the generic Handler
// interface and strongly-typed implementations.
type BaseHandler[Spec any, Res any, State any, Remote RemoteResource] struct {
	resourceType       string
	specKind           string
	importMetadataName string
	resources          map[string]*Res
	importMetadata     map[string]*ImportResourceInfo
	Impl               HandlerImpl[Spec, Res, State, Remote]
}

func NewHandler[Spec any, Res any, State any, Remote RemoteResource](
	specKind string,
	resourceType string,
	importMetadataName string,
	impl HandlerImpl[Spec, Res, State, Remote]) *BaseHandler[Spec, Res, State, Remote] {
	return &BaseHandler[Spec, Res, State, Remote]{
		resourceType:       resourceType,
		specKind:           specKind,
		importMetadataName: importMetadataName,
		resources:          make(map[string]*Res),
		importMetadata:     make(map[string]*ImportResourceInfo),
		Impl:               impl,
	}
}

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
}

func (h *BaseHandler[Spec, Res, State, Remote]) GetResourceType() string {
	return h.resourceType
}

func (h *BaseHandler[Spec, Res, State, Remote]) GetSpecKind() string {
	return h.specKind
}

func (h *BaseHandler[Spec, Res, State, Remote]) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()

	remoteResources, err := h.Impl.LoadImportableResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading importable resources: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, remoteData := range remoteResources {
		metadata := (*remoteData).GetResourceMetadata()
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  metadata.Name,
			Scope: h.resourceType,
		})

		reference := fmt.Sprintf("#/%s/%s/%s", h.specKind, h.importMetadataName, externalID)

		if err != nil {
			return nil, fmt.Errorf("generating externalID for source '%s': %w", metadata.Name, err)
		}
		resourceMap[metadata.ID] = &resources.RemoteResource{
			ID:         metadata.ID,
			ExternalID: externalID,
			Reference:  reference,
			Data:       remoteData,
		}
	}

	collection.Set(h.resourceType, resourceMap)
	return collection, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) loadImportMetadata(fileMetadata map[string]any) error {
	metadata := specs.Metadata{}
	err := mapstructure.Decode(fileMetadata, &metadata)
	if err != nil {
		return fmt.Errorf("decoding import metadata: %w", err)
	}
	workspaces := metadata.Import.Workspaces
	for _, workspaceMetadata := range workspaces {
		workspaceId := workspaceMetadata.WorkspaceID
		resources := workspaceMetadata.Resources
		for _, resourceMetadata := range resources {
			h.importMetadata[resourceMetadata.LocalID] = &ImportResourceInfo{
				WorkspaceId: workspaceId,
				RemoteId:    resourceMetadata.RemoteID,
			}
		}
	}
	return nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	id, ok := s.Spec["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id not found in event stream source spec")
	}
	return &specs.ParsedSpec{ExternalIDs: []string{id}}, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) LoadSpec(path string, s *specs.Spec) error {
	spec := h.Impl.NewSpec()

	// Convert spec map to struct using mapstructure
	if err := mapstructure.Decode(s.Spec, spec); err != nil {
		return fmt.Errorf("converting spec: %w", err)
	}

	if err := h.Impl.ValidateSpec(spec); err != nil {
		return fmt.Errorf("validating spec: %w", err)
	}

	rs, err := h.Impl.ExtractResourcesFromSpec(path, spec)
	if err != nil {
		return fmt.Errorf("extracting resources from spec: %w", err)
	}
	for id, r := range rs {
		if _, ok := h.resources[id]; ok {
			return fmt.Errorf("a resource of type '%s' with id '%s' already exists", h.resourceType, id)
		}
		h.resources[id] = r
	}

	return h.loadImportMetadata(s.Metadata)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Validate(graph *resources.Graph) error {
	for _, source := range h.resources {
		if err := h.Impl.ValidateResource(source, graph); err != nil {
			return fmt.Errorf("validating event stream source spec: %w", err)
		}
	}
	return nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for resourceId, s := range h.resources {
		opts := []resources.ResourceOpts{
			resources.WithRawData(s),
		}
		if importMetadata, ok := h.importMetadata[resourceId]; ok {
			opts = append(opts, resources.WithResourceImportMetadata(importMetadata.RemoteId, importMetadata.WorkspaceId))
		}
		r := resources.NewResource(
			resourceId,
			h.resourceType,
			resources.ResourceData{}, // deprecated, will be removed
			[]string{},
			opts...,
		)
		result = append(result, r)
	}
	return result, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()

	remoteResources, err := h.Impl.LoadRemoteResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, remoteData := range remoteResources {
		metadata := (*remoteData).GetResourceMetadata()
		resourceMap[metadata.ID] = &resources.RemoteResource{
			ID:         metadata.ID,
			ExternalID: metadata.ExternalID,
			Data:       remoteData,
		}
	}

	collection.Set(h.resourceType, resourceMap)
	return collection, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	s := state.EmptyState()
	remoteResources := collection.GetAll(h.resourceType)

	for _, remoteResource := range remoteResources {
		// Type-safe cast - Remote is known via generics
		remote, ok := remoteResource.Data.(*Remote)
		if !ok {
			return nil, fmt.Errorf("invalid remote resource type for %s: expected %T, got %T",
				h.resourceType, (*Remote)(nil), remoteResource.Data)
		}

		// Call impl with typed remote
		inputRaw, outputRaw, err := h.Impl.MapRemoteToState(remote, collection)
		if err != nil {
			return nil, fmt.Errorf("mapping remote to state for %s: %w", remoteResource.ID, err)
		}

		// Skip if nil (convention: nil Resource = skip)
		if inputRaw == nil {
			continue
		}

		s.AddResource(&state.ResourceState{
			ID:        remoteResource.ExternalID,
			Type:      h.resourceType,
			InputRaw:  inputRaw,  // Type: *Res
			OutputRaw: outputRaw, // Type: *State
		})
	}
	return s, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) Create(ctx context.Context, rawData any) (any, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid resource data type. Found %v, expected %v", reflect.TypeOf(rawData), reflect.TypeOf((*Res)(nil)))
	}
	return h.Impl.Create(ctx, raw)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Update(ctx context.Context, newData any, oldData any, oldState any) (any, error) {
	newRaw, ok := newData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid resource data type. Found %v, expected %v", reflect.TypeOf(newData), reflect.TypeOf((*Res)(nil)))
	}

	oldRaw, ok := oldData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid old resource data type. Found %v, expected %v", reflect.TypeOf(oldData), reflect.TypeOf((*Res)(nil)))
	}

	oldRawState, ok := oldState.(*State)
	if !ok {
		return nil, fmt.Errorf("invalid old resource state data type. Found %v, expected %v", reflect.TypeOf(oldState), reflect.TypeOf((*State)(nil)))
	}
	return h.Impl.Update(ctx, newRaw, oldRaw, oldRawState)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Delete(ctx context.Context, ID string, oldData any, oldState any) error {
	oldRaw, ok := oldData.(*Res)
	if !ok {
		return fmt.Errorf("invalid old resource data type. Found %v, expected %v", reflect.TypeOf(oldData), reflect.TypeOf((*Res)(nil)))
	}

	oldRawState, ok := oldState.(*State)
	if !ok {
		return fmt.Errorf("invalid old resource state data type. Found %v, expected %v", reflect.TypeOf(oldState), reflect.TypeOf((*State)(nil)))
	}
	return h.Impl.Delete(ctx, ID, oldRaw, oldRawState)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Import(ctx context.Context, rawData any, remoteId string) (any, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid resource data type. Found %v, expected %v", reflect.TypeOf(rawData), reflect.TypeOf((*Res)(nil)))
	}
	return h.Impl.Import(ctx, raw, remoteId)
}

func (h *BaseHandler[Spec, Res, State, Remote]) CreatePropertyRef(
	urn string,
	extractor func(*State) (string, error),
) *resources.PropertyRef {
	return &resources.PropertyRef{
		URN: urn,
		Resolve: func(outputRaw any) (string, error) {
			typedState, ok := outputRaw.(*State)
			if !ok {
				expectedType := reflect.TypeOf((*State)(nil))
				actualType := reflect.TypeOf(outputRaw)
				return "", fmt.Errorf(
					"invalid state type for %s: expected %v, got %v",
					urn, expectedType, actualType,
				)
			}

			return extractor(typedState)
		},
	}
}
