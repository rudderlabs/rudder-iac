package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/parser"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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
	return handler.NewHandler[model.LibrarySpec, model.LibraryResource, model.LibraryState, model.RemoteLibrary](h)
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
	if spec.Language != "javascript" && spec.Language != "python" && spec.Language != "pythonfaas" {
		return fmt.Errorf("language must be 'javascript', 'python', or 'pythonfaas', got: %s", spec.Language)
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
	libraries, err := h.store.ListLibraries(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing libraries: %w", err)
	}

	// Filter resources WITHOUT external IDs (unmanaged resources)
	result := make([]*model.RemoteLibrary, 0)
	for _, l := range libraries {
		if l.ExternalID == "" {
			result = append(result, &model.RemoteLibrary{TransformationLibrary: l})
		}
	}
	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteLibrary, urnResolver handler.URNResolver) (*model.LibraryResource, *model.LibraryState, error) {
	resource := &model.LibraryResource{
		ID:          remote.ExternalID,
		Name:        remote.Name,
		Description: remote.Description,
		Language:    remote.Language,
		Code:        remote.Code,
		ImportName:  remote.HandleName,
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
	req := &transformations.CreateLibraryRequest{
		Name:        newData.Name,
		Description: newData.Description,
		Code:        newData.Code,
		Language:    newData.Language,
		ExternalID:  newData.ID,
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
	// Get the existing remote library
	remote, err := h.store.GetLibrary(ctx, remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting library %s: %w", remoteId, err)
	}

	// Set the externalID to link the remote resource to local management
	if err := h.store.SetLibraryExternalID(ctx, remote.ID, data.ID); err != nil {
		return nil, fmt.Errorf("setting library external ID: %w", err)
	}

	return &model.LibraryState{
		ID:        remote.ID,
		VersionID: remote.VersionID,
	}, nil
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

// FormatForExport implements the export functionality for libraries during import.
// It generates two FormattableEntity objects per library: a YAML spec and a code file.
func (h *HandlerImpl) FormatForExport(
	remotes map[string]*model.RemoteLibrary,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	if len(remotes) == 0 {
		return nil, nil
	}

	formattables := make([]writer.FormattableEntity, 0)

	for externalID, remote := range remotes {
		// Validate language
		if remote.Language != "javascript" && remote.Language != "python" && remote.Language != "pythonfaas" {
			return nil, fmt.Errorf("unsupported language '%s' for library %s: only 'javascript', 'python', and 'pythonfaas' are supported", remote.Language, remote.ID)
		}

		// Determine file extension and folder based on language
		var ext string
		var langFolder string
		switch remote.Language {
		case "javascript":
			ext = ".js"
			langFolder = "javascript"
		case "python", "pythonfaas":
			ext = ".py"
			langFolder = "python"
		}

		// Code file path: transformations/<language-folder>/<external-id>.<ext>
		codeFilePath := filepath.Join("transformations", langFolder, externalID+ext)

		// Build import metadata
		workspaceMetadata := specs.WorkspaceImportMetadata{
			WorkspaceID: remote.WorkspaceID,
			Resources: []specs.ImportIds{
				{
					LocalID:  externalID,
					RemoteID: remote.ID,
				},
			},
		}

		// Create spec with file reference and import_name
		spec, err := toImportSpec(
			HandlerMetadata.SpecKind,
			HandlerMetadata.SpecMetadataName,
			workspaceMetadata,
			map[string]any{
				"id":          externalID,
				"name":        remote.Name,
				"description": remote.Description,
				"language":    remote.Language,
				"file":        codeFilePath,
				"import_name": remote.HandleName,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("creating spec for library %s: %w", remote.ID, err)
		}

		// Generate unique filename for YAML spec
		fileName, err := idNamer.Name(namer.ScopeName{
			Name:  externalID,
			Scope: "file-name-library",
		})
		if err != nil {
			return nil, fmt.Errorf("generating file name for library %s: %w", remote.ID, err)
		}

		// Add YAML spec entity
		formattables = append(formattables, writer.FormattableEntity{
			Content:      spec,
			RelativePath: filepath.Join("transformations", fileName+loader.ExtensionYAML),
		})

		// Add code file entity
		formattables = append(formattables, writer.FormattableEntity{
			Content:      remote.Code,
			RelativePath: codeFilePath,
		})
	}

	return formattables, nil
}

// toImportSpec creates a Spec with import metadata for a library resource.
func toImportSpec(
	kind string,
	metadataName string,
	workspaceMetadata specs.WorkspaceImportMetadata,
	specData map[string]any,
) (*specs.Spec, error) {
	metadata := specs.Metadata{
		Name: metadataName,
		Import: &specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{workspaceMetadata},
		},
	}

	metadataMap, err := metadata.ToMap()
	if err != nil {
		return nil, fmt.Errorf("converting metadata to map: %w", err)
	}

	return &specs.Spec{
		Version:  specs.SpecVersion,
		Kind:     kind,
		Metadata: metadataMap,
		Spec:     specData,
	}, nil
}
