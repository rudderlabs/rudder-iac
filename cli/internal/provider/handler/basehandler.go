package handler

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

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
	metadata       HandlerMetadata
	resources      map[string]*Res
	importMetadata map[string]*importResourceInfo
	Impl           HandlerImpl[Spec, Res, State, Remote]
}

type importResourceInfo struct {
	WorkspaceId string
	RemoteId    string
}

func NewHandler[Spec any, Res any, State any, Remote RemoteResource](
	impl HandlerImpl[Spec, Res, State, Remote],
) *BaseHandler[Spec, Res, State, Remote] {
	m := impl.Metadata()
	return &BaseHandler[Spec, Res, State, Remote]{
		metadata:       m,
		resources:      make(map[string]*Res),
		importMetadata: make(map[string]*importResourceInfo),
		Impl:           impl,
	}
}

func (h *BaseHandler[Spec, Res, State, Remote]) ResourceType() string {
	return h.metadata.ResourceType
}

func (h *BaseHandler[Spec, Res, State, Remote]) SpecKind() string {
	return h.metadata.SpecKind
}

func (h *BaseHandler[Spec, Res, State, Remote]) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()

	remoteResources, err := h.Impl.LoadImportableResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading importable resources: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, remoteData := range remoteResources {
		metadata := (*remoteData).Metadata()
		externalID, err := idNamer.Name(namer.ScopeName{
			Name:  metadata.Name,
			Scope: h.metadata.ResourceType,
		})

		reference := fmt.Sprintf("#/%s/%s/%s", h.metadata.SpecKind, h.metadata.SpecMetadataName, externalID)

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

	collection.Set(h.metadata.ResourceType, resourceMap)
	return collection, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) loadImportMetadata(m *specs.WorkspacesImportMetadata) error {
	workspaces := m.Workspaces
	for _, workspaceMetadata := range workspaces {
		workspaceId := workspaceMetadata.WorkspaceID
		resources := workspaceMetadata.Resources
		for _, resourceMetadata := range resources {
			if err := resourceMetadata.Validate(); err != nil {
				return fmt.Errorf("invalid import metadata for workspace '%s': %w", workspaceId, err)
			}
			if resourceMetadata.URN == "" {
				return fmt.Errorf("urn field is required for import metadata in workspace '%s' (local_id is not supported)", workspaceId)
			}
			h.importMetadata[resourceMetadata.URN] = &importResourceInfo{
				WorkspaceId: workspaceId,
				RemoteId:    resourceMetadata.RemoteID,
			}
		}
	}
	return nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	resourceType := h.metadata.ResourceType

	// First, try to extract a single "id" field
	if id, ok := s.Spec["id"].(string); ok {
		return &specs.ParsedSpec{
			URNs: []specs.URNEntry{{
				URN:             resources.URN(id, resourceType),
				JSONPointerPath: "/spec/id",
			}},
		}, nil
	}

	// If the spec has a single field that is an array, extract IDs from array elements
	if len(s.Spec) == 1 {
		for specKey, value := range s.Spec {
			if arr, ok := value.([]any); ok {
				urnEntries := make([]specs.URNEntry, 0, len(arr))
				for i, item := range arr {
					if itemMap, ok := item.(map[string]any); ok {
						if id, ok := itemMap["id"].(string); ok {
							urnEntries = append(urnEntries, specs.URNEntry{
								URN:             resources.URN(id, resourceType),
								JSONPointerPath: fmt.Sprintf("/spec/%s/%d/id", specKey, i),
							})
						} else {
							return nil, fmt.Errorf("array item at index %d does not have an 'id' field", i)
						}
					} else {
						return nil, fmt.Errorf("array item at index %d is not a map", i)
					}
				}
				return &specs.ParsedSpec{URNs: urnEntries}, nil
			}
		}
	}

	return nil, fmt.Errorf("id not found in spec")
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
			return fmt.Errorf("a resource of type '%s' with id '%s' already exists", h.metadata.ResourceType, id)
		}
		h.resources[id] = r
	}

	commonMetadata, err := s.CommonMetadata()
	if err != nil {
		return fmt.Errorf("getting common metadata: %w", err)
	}

	if commonMetadata.Import != nil {
		if err := h.loadImportMetadata(commonMetadata.Import); err != nil {
			return fmt.Errorf("loading import metadata: %w", err)
		}
	}

	return nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) Validate(graph *resources.Graph) error {
	for _, source := range h.resources {
		if err := h.Impl.ValidateResource(source, graph); err != nil {
			return fmt.Errorf("validating event stream source spec: %w", err)
		}
	}
	return nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) Resources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for resourceId, s := range h.resources {
		opts := []resources.ResourceOpts{
			resources.WithRawData(s),
		}
		// Construct URN to look up import metadata (keyed by URN, not resourceId)
		urn := resources.URN(resourceId, h.metadata.ResourceType)
		if importMetadata, ok := h.importMetadata[urn]; ok {
			opts = append(opts, resources.WithResourceImportMetadata(importMetadata.RemoteId, importMetadata.WorkspaceId))
		}
		r := resources.NewResource(
			resourceId,
			h.metadata.ResourceType,
			resources.ResourceData{}, // deprecated, will be removed
			[]string{},
			opts...,
		)
		result = append(result, r)
	}
	return result, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()

	remoteResources, err := h.Impl.LoadRemoteResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	resourceMap := make(map[string]*resources.RemoteResource)
	for _, remoteData := range remoteResources {
		metadata := (*remoteData).Metadata()
		resourceMap[metadata.ID] = &resources.RemoteResource{
			ID:         metadata.ID,
			ExternalID: metadata.ExternalID,
			Data:       remoteData,
		}
	}

	collection.Set(h.metadata.ResourceType, resourceMap)
	return collection, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	s := state.EmptyState()
	remoteResources := collection.GetAll(h.metadata.ResourceType)

	for _, remoteResource := range remoteResources {
		// Type-safe cast - Remote is known via generics
		remote, ok := remoteResource.Data.(*Remote)
		if !ok {
			return nil, &ErrInvalidDataType{
				Expected: (*Remote)(nil),
				Actual:   remoteResource.Data,
			}
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
			Type:      h.metadata.ResourceType,
			InputRaw:  inputRaw,  // Type: *Res
			OutputRaw: outputRaw, // Type: *State
		})
	}
	return s, nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) Create(ctx context.Context, rawData any) (any, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, &ErrInvalidDataType{Expected: (*Res)(nil), Actual: rawData}
	}
	return h.Impl.Create(ctx, raw)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Update(ctx context.Context, newData any, oldData any, oldState any) (any, error) {
	newRaw, ok := newData.(*Res)
	if !ok {
		return nil, &ErrInvalidDataType{Expected: (*Res)(nil), Actual: newData}
	}

	oldRaw, ok := oldData.(*Res)
	if !ok {
		return nil, &ErrInvalidDataType{Expected: (*Res)(nil), Actual: oldData}
	}

	oldRawState, ok := oldState.(*State)
	if !ok {
		return nil, &ErrInvalidDataType{Expected: (*State)(nil), Actual: oldState}
	}
	return h.Impl.Update(ctx, newRaw, oldRaw, oldRawState)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Delete(ctx context.Context, ID string, oldData any, oldState any) error {
	oldRaw, ok := oldData.(*Res)
	if !ok {
		return &ErrInvalidDataType{Expected: (*Res)(nil), Actual: oldData}
	}

	oldRawState, ok := oldState.(*State)
	if !ok {
		return &ErrInvalidDataType{Expected: (*State)(nil), Actual: oldState}
	}
	return h.Impl.Delete(ctx, ID, oldRaw, oldRawState)
}

func (h *BaseHandler[Spec, Res, State, Remote]) Import(ctx context.Context, rawData any, remoteId string) (any, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, &ErrInvalidDataType{Expected: (*Res)(nil), Actual: rawData}
	}
	return h.Impl.Import(ctx, raw, remoteId)
}

func (h *BaseHandler[Spec, Res, State, Remote]) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	all := collection.GetAll(h.metadata.ResourceType)
	if len(all) == 0 {
		return nil, nil
	}

	remotes := make(map[string]*Remote, len(all))
	for _, res := range all {
		remote, ok := res.Data.(*Remote)
		if !ok {
			return nil, &ErrInvalidDataType{Expected: (*Remote)(nil), Actual: res.Data}
		}

		remotes[res.ExternalID] = remote
	}

	return h.Impl.FormatForExport(remotes, idNamer, inputResolver)
}

func CreatePropertyRef[State any](
	urn string,
	extractor func(*State) (string, error),
) *resources.PropertyRef {
	return &resources.PropertyRef{
		URN: urn,
		Resolve: func(outputRaw any) (string, error) {
			typedState, ok := outputRaw.(*State)
			if !ok {
				return "", &ErrInvalidDataType{Expected: (*State)(nil), Actual: outputRaw}
			}

			return extractor(typedState)
		},
	}
}

type ErrInvalidDataType struct {
	Expected any
	Actual   any
}

func (e *ErrInvalidDataType) Error() string {
	expectedType := reflect.TypeOf(e.Expected)
	actualType := reflect.TypeOf(e.Actual)
	return fmt.Sprintf("invalid resource data type. Found %v, expected %v", actualType, expectedType)
}
