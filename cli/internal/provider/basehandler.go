package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

var errNotImplemented = fmt.Errorf("not implemented")

type ImportResourceInfo struct {
	WorkspaceId string
	RemoteId    string
}

// URNResolver provides URN resolution from remote resource IDs
type URNResolver interface {
	GetURNByID(resourceType string, id string) (string, error)
}

type RemoteResourceMetadata struct {
	ID          string
	ExternalID  string
	WorkspaceID string
	Name        string
}

type RemoteResource interface {
	GetResourceMetadata() *RemoteResourceMetadata
}

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

type HandlerImpl[Spec any, Res any, State any, Remote RemoteResource] interface {
	NewSpec() *Spec
	ValidateSpec(spec *Spec) error
	// NOTE: path should be part of the spec
	ExtractResourcesFromSpec(path string, spec *Spec) (map[string]*Res, error)

	ValidateResource(resource *Res, graph *resources.Graph) error

	LoadRemoteResources(ctx context.Context) ([]*Remote, error)
	LoadImportableResources(ctx context.Context) ([]*Remote, error)
	MapRemoteToState(remote *Remote, urnResolver URNResolver) (*Res, *State, error)

	Create(ctx context.Context, data *Res) (*State, error)
	Update(ctx context.Context, newData *Res, oldData *Res, oldState *State) (*State, error)
	Import(ctx context.Context, data *Res, remoteId string) (*State, error)
	Delete(ctx context.Context, id string, oldData *Res, oldState *State) error
}

func (h *BaseHandler[Spec, Res, State, Remote]) GetResourceType() string {
	return h.resourceType
}

func (h *BaseHandler[Spec, Res, State, Remote]) GetSpecKind() string {
	return h.specKind
}

func (h *BaseHandler[Spec, Res, State, Remote]) FetchImportData(ctx context.Context, args importremote.ImportArgs) ([]importremote.ImportData, error) {
	return nil, nil
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

func (h *BaseHandler[Spec, Res, State, Remote]) loadImportMetadata(fileMetadata map[string]interface{}) error {
	metadata := importremote.Metadata{}
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

// NOTE: Check this. ResourceKind and MetadataName were hardcoded in the event stream source handler
// This is supposed to return a resource's refererence but it needs to load metadata from the spec file (instead of hardcoding it)
// func getFileMetadata(externalID string) string {
// 	return fmt.Sprintf("#/%s/%s/%s",
// 		ResourceKind,
// 		MetadataName,
// 		externalID,
// 	)
// }

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
