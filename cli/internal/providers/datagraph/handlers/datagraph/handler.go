package datagraph

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler/export"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type DataGraphHandler = handler.BaseHandler[model.DataGraphSpec, model.DataGraphResource, model.DataGraphState, model.RemoteDataGraph]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "data-graph",
	SpecKind:         "data-graph",
	SpecMetadataName: "data-graph",
}

// HandlerImpl implements the HandlerImpl interface for data graph resources
type HandlerImpl struct {
	*export.MultiSpecExportStrategy[model.DataGraphSpec, model.RemoteDataGraph]
	client dgClient.DataGraphStore
}

// NewHandler creates a new BaseHandler for data graph resources
func NewHandler(client dgClient.DataGraphStore) *DataGraphHandler {
	h := &HandlerImpl{client: client}
	h.MultiSpecExportStrategy = &export.MultiSpecExportStrategy[model.DataGraphSpec, model.RemoteDataGraph]{Handler: h}
	return handler.NewHandler(h)
}

func (h *HandlerImpl) Metadata() handler.HandlerMetadata {
	return HandlerMetadata
}

func (h *HandlerImpl) NewSpec() *model.DataGraphSpec {
	return &model.DataGraphSpec{}
}

func (h *HandlerImpl) ValidateSpec(spec *model.DataGraphSpec) error {
	if spec.ID == "" {
		return fmt.Errorf("id is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("name is required")
	}
	if spec.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	return nil
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *model.DataGraphSpec) (map[string]*model.DataGraphResource, error) {
	resource := &model.DataGraphResource{
		ID:        spec.ID,
		Name:      spec.Name,
		AccountID: spec.AccountID,
	}
	return map[string]*model.DataGraphResource{
		spec.ID: resource,
	}, nil
}

func (h *HandlerImpl) ValidateResource(resource *model.DataGraphResource, graph *resources.Graph) error {
	if resource.Name == "" {
		return fmt.Errorf("name is required")
	}
	if resource.AccountID == "" {
		return fmt.Errorf("account_id is required")
	}
	return nil
}

// listAllDataGraphs fetches all data graphs with pagination and optional filtering
func (h *HandlerImpl) listAllDataGraphs(ctx context.Context, hasExternalID *bool) ([]*model.RemoteDataGraph, error) {
	var allDataGraphs []*model.RemoteDataGraph
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListDataGraphs(ctx, page, perPage, hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("listing data graphs: %w", err)
		}

		// Add all resources from current page
		for i := range resp.Data {
			allDataGraphs = append(allDataGraphs, &model.RemoteDataGraph{
				DataGraph: &resp.Data[i],
			})
		}

		// Check if there are more pages
		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allDataGraphs, nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*model.RemoteDataGraph, error) {
	// Fetch all data graphs with external IDs using API filtering
	hasExternalID := true
	return h.listAllDataGraphs(ctx, &hasExternalID)
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*model.RemoteDataGraph, error) {
	// Fetch all data graphs without external IDs using API filtering
	hasExternalID := false
	return h.listAllDataGraphs(ctx, &hasExternalID)
}

func (h *HandlerImpl) MapRemoteToState(remote *model.RemoteDataGraph, urnResolver handler.URNResolver) (*model.DataGraphResource, *model.DataGraphState, error) {
	// Skip resources without external IDs
	if remote.ExternalID == "" {
		return nil, nil, nil
	}

	resource := &model.DataGraphResource{
		ID:        remote.ExternalID,
		Name:      remote.Name,
		AccountID: remote.WarehouseAccountID,
	}

	state := &model.DataGraphState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *model.DataGraphResource) (*model.DataGraphState, error) {
	req := &dgClient.CreateDataGraphRequest{
		Name:               data.Name,
		WarehouseAccountID: data.AccountID,
		ExternalID:         data.ID,
	}

	remote, err := h.client.CreateDataGraph(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating data graph: %w", err)
	}

	return &model.DataGraphState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *model.DataGraphResource, oldData *model.DataGraphResource, oldState *model.DataGraphState) (*model.DataGraphState, error) {
	req := &dgClient.UpdateDataGraphRequest{
		Name: newData.Name,
	}

	remote, err := h.client.UpdateDataGraph(ctx, oldState.ID, req)
	if err != nil {
		return nil, fmt.Errorf("updating data graph: %w", err)
	}

	return &model.DataGraphState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.DataGraphResource, remoteID string) (*model.DataGraphState, error) {
	// Set external ID on the remote resource
	if err := h.client.SetExternalID(ctx, remoteID, data.ID); err != nil {
		return nil, fmt.Errorf("setting external ID: %w", err)
	}

	// Fetch the remote resource to get updated data
	remote, err := h.client.GetDataGraph(ctx, remoteID)
	if err != nil {
		return nil, fmt.Errorf("getting data graph: %w", err)
	}

	return &model.DataGraphState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *model.DataGraphResource, oldState *model.DataGraphState) error {
	if err := h.client.DeleteDataGraph(ctx, oldState.ID); err != nil {
		return fmt.Errorf("deleting data graph: %w", err)
	}
	return nil
}

// MapRemoteToSpec converts a remote data graph to a spec for export
func (h *HandlerImpl) MapRemoteToSpec(externalID string, remote *model.RemoteDataGraph) (*export.SpecExportData[model.DataGraphSpec], error) {
	return &export.SpecExportData[model.DataGraphSpec]{
		Data: &model.DataGraphSpec{
			ID:        externalID,
			Name:      remote.Name,
			AccountID: remote.WarehouseAccountID,
		},
		RelativePath: fmt.Sprintf("data-graphs/%s.yaml", externalID),
	}, nil
}
