package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

var errNotImplemented = fmt.Errorf("not implemented")

type ImportResourceInfo struct {
	WorkspaceId string
	RemoteId    string
}

type BaseHandler[Spec any, Res any] struct {
	ResourceType   string
	resources      map[string]*Res
	importMetadata map[string]*ImportResourceInfo
	Impl           HandlerImpl[Spec, Res]
}

func NewHandler[Spec any, Res any](resourceType string, impl HandlerImpl[Spec, Res]) *BaseHandler[Spec, Res] {
	return &BaseHandler[Spec, Res]{
		ResourceType:   resourceType,
		resources:      make(map[string]*Res),
		importMetadata: make(map[string]*ImportResourceInfo),
		Impl:           impl,
	}
}

type HandlerImpl[Spec any, Res any] interface {
	NewSpec() *Spec
	ValidateSpec(spec *Spec) error
	// NOTE: path should be part of the spec
	ExtractResourcesFromSpec(path string, spec *Spec) (map[string]*Res, error)

	ValidateResource(resource *Res, graph *resources.Graph) error

	Create(ctx context.Context, data *Res) (*resources.ResourceData, error)
	Update(ctx context.Context, data *Res, state resources.ResourceData) (*resources.ResourceData, error)
	Import(ctx context.Context, data *Res, remoteId string) (*resources.ResourceData, error)
	Delete(ctx context.Context, id string, state resources.ResourceData) error
}

func (h *BaseHandler[Spec, Res]) FetchImportData(ctx context.Context, args importremote.ImportArgs) ([]importremote.ImportData, error) {
	return nil, nil
}

func (h *BaseHandler[Spec, Res]) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	return nil, nil
}

func (h *BaseHandler[Spec, Res]) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	return nil, nil
}

func (h *BaseHandler[Spec, Res]) loadImportMetadata(fileMetadata map[string]interface{}) error {
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

func (h *BaseHandler[Spec, Res]) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	id, ok := s.Spec["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id not found in event stream source spec")
	}
	return &specs.ParsedSpec{ExternalIDs: []string{id}}, nil
}

func (h *BaseHandler[Spec, Res]) LoadSpec(path string, s *specs.Spec) error {
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
			return fmt.Errorf("a resource of type '%s' with id '%s' already exists", h.ResourceType, id)
		}
		h.resources[id] = r
	}

	return h.loadImportMetadata(s.Metadata)
}

func (h *BaseHandler[Spec, Res]) Validate(graph *resources.Graph) error {
	for _, source := range h.resources {
		if err := h.Impl.ValidateResource(source, graph); err != nil {
			return fmt.Errorf("validating event stream source spec: %w", err)
		}
	}
	return nil
}

func (h *BaseHandler[Spec, Res]) GetResources() ([]*resources.Resource, error) {
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
			h.ResourceType,
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

func (h *BaseHandler[Spec, Res]) Create(ctx context.Context, rawData any) (*resources.ResourceData, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid resource data type. Found %v, expected %v", reflect.TypeOf(rawData), reflect.TypeOf((*Res)(nil)))
	}
	return h.Impl.Create(ctx, raw)
}

func (h *BaseHandler[Spec, Res]) Update(ctx context.Context, rawData any, state resources.ResourceData) (*resources.ResourceData, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid resource data type. Found %v, expected %v", reflect.TypeOf(rawData), reflect.TypeOf((*Res)(nil)))
	}
	return h.Impl.Update(ctx, raw, state)
}

func (h *BaseHandler[Spec, Res]) Delete(ctx context.Context, ID string, state resources.ResourceData) error {
	return h.Impl.Delete(ctx, ID, state)
}

func (h *BaseHandler[Spec, Res]) Import(ctx context.Context, rawData any, remoteId string) (*resources.ResourceData, error) {
	raw, ok := rawData.(*Res)
	if !ok {
		return nil, fmt.Errorf("invalid resource data type. Found %v, expected %v", reflect.TypeOf(rawData), reflect.TypeOf((*Res)(nil)))
	}
	return h.Impl.Import(ctx, raw, remoteId)
}
