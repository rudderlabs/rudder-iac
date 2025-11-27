package writer

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/example/backend"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type WriterHandler = handler.BaseHandler[WriterSpec, WriterResource, WriterState, RemoteWriter]

const (
	ResourceType = "example_writer"
	SpecKind     = "writer"
)

func ParseWriterReference(ref string) (string, error) {
	specRef, err := specs.ParseSpecReference(ref, map[string]string{SpecKind: ResourceType})
	if err != nil {
		return "", err
	}
	return specRef.URN, nil
}

var CreateWriterReference = func(urn string) *resources.PropertyRef {
	return handler.CreatePropertyRef(urn, func(stateOutput *WriterState) (string, error) {
		return stateOutput.ID, nil
	})
}

// HandlerImpl implements the HandlerImpl interface for writer resources
type HandlerImpl struct {
	backend *backend.Backend
}

func NewHandlerImpl(backend *backend.Backend) *HandlerImpl {
	return &HandlerImpl{
		backend: backend,
	}
}

func (h *HandlerImpl) NewSpec() *WriterSpec {
	return &WriterSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *WriterSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *WriterSpec) (map[string]*WriterResource, error) {
	resource := &WriterResource{
		ID:   spec.ID,
		Name: spec.Name,
	}
	return map[string]*WriterResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) ValidateResource(resource *WriterResource, graph *resources.Graph) error {
	if resource.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*RemoteWriter, error) {
	remoteWriters := h.backend.GetAllWriters()

	// Filter only managed resources (those with external IDs)
	result := make([]*RemoteWriter, 0)
	for _, w := range remoteWriters {
		if w.ExternalID != "" {
			result = append(result, &RemoteWriter{RemoteWriter: w})
		}
	}
	return result, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*RemoteWriter, error) {
	remoteWriters := h.backend.GetAllWriters()

	// Return all resources for import
	result := make([]*RemoteWriter, 0, len(remoteWriters))
	for _, w := range remoteWriters {
		result = append(result, &RemoteWriter{RemoteWriter: w})
	}
	return result, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *RemoteWriter, urnResolver handler.URNResolver) (*WriterResource, *WriterState, error) {
	resource := &WriterResource{
		ID:   remote.ExternalID,
		Name: remote.Name,
	}

	state := &WriterState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *WriterResource) (*WriterState, error) {
	remoteWriter, err := h.backend.CreateWriter(data.Name, data.ID)
	if err != nil {
		return nil, fmt.Errorf("creating writer in backend: %w", err)
	}

	return &WriterState{
		ID: remoteWriter.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *WriterResource, oldData *WriterResource, oldState *WriterState) (*WriterState, error) {
	remoteWriter, err := h.backend.UpdateWriter(oldState.ID, newData.Name)
	if err != nil {
		return nil, fmt.Errorf("updating writer in backend: %w", err)
	}

	return &WriterState{
		ID: remoteWriter.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *WriterResource, remoteId string) (*WriterState, error) {
	// Set external ID on the remote resource
	if err := h.backend.SetWriterExternalID(remoteId, ""); err != nil {
		return nil, fmt.Errorf("setting external ID on writer: %w", err)
	}

	remoteWriter, err := h.backend.GetWriter(remoteId)
	if err != nil {
		return nil, fmt.Errorf("getting writer from backend: %w", err)
	}

	return &WriterState{
		ID: remoteWriter.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *WriterResource, oldState *WriterState) error {
	return h.backend.DeleteWriter(oldState.ID)
}

func (h *HandlerImpl) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	// Example provider doesn't support export
	return nil, nil
}

// NewHandler creates a new BaseHandler for writer resources
func NewHandler(backend *backend.Backend) *WriterHandler {
	return handler.NewHandler(
		SpecKind,
		ResourceType,
		"writers", // import metadata name
		NewHandlerImpl(backend),
	)
}
