package writer

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/testutils/example/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type WriterHandler = handler.BaseHandler[model.WriterSpec, model.WriterResource, model.WriterState, model.RemoteWriter]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "example-writer",
	SpecKind:         "writer",
	SpecMetadataName: "writers",
}

// HandlerImpl implements the HandlerImpl interface for writer resources
type HandlerImpl struct {
	*export.MultiSpecExportStrategy[model.WriterSpec, model.RemoteWriter]
	backend *backend.Backend
}

// NewHandler creates a new BaseHandler for writer resources
func NewHandler(backend *backend.Backend) *WriterHandler {
	h := &HandlerImpl{backend: backend}
	h.MultiSpecExportStrategy = &export.MultiSpecExportStrategy[model.WriterSpec, model.RemoteWriter]{Handler: h}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *model.WriterSpec {
	return &model.WriterSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *model.WriterSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.WriterSpec) (map[string]*model.WriterResource, error) {
	resource := &model.WriterResource{
		ID:   spec.ID,
		Name: spec.Name,
	}
	return map[string]*model.WriterResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) ValidateResource(resource *model.WriterResource, graph *resources.Graph) error {
	if resource.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*model.RemoteWriter, error) {
	remoteWriters := h.backend.AllWriters()

	// Filter only managed resources (those with external IDs)
	result := make([]*model.RemoteWriter, 0)
	for _, w := range remoteWriters {
		if w.ExternalID != "" {
			result = append(result, &model.RemoteWriter{RemoteWriter: w})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*model.RemoteWriter, error) {
	remoteWriters := h.backend.AllWriters()

	fmt.Println("Loading importable resources for writers")

	// Return all resources for import
	result := make([]*model.RemoteWriter, 0, len(remoteWriters))
	for _, w := range remoteWriters {
		result = append(result, &model.RemoteWriter{RemoteWriter: w})
	}
	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteWriter, urnResolver handler.URNResolver) (*model.WriterResource, *model.WriterState, error) {
	resource := &model.WriterResource{
		ID:   remote.ExternalID,
		Name: remote.Name,
	}

	state := &model.WriterState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *model.WriterResource) (*model.WriterState, error) {
	remoteWriter, err := h.backend.CreateWriter(data.Name, data.ID)
	if err != nil {
		return nil, fmt.Errorf("creating writer in backend: %w", err)
	}

	return &model.WriterState{
		ID: remoteWriter.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *model.WriterResource, oldData *model.WriterResource, oldState *model.WriterState) (*model.WriterState, error) {
	remoteWriter, err := h.backend.UpdateWriter(oldState.ID, newData.Name)
	if err != nil {
		return nil, fmt.Errorf("updating writer in backend: %w", err)
	}

	return &model.WriterState{
		ID: remoteWriter.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.WriterResource, remoteId string) (*model.WriterState, error) {
	// Set external ID on the remote resource
	if err := h.backend.SetWriterExternalID(remoteId, ""); err != nil {
		return nil, fmt.Errorf("setting external ID on writer: %w", err)
	}

	remoteWriter, err := h.backend.Writer(remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting writer from backend: %w", err)
	}

	return &model.WriterState{
		ID: remoteWriter.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.WriterResource, oldState *model.WriterState) error {
	return h.backend.DeleteWriter(oldState.ID)
}

func (h *HandlerImpl) MapRemoteToSpec(externalID string, remote *model.RemoteWriter) (*export.SpecExportData[model.WriterSpec], error) {
	return &export.SpecExportData[model.WriterSpec]{
		Data: &model.WriterSpec{
			ID:   externalID,
			Name: remote.Name,
		},
		RelativePath: fmt.Sprintf("writers/%s.yaml", externalID),
	}, nil
}

func ParseWriterReference(ref string) (string, error) {
	specRef, err := specs.ParseSpecReference(ref, map[string]string{HandlerMetadata.SpecKind: HandlerMetadata.ResourceType})
	if err != nil {
		return "", err
	}
	return specRef.URN, nil
}

var CreateWriterReference = func(urn string) *resources.PropertyRef {
	return handler.CreatePropertyRef(urn, func(stateOutput *model.WriterState) (string, error) {
		return stateOutput.ID, nil
	})
}
