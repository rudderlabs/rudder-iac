package transformations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	transformationsClient "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type TransformationHandler = handler.BaseHandler[TransformationSpec, TransformationResource, TransformationState, RemoteTransformation]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "transformation",
	SpecKind:         "transformation",
	SpecMetadataName: "transformation",
}

type HandlerImpl struct {
	client transformationsClient.TransformationStore
}

// NewHandler creates a new BaseHandler for transformation resources
func NewHandler(client transformationsClient.TransformationStore) *TransformationHandler {
	h := &HandlerImpl{client: client}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *TransformationSpec {
	return &TransformationSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *TransformationSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	if spec.Language == "" {
		return fmt.Errorf("language is required")
	}

	// Validate mutually exclusive code/file
	if spec.Code != "" && spec.File != "" {
		return fmt.Errorf("code and file are mutually exclusive")
	}
	if spec.Code == "" && spec.File == "" {
		return fmt.Errorf("either code or file must be specified")
	}

	// Validate language
	if spec.Language != "javascript" && spec.Language != "python" {
		return fmt.Errorf("language must be 'javascript' or 'python', got: %s", spec.Language)
	}

	return nil
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *TransformationSpec) (map[string]*TransformationResource, error) {
	codeStr := spec.Code
	if spec.File != "" {
		filePath := spec.File
		if !filepath.IsAbs(filePath) {
			// Resolve relative to spec file
			specDir := filepath.Dir(path)
			filePath = filepath.Clean(filepath.Join(specDir, spec.File))
		}

		codeBytes, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("reading code file %s (resolved to %s): %w", spec.File, filePath, err)
		}
		codeStr = string(codeBytes)
	}

	resource := &TransformationResource{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Language:    spec.Language,
		Code:        codeStr,
		Tests:       spec.Tests,
	}

	return map[string]*TransformationResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) ValidateResource(resource *TransformationResource, graph *resources.Graph) error {
	if resource.Code == "" {
		return fmt.Errorf("code is required")
	}
	return nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteTransformation, error) {
	transformations, err := h.client.ListTransformations(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing transformations: %w", err)
	}

	// Filter only managed resources (those with external IDs)
	result := make([]*RemoteTransformation, 0)
	for _, t := range transformations {
		if t.ExternalID != "" {
			result = append(result, &RemoteTransformation{Transformation: t})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteTransformation, error) {
	transformations, err := h.client.ListTransformations(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing transformations: %w", err)
	}

	// Return all resources for import
	result := make([]*RemoteTransformation, 0, len(transformations))
	for _, t := range transformations {
		result = append(result, &RemoteTransformation{Transformation: t})
	}
	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *RemoteTransformation, urnResolver handler.URNResolver) (*TransformationResource, *TransformationState, error) {
	resource := &TransformationResource{
		ID:          remote.ExternalID,
		Name:        remote.Name,
		Description: remote.Description,
		Language:    remote.Language,
		Code:        remote.Code,
	}

	state := &TransformationState{
		ID:        remote.ID,
		VersionID: remote.VersionID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *TransformationResource) (*TransformationState, error) {
	req := transformationsClient.CreateTransformationRequest{
		Name:        data.Name,
		Description: data.Description,
		Code:        data.Code,
		Language:    data.Language,
		ExternalID:  data.ID,
	}

	// Always use publish=false, batch publish happens later
	created, err := h.client.CreateTransformation(ctx, req, false)
	if err != nil {
		return nil, fmt.Errorf("creating transformation: %w", err)
	}

	return &TransformationState{
		ID:        created.ID,
		VersionID: created.VersionID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *TransformationResource, oldData *TransformationResource, oldState *TransformationState) (*TransformationState, error) {
	req := transformationsClient.CreateTransformationRequest{
		Name:        newData.Name,
		Description: newData.Description,
		Code:        newData.Code,
		Language:    newData.Language,
		ExternalID:  newData.ID,
	}

	// Always use publish=false
	updated, err := h.client.UpdateTransformation(ctx, oldState.ID, req, false)
	if err != nil {
		return nil, fmt.Errorf("updating transformation: %w", err)
	}

	return &TransformationState{
		ID:        updated.ID,
		VersionID: updated.VersionID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *TransformationResource, remoteId string) (*TransformationState, error) {
	// Get the existing transformation
	existing, err := h.client.GetTransformation(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting transformation: %w", err)
	}

	// Update with external ID
	req := transformationsClient.CreateTransformationRequest{
		Name:        existing.Name,
		Description: existing.Description,
		Code:        existing.Code,
		Language:    existing.Language,
		ExternalID:  data.ID,
	}

	updated, err := h.client.UpdateTransformation(ctx, remoteId, req, false)
	if err != nil {
		return nil, fmt.Errorf("setting external ID on transformation: %w", err)
	}

	return &TransformationState{
		ID:        updated.ID,
		VersionID: updated.VersionID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *TransformationResource, oldState *TransformationState) error {
	return h.client.DeleteTransformation(ctx, oldState.ID)
}

func (h *HandlerImpl) FormatForExport(
	collection map[string]*RemoteTransformation,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	// For now, we only support single transformation per spec
	if len(collection) == 0 {
		return nil, nil
	}

	var entities []writer.FormattableEntity
	for externalID, t := range collection {
		spec := &TransformationSpec{
			ID:          externalID,
			Name:        t.Name,
			Description: t.Description,
			Language:    t.Language,
			Code:        t.Code,
		}

		entities = append(entities, writer.FormattableEntity{
			Content:      spec,
			RelativePath: fmt.Sprintf("transformations/%s.yaml", externalID),
		})
	}

	return entities, nil
}
