package transformation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/parser"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type TransformationHandler = handler.BaseHandler[
	model.TransformationSpec,
	model.TransformationResource,
	model.TransformationState,
	model.RemoteTransformation,
]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "transformation",
	SpecKind:         "transformation",
	SpecMetadataName: "transformations",
}

// HandlerImpl implements the HandlerImpl interface for transformation resources
type HandlerImpl struct {
	*export.MultiSpecExportStrategy[model.TransformationSpec, model.RemoteTransformation]
	store transformations.TransformationStore
}

// NewHandler creates a new BaseHandler for transformation resources
func NewHandler(store transformations.TransformationStore) *TransformationHandler {
	h := &HandlerImpl{store: store}
	h.MultiSpecExportStrategy = &export.MultiSpecExportStrategy[model.TransformationSpec, model.RemoteTransformation]{Handler: h}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *model.TransformationSpec {
	return &model.TransformationSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *model.TransformationSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	if spec.Code != "" && spec.File != "" {
		return fmt.Errorf("code and file are mutually exclusive")
	}
	if spec.Code == "" && spec.File == "" {
		return fmt.Errorf("either code or file must be specified")
	}
	if spec.Language != "javascript" && spec.Language != "python" {
		return fmt.Errorf("language must be 'javascript' or 'python', got: %s", spec.Language)
	}
	return nil
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.TransformationSpec) (map[string]*model.TransformationResource, error) {
	resource := &model.TransformationResource{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Language:    spec.Language,
		Tests:       spec.Tests,
	}

	// Resolve code from file if specified
	if spec.File != "" {
		specDir := filepath.Dir(path)
		codePath := spec.File
		if !filepath.IsAbs(codePath) {
			codePath = filepath.Join(specDir, spec.File)
		}

		codeBytes, err := os.ReadFile(codePath)
		if err != nil {
			return nil, fmt.Errorf("reading code file %s: %w", codePath, err)
		}
		resource.Code = string(codeBytes)
	} else {
		resource.Code = spec.Code
	}

	return map[string]*model.TransformationResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) ValidateResource(resource *model.TransformationResource, graph *resources.Graph) error {
	if resource.Code == "" {
		return fmt.Errorf("code is required")
	}

	// Validate code syntax
	codeParser, err := parser.NewParser(resource.Language)
	if err != nil {
		return fmt.Errorf("creating parser for language %s: %w", resource.Language, err)
	}

	if err := codeParser.ValidateSyntax(resource.Code); err != nil {
		return fmt.Errorf("validating code syntax: %w", err)
	}

	return nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*model.RemoteTransformation, error) {
	transformations, err := h.store.ListTransformations(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing transformations: %w", err)
	}

	// Filter only managed resources (those with external IDs)
	result := make([]*model.RemoteTransformation, 0)
	for _, t := range transformations {
		if t.ExternalID != "" {
			result = append(result, &model.RemoteTransformation{Transformation: t})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*model.RemoteTransformation, error) {
	// TODO: Implement when we add List operation to the store
	return []*model.RemoteTransformation{}, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteTransformation, urnResolver handler.URNResolver) (*model.TransformationResource, *model.TransformationState, error) {
	resource := &model.TransformationResource{
		ID:          remote.ExternalID,
		Name:        remote.Name,
		Description: remote.Description,
		Language:    remote.Language,
		Code:        remote.Code,
	}

	state := &model.TransformationState{
		ID:        remote.ID,
		VersionID: remote.VersionID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *model.TransformationResource) (*model.TransformationState, error) {
	req := &transformations.CreateTransformationRequest{
		Name:        data.Name,
		Description: data.Description,
		Code:        data.Code,
		Language:    data.Language,
		ExternalID:  data.ID,
	}

	// Always use publish=false, batch publish happens later
	created, err := h.store.CreateTransformation(ctx, req, false)
	if err != nil {
		return nil, fmt.Errorf("creating transformation: %w", err)
	}

	return &model.TransformationState{
		ID:        created.ID,
		VersionID: created.VersionID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *model.TransformationResource, oldData *model.TransformationResource, oldState *model.TransformationState) (*model.TransformationState, error) {
	req := &transformations.CreateTransformationRequest{
		Name:        newData.Name,
		Description: newData.Description,
		Code:        newData.Code,
		Language:    newData.Language,
		ExternalID:  newData.ID,
	}

	// Always use publish=false, batch publish happens later
	updated, err := h.store.UpdateTransformation(ctx, oldState.ID, req, false)
	if err != nil {
		return nil, fmt.Errorf("updating transformation: %w", err)
	}

	return &model.TransformationState{
		ID:        updated.ID,
		VersionID: updated.VersionID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.TransformationResource, remoteId string) (*model.TransformationState, error) {
	// TODO: Implement when we add Get operation to the store
	return nil, fmt.Errorf("import not implemented yet")
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.TransformationResource, oldState *model.TransformationState) error {
	if err := h.store.DeleteTransformation(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting transformation: %w", err)
	}
	return nil
}

func (h *HandlerImpl) MapRemoteToSpec(externalID string, remote *model.RemoteTransformation) (*export.SpecExportData[model.TransformationSpec], error) {
	// TODO: Implement export functionality
	return nil, fmt.Errorf("export not implemented yet")
}
