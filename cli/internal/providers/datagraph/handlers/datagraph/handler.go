package datagraph

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type DataGraphHandler = handler.BaseHandler[struct{}, model.DataGraphResource, model.DataGraphState, model.RemoteDataGraph]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "data-graph",
	SpecKind:         "data-graph",
	SpecMetadataName: "data-graph",
}

// HandlerImpl implements the HandlerImpl interface for data graph resources
// Note: The provider handles all spec parsing and resource extraction for data-graph specs
type HandlerImpl struct {
	client dgClient.DataGraphStore
}

// NewHandler creates a new BaseHandler for data graph resources
func NewHandler(client dgClient.DataGraphStore) *DataGraphHandler {
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
	// Spec validation is handled by the provider
	return fmt.Errorf("data graph handler does not handle spec validation - handled by provider")
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *struct{}) (map[string]*model.DataGraphResource, error) {
	// Resource extraction is handled by the provider
	return nil, fmt.Errorf("data graph handler does not handle spec extraction - handled by provider")
}

func (h *HandlerImpl) ValidateResource(resource *model.DataGraphResource, graph *resources.Graph) error {
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
		AccountID: remote.AccountID,
	}

	state := &model.DataGraphState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *model.DataGraphResource) (*model.DataGraphState, error) {
	req := &dgClient.CreateDataGraphRequest{
		AccountID:  data.AccountID,
		ExternalID: data.ID,
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
	// Data graphs do not support updates after creation
	// The only mutable field is externalId which is managed through Import operation
	return nil, fmt.Errorf("data graphs do not support updates after creation; delete and recreate the resource to apply changes")
}

func (h *HandlerImpl) Import(ctx context.Context, data *model.DataGraphResource, remoteID string) (*model.DataGraphState, error) {
	// Set external ID on the remote resource and get the updated data graph
	remote, err := h.client.SetExternalID(ctx, remoteID, data.ID)
	if err != nil {
		return nil, fmt.Errorf("setting external ID: %w", err)
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

// FormatForExport formats resources for export
// Export is not currently implemented for data graphs
func (h *HandlerImpl) FormatForExport(
	collection map[string]*model.RemoteDataGraph,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	// Export is handled directly by the provider for data graphs
	return nil, nil
}

// CreateDataGraphReference creates a PropertyRef that points to a data graph's remote ID
// This is used by other resources (like models) that need to reference a data graph
// The urn parameter should be in the format "data-graph:external-id"
func CreateDataGraphReference(urn string) *resources.PropertyRef {
	return handler.CreatePropertyRef(
		urn,
		func(state *model.DataGraphState) (string, error) {
			return state.ID, nil
		},
	)
}
