package model

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type ModelHandler = handler.BaseHandler[struct{}, dgModel.ModelResource, dgModel.ModelState, dgModel.RemoteModel]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "data-graph-model",
	SpecKind:         "data-graph", // Models are inline in data-graph specs
	SpecMetadataName: "data-graph",
}

// HandlerImpl implements the HandlerImpl interface for model resources
// Note: Models don't have their own spec kind - they're inline in data-graph specs
// The provider handles all spec parsing and resource extraction
type HandlerImpl struct {
	client dgClient.DataGraphClient
}

// NewHandler creates a new BaseHandler for model resources

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

func NewHandler(client dgClient.DataGraphClient) *ModelHandler {
	h := &HandlerImpl{client: client}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *struct{} {
	return &struct{}{}
}

func (h *HandlerImpl) ValidateSpec(spec *struct{}) error {
	// Models don't have standalone specs - they're inline in data-graph specs
	// The provider handles all spec validation
	return fmt.Errorf("model handler does not support standalone specs - models are inline in data-graph specs")
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *struct{}) (map[string]*dgModel.ModelResource, error) {
	// Models don't have standalone specs - they're inline in data-graph specs
	// The provider handles all resource extraction
	return nil, fmt.Errorf("model handler does not support standalone spec extraction - models are inline in data-graph specs")
}

func (h *HandlerImpl) ValidateResource(resource *dgModel.ModelResource, graph *resources.Graph) error {
	if resource.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if resource.Type != "entity" && resource.Type != "event" {
		return fmt.Errorf("type must be 'entity' or 'event'")
	}
	if resource.Table == "" {
		return fmt.Errorf("table is required")
	}
	if resource.DataGraphRef == nil {
		return fmt.Errorf("data_graph reference is required")
	}

	// Type-specific validation
	switch resource.Type {
	case "entity":
		if resource.PrimaryID == "" {
			return fmt.Errorf("primary_id is required for entity models")
		}
	case "event":
		if resource.Timestamp == "" {
			return fmt.Errorf("timestamp is required for event models")
		}
	}

	// Validate that the referenced data graph exists
	if _, exists := graph.GetResource(resource.DataGraphRef.URN); !exists {
		return fmt.Errorf("referenced data graph %s does not exist", resource.DataGraphRef.URN)
	}

	return nil
}

// listAllEntityModels fetches all entity models for a data graph with pagination
func (h *HandlerImpl) listAllEntityModels(ctx context.Context, dataGraphID string, hasExternalID *bool) ([]*dgModel.RemoteModel, error) {
	var allModels []*dgModel.RemoteModel
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListModels(ctx, &dgClient.ListModelsRequest{DataGraphID: dataGraphID, Page: page, PageSize: perPage, ModelType: stringPtr("entity"), HasExternalID: hasExternalID})
		if err != nil {
			return nil, fmt.Errorf("listing entity models: %w", err)
		}

		// Add all models from current page
		for i := range resp.Data {
			allModels = append(allModels, &dgModel.RemoteModel{
				Model: &resp.Data[i],
			})
		}

		// Check if there are more pages
		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allModels, nil
}

// listAllEventModels fetches all event models for a data graph with pagination
func (h *HandlerImpl) listAllEventModels(ctx context.Context, dataGraphID string, hasExternalID *bool) ([]*dgModel.RemoteModel, error) {
	var allModels []*dgModel.RemoteModel
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListModels(ctx, &dgClient.ListModelsRequest{DataGraphID: dataGraphID, Page: page, PageSize: perPage, ModelType: stringPtr("event"), HasExternalID: hasExternalID})
		if err != nil {
			return nil, fmt.Errorf("listing event models: %w", err)
		}

		// Add all models from current page
		for i := range resp.Data {
			// Set DataGraphID if not already set by API
			if resp.Data[i].DataGraphID == "" {
				resp.Data[i].DataGraphID = dataGraphID
			}
			allModels = append(allModels, &dgModel.RemoteModel{
				Model: &resp.Data[i],
			})
		}

		// Check if there are more pages
		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allModels, nil
}

// listAllModelsForDataGraph fetches all models (entity and event) for a data graph
func (h *HandlerImpl) listAllModelsForDataGraph(ctx context.Context, dataGraphID string, hasExternalID *bool) ([]*dgModel.RemoteModel, error) {
	var allModels []*dgModel.RemoteModel

	// Fetch entity models
	entityModels, err := h.listAllEntityModels(ctx, dataGraphID, hasExternalID)
	if err != nil {
		return nil, err
	}
	allModels = append(allModels, entityModels...)

	// Fetch event models
	eventModels, err := h.listAllEventModels(ctx, dataGraphID, hasExternalID)
	if err != nil {
		return nil, err
	}
	allModels = append(allModels, eventModels...)

	return allModels, nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*dgModel.RemoteModel, error) {
	hasExternalID := true

	// First, get all data graphs with external IDs
	dataGraphs, err := h.listAllDataGraphs(ctx, &hasExternalID)
	if err != nil {
		return nil, err
	}

	// For each data graph, fetch all models with external IDs
	var allModels []*dgModel.RemoteModel
	for _, dg := range dataGraphs {
		models, err := h.listAllModelsForDataGraph(ctx, dg.ID, &hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("loading models for data graph %s: %w", dg.ID, err)
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*dgModel.RemoteModel, error) {
	hasExternalID := false

	// First, get all data graphs
	dataGraphs, err := h.listAllDataGraphs(ctx, nil)
	if err != nil {
		return nil, err
	}

	// For each data graph, fetch all models without external IDs
	var allModels []*dgModel.RemoteModel
	for _, dg := range dataGraphs {
		models, err := h.listAllModelsForDataGraph(ctx, dg.ID, &hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("loading importable models for data graph %s: %w", dg.ID, err)
		}
		allModels = append(allModels, models...)
	}

	return allModels, nil
}

// listAllDataGraphs fetches all data graphs with pagination
func (h *HandlerImpl) listAllDataGraphs(ctx context.Context, hasExternalID *bool) ([]*dgClient.DataGraph, error) {
	var allDataGraphs []*dgClient.DataGraph
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListDataGraphs(ctx, page, perPage, hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("listing data graphs: %w", err)
		}

		// Add all data graphs from current page
		for i := range resp.Data {
			allDataGraphs = append(allDataGraphs, &resp.Data[i])
		}

		// Check if there are more pages
		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allDataGraphs, nil
}

func (h *HandlerImpl) MapRemoteToState(remote *dgModel.RemoteModel, urnResolver handler.URNResolver) (*dgModel.ModelResource, *dgModel.ModelState, error) {
	// Skip resources without external IDs
	if remote.ExternalID == "" {
		return nil, nil, nil
	}

	// Resolve the data graph's URN from its remote ID
	dataGraphURN, err := urnResolver.GetURNByID(datagraph.HandlerMetadata.ResourceType, remote.DataGraphID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving data graph URN: %w", err)
	}

	// Create PropertyRef to the data graph using the resolved URN
	dataGraphRef := datagraph.CreateDataGraphReference(dataGraphURN)

	resource := &dgModel.ModelResource{
		ID:           remote.ExternalID,
		DisplayName:  remote.Name,
		Type:         remote.Type,
		Table:        remote.TableRef,
		Description:  remote.Description,
		DataGraphRef: dataGraphRef,
		PrimaryID:    remote.PrimaryID,
		Root:         remote.Root,
		Timestamp:    remote.Timestamp,
	}

	state := &dgModel.ModelState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *dgModel.ModelResource) (*dgModel.ModelState, error) {
	dataGraphRemoteID := data.DataGraphRef.Value

	var remote *dgClient.Model
	var err error

	switch data.Type {
	case "entity":
		req := &dgClient.CreateModelRequest{
			DataGraphID: dataGraphRemoteID,
			Type:        "entity",
			Name:        data.DisplayName,
			Description: data.Description,
			TableRef:    data.Table,
			ExternalID:  data.ID,
			PrimaryID:   data.PrimaryID,
			Root:        data.Root,
		}
		remote, err = h.client.CreateModel(ctx, req)
	case "event":
		req := &dgClient.CreateModelRequest{
			DataGraphID: dataGraphRemoteID,
			Type:        "event",
			Name:        data.DisplayName,
			Description: data.Description,
			TableRef:    data.Table,
			ExternalID:  data.ID,
			Timestamp:   data.Timestamp,
		}
		remote, err = h.client.CreateModel(ctx, req)
	default:
		return nil, fmt.Errorf("invalid model type: %s", data.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("creating %s model: %w", data.Type, err)
	}

	return &dgModel.ModelState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *dgModel.ModelResource, oldData *dgModel.ModelResource, oldState *dgModel.ModelState) (*dgModel.ModelState, error) {
	dataGraphRemoteID := oldData.DataGraphRef.Value

	var remote *dgClient.Model
	var err error

	switch newData.Type {
	case "entity":
		req := &dgClient.UpdateModelRequest{
			DataGraphID: dataGraphRemoteID,
			ModelID:     oldState.ID,
			Type:        "entity",
			Name:        newData.DisplayName,
			Description: newData.Description,
			TableRef:    newData.Table,
			PrimaryID:   newData.PrimaryID,
			Root:        newData.Root,
		}
		remote, err = h.client.UpdateModel(ctx, req)
	case "event":
		req := &dgClient.UpdateModelRequest{
			DataGraphID: dataGraphRemoteID,
			ModelID:     oldState.ID,
			Type:        "event",
			Name:        newData.DisplayName,
			Description: newData.Description,
			TableRef:    newData.Table,
			Timestamp:   newData.Timestamp,
		}
		remote, err = h.client.UpdateModel(ctx, req)
	default:
		return nil, fmt.Errorf("invalid model type: %s", newData.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("updating %s model: %w", newData.Type, err)
	}

	return &dgModel.ModelState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *dgModel.ModelResource, remoteID string) (*dgModel.ModelState, error) {
	dataGraphRemoteID := data.DataGraphRef.Value

	// Set external ID on the remote resource and get the updated model
	remote, setErr := h.client.SetModelExternalID(ctx, &dgClient.SetModelExternalIDRequest{
		DataGraphID: dataGraphRemoteID,
		ModelID:     remoteID,
		ExternalID:  data.ID,
	})
	if setErr != nil {
		return nil, fmt.Errorf("setting external ID: %w", setErr)
	}

	return &dgModel.ModelState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *dgModel.ModelResource, oldState *dgModel.ModelState) error {
	dataGraphRemoteID := oldData.DataGraphRef.Value

	err := h.client.DeleteModel(ctx, &dgClient.DeleteModelRequest{
		DataGraphID: dataGraphRemoteID,
		ModelID:     oldState.ID,
	})
	if err != nil {
		return fmt.Errorf("deleting %s model: %w", oldData.Type, err)
	}

	return nil
}

// FormatForExport formats resources for export
// Models are exported inline in data-graph specs, so this is not supported
func (h *HandlerImpl) FormatForExport(
	collection map[string]*dgModel.RemoteModel,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	// Models don't have standalone specs - they're exported inline in data-graph specs
	return nil, nil
}
