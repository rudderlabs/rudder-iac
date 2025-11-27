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
	resourceType       string
	specKind           string
	importMetadataName string
	resources          map[string]*Res
	importMetadata     map[string]*importResourceInfo
	Impl               HandlerImpl[Spec, Res, State, Remote]
}

type importResourceInfo struct {
	WorkspaceId string
	RemoteId    string
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
		importMetadata:     make(map[string]*importResourceInfo),
		Impl:               impl,
	}
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
			h.importMetadata[resourceMetadata.LocalID] = &importResourceInfo{
				WorkspaceId: workspaceId,
				RemoteId:    resourceMetadata.RemoteID,
			}
		}
	}
	return nil
}

func (h *BaseHandler[Spec, Res, State, Remote]) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	// First, try to extract a single "id" field
	if id, ok := s.Spec["id"].(string); ok {
		return &specs.ParsedSpec{ExternalIDs: []string{id}}, nil
	}

	// If the spec has a single field that is an array, extract IDs from array elements
	if len(s.Spec) == 1 {
		for _, value := range s.Spec {
			if arr, ok := value.([]interface{}); ok {
				externalIDs := make([]string, 0, len(arr))
				for i, item := range arr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if id, ok := itemMap["id"].(string); ok {
							externalIDs = append(externalIDs, id)
						} else {
							return nil, fmt.Errorf("array item at index %d does not have an 'id' field", i)
						}
					} else {
						return nil, fmt.Errorf("array item at index %d is not a map", i)
					}
				}
				return &specs.ParsedSpec{ExternalIDs: externalIDs}, nil
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

func (h *BaseHandler[Spec, Res, State, Remote]) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	return h.Impl.FormatForExport(ctx, collection, idNamer, inputResolver)
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
