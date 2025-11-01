package provider

import (
	"context"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type ImportResourceInfo struct {
	WorkspaceId string
	RemoteId    string
}

type BaseHandler[S any, R any] struct {
	ResourceType   string
	resources      map[string]*R
	importMetadata map[string]*ImportResourceInfo
	Impl           HandlerImpl[S, R]
}

func NewHandler[S any, R any](resourceType string, impl HandlerImpl[S, R]) *BaseHandler[S, R] {
	return &BaseHandler[S, R]{
		ResourceType:   resourceType,
		resources:      make(map[string]*R),
		importMetadata: make(map[string]*ImportResourceInfo),
		Impl:           impl,
	}
}

type HandlerImpl[S any, R any] interface {
	NewSpec() *S
	ValidateSpec(spec *S) error
	// NOTE: path should be part of the spec
	ExtractResourcesFromSpec(path string, spec *S) (map[string]*R, error)

	ValidateResource(resource *R, graph *resources.Graph) error

	EncodeResource(resource *R) resources.ResourceData
}

func (h *BaseHandler[S, R]) FetchImportData(ctx context.Context, args importremote.ImportArgs) ([]importremote.ImportData, error) {
	return nil, nil
}

func (h *BaseHandler[S, R]) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	return nil, nil
}

func (h *BaseHandler[S, R]) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	return nil, nil
}

func (h *BaseHandler[S, R]) loadImportMetadata(fileMetadata map[string]interface{}) error {
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

func (h *BaseHandler[S, R]) ParseSpec(_ string, s *specs.Spec) (*specs.ParsedSpec, error) {
	id, ok := s.Spec["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id not found in event stream source spec")
	}
	return &specs.ParsedSpec{ExternalIDs: []string{id}}, nil
}

func (h *BaseHandler[S, R]) LoadSpec(path string, s *specs.Spec) error {
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
			return fmt.Errorf("resource with id %s already exists", id)
		}
		h.resources[id] = r
	}

	return h.loadImportMetadata(s.Metadata)
}

func (h *BaseHandler[S, R]) Validate(graph *resources.Graph) error {
	for _, source := range h.resources {
		if err := h.Impl.ValidateResource(source, graph); err != nil {
			return fmt.Errorf("validating event stream source spec: %w", err)
		}
	}
	return nil
}

func (h *BaseHandler[S, R]) GetResources() ([]*resources.Resource, error) {
	result := make([]*resources.Resource, 0, len(h.resources))
	for resourceId, s := range h.resources {
		data := h.Impl.EncodeResource(s)

		opts := []resources.ResourceOpts{}
		if importMetadata, ok := h.importMetadata[resourceId]; ok {
			opts = []resources.ResourceOpts{
				resources.WithResourceImportMetadata(importMetadata.RemoteId, importMetadata.WorkspaceId),
			}
		}
		r := resources.NewResource(
			resourceId,
			h.ResourceType,
			data,
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
