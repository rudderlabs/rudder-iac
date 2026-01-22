package library

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
	"github.com/samber/lo"
)

type LibraryHandler = handler.BaseHandler[
	model.LibrarySpec,
	model.LibraryResource,
	model.LibraryState,
	model.RemoteLibrary,
]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "transformation-library",
	SpecKind:         "transformation-library",
	SpecMetadataName: "transformation-libraries",
}

// HandlerImpl implements the HandlerImpl interface for library resources
type HandlerImpl struct {
	*export.MultiSpecExportStrategy[model.LibrarySpec, model.RemoteLibrary]
	store transformations.TransformationStore
}

// NewHandler creates a new BaseHandler for library resources
func NewHandler(store transformations.TransformationStore) *LibraryHandler {
	h := &HandlerImpl{store: store}
	h.MultiSpecExportStrategy = &export.MultiSpecExportStrategy[model.LibrarySpec, model.RemoteLibrary]{Handler: h}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *model.LibrarySpec {
	return &model.LibrarySpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *model.LibrarySpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	if spec.ImportName == "" {
		return fmt.Errorf("import_name is required")
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

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.LibrarySpec) (map[string]*model.LibraryResource, error) {
	resource := &model.LibraryResource{
		ID:          spec.ID,
		Name:        spec.Name,
		Description: spec.Description,
		Language:    spec.Language,
		ImportName:  spec.ImportName,
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

	return map[string]*model.LibraryResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) ValidateResource(resource *model.LibraryResource, graph *resources.Graph) error {
	expectedImportName := lo.CamelCase(resource.Name)
	if resource.ImportName != expectedImportName {
		return fmt.Errorf("import_name must be camelCase of name: expected '%s', got '%s'", expectedImportName, resource.ImportName)
	}

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

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*model.RemoteLibrary, error) {
	libraries, err := h.store.ListLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing libraries: %w", err)
	}

	// Filter only managed resources (those with external IDs)
	result := make([]*model.RemoteLibrary, 0)
	for _, l := range libraries {
		if l.ExternalID != "" {
			result = append(result, &model.RemoteLibrary{TransformationLibrary: l})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*model.RemoteLibrary, error) {
	// TODO: Implement when we add List operation to the store
	return []*model.RemoteLibrary{}, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteLibrary, urnResolver handler.URNResolver) (*model.LibraryResource, *model.LibraryState, error) {
	resource := &model.LibraryResource{
		ID:          remote.ExternalID,
		Name:        remote.Name,
		Description: remote.Description,
		Language:    remote.Language,
		Code:        remote.Code,
		ImportName:  remote.ImportName,
	}

	state := &model.LibraryState{
		ID:        remote.ID,
		VersionID: remote.VersionID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *model.LibraryResource) (*model.LibraryState, error) {
	req := &transformations.CreateLibraryRequest{
		Name:        data.Name,
		Description: data.Description,
		Code:        data.Code,
		Language:    data.Language,
		ExternalID:  data.ID,
	}

	// Always use publish=false, batch publish happens later
	created, err := h.store.CreateLibrary(ctx, req, false)
	if err != nil {
		return nil, fmt.Errorf("creating library: %w", err)
	}

	return &model.LibraryState{
		ID:        created.ID,
		VersionID: created.VersionID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *model.LibraryResource, oldData *model.LibraryResource, oldState *model.LibraryState) (*model.LibraryState, error) {
	req := &transformations.UpdateLibraryRequest{
		Name:        newData.Name,
		Description: newData.Description,
		Code:        newData.Code,
		Language:    newData.Language,
	}

	// Always use publish=false, batch publish happens later
	updated, err := h.store.UpdateLibrary(ctx, oldState.ID, req, false)
	if err != nil {
		return nil, fmt.Errorf("updating library: %w", err)
	}

	return &model.LibraryState{
		ID:        updated.ID,
		VersionID: updated.VersionID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.LibraryResource, remoteId string) (*model.LibraryState, error) {
	// TODO: Implement when we add Get operation to the store
	return nil, fmt.Errorf("import not implemented yet")
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.LibraryResource, oldState *model.LibraryState) error {
	if err := h.store.DeleteLibrary(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting library: %w", err)
	}
	return nil
}

func (h *HandlerImpl) MapRemoteToSpec(externalID string, remote *model.RemoteLibrary) (*export.SpecExportData[model.LibrarySpec], error) {
	// TODO: Implement export functionality
	return nil, fmt.Errorf("export not implemented yet")
}
