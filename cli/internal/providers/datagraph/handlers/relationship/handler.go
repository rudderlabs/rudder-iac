package relationship

import (
	"context"
	"fmt"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

type RelationshipHandler = handler.BaseHandler[struct{}, dgModel.RelationshipResource, dgModel.RelationshipState, dgModel.RemoteRelationship]

var HandlerMetadata = handler.HandlerMetadata{
	ResourceType:     "data-graph-relationship",
	SpecKind:         "data-graph", // Relationships are inline in data-graph specs
	SpecMetadataName: "data-graph",
}

// HandlerImpl implements the HandlerImpl interface for relationship resources
// Note: Relationships don't have their own spec kind - they're inline in model specs
// The provider handles all spec parsing and resource extraction
type HandlerImpl struct {
	client dgClient.DataGraphClient
}

// NewHandler creates a new BaseHandler for relationship resources
func NewHandler(client dgClient.DataGraphClient) *RelationshipHandler {
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
	// Relationships don't have standalone specs - they're inline in model specs
	// The provider handles all spec validation
	return fmt.Errorf("relationship handler does not support standalone specs - relationships are inline in data-graph specs")
}

func (h *HandlerImpl) ExtractResourcesFromSpec(path string, spec *struct{}) (map[string]*dgModel.RelationshipResource, error) {
	// Relationships don't have standalone specs - they're inline in model specs
	// The provider handles all resource extraction
	return nil, fmt.Errorf("relationship handler does not support standalone spec extraction - relationships are inline in data-graph specs")
}

func (h *HandlerImpl) ValidateResource(resource *dgModel.RelationshipResource, graph *resources.Graph) error {
	if resource.DisplayName == "" {
		return fmt.Errorf("display_name is required")
	}
	if resource.Cardinality == "" {
		return fmt.Errorf("cardinality is required")
	}
	if resource.DataGraphRef == nil {
		return fmt.Errorf("data_graph reference is required")
	}
	if resource.SourceModelRef == nil {
		return fmt.Errorf("source model reference is required")
	}
	if resource.TargetModelRef == nil {
		return fmt.Errorf("target model reference is required")
	}
	if resource.SourceJoinKey == "" {
		return fmt.Errorf("source_join_key is required")
	}
	if resource.TargetJoinKey == "" {
		return fmt.Errorf("target_join_key is required")
	}

	// Validate that the referenced data graph exists
	if _, exists := graph.GetResource(resource.DataGraphRef.URN); !exists {
		return fmt.Errorf("referenced data graph %s does not exist", resource.DataGraphRef.URN)
	}

	// Validate that the referenced models exist and get both for validation
	sourceModelRes, exists := graph.GetResource(resource.SourceModelRef.URN)
	if !exists {
		return fmt.Errorf("referenced source model %s does not exist", resource.SourceModelRef.URN)
	}
	targetModelRes, exists := graph.GetResource(resource.TargetModelRef.URN)
	if !exists {
		return fmt.Errorf("referenced target model %s does not exist", resource.TargetModelRef.URN)
	}

	// Get source and target models to determine relationship constraints
	sourceModel, ok := sourceModelRes.RawData().(*dgModel.ModelResource)
	if !ok {
		return fmt.Errorf("source model reference does not point to a valid model resource")
	}
	targetModel, ok := targetModelRes.RawData().(*dgModel.ModelResource)
	if !ok {
		return fmt.Errorf("target model reference does not point to a valid model resource")
	}

	// Validate cardinality based on source and target model types
	if sourceModel.Type == "event" {
		// Event models cannot connect to other event models
		if targetModel.Type == "event" {
			return fmt.Errorf("event models cannot be connected to other event models")
		}
		// Event models can only connect to entity models with many-to-one cardinality
		if resource.Cardinality != "many-to-one" {
			return fmt.Errorf("relationships from event models must have cardinality 'many-to-one', got %q", resource.Cardinality)
		}
	} else {
		// Entity models as source
		if targetModel.Type == "event" {
			// Entity to event relationships can only have one-to-many cardinality
			if resource.Cardinality != "one-to-many" {
				return fmt.Errorf("relationships from entity models to event models must have cardinality 'one-to-many', got %q", resource.Cardinality)
			}
		} else {
			// Entity to entity relationships can have any valid cardinality
			validCardinalities := map[string]bool{
				"one-to-one":  true,
				"one-to-many": true,
				"many-to-one": true,
			}
			if !validCardinalities[resource.Cardinality] {
				return fmt.Errorf("cardinality must be one of: one-to-one, one-to-many, many-to-one")
			}
		}
	}

	return nil
}

// listAllRelationshipsForDataGraph fetches all relationships for a data graph with pagination
func (h *HandlerImpl) listAllRelationshipsForDataGraph(ctx context.Context, dataGraphID string, hasExternalID *bool) ([]*dgModel.RemoteRelationship, error) {
	var allRelationships []*dgModel.RemoteRelationship
	page := 1
	perPage := 100

	for {
		resp, err := h.client.ListRelationships(ctx, &dgClient.ListRelationshipsRequest{
			DataGraphID:   dataGraphID,
			Page:          page,
			PageSize:      perPage,
			HasExternalID: hasExternalID,
		})
		if err != nil {
			return nil, fmt.Errorf("listing relationships: %w", err)
		}

		// Add all relationships from current page
		for i := range resp.Data {
			allRelationships = append(allRelationships, &dgModel.RemoteRelationship{
				Relationship: &resp.Data[i],
			})
		}

		// Check if there are more pages
		if resp.Paging.Next == "" {
			break
		}
		page++
	}

	return allRelationships, nil
}

func (h *HandlerImpl) LoadRemoteResources(ctx context.Context) ([]*dgModel.RemoteRelationship, error) {
	hasExternalID := true

	// First, get all data graphs with external IDs
	dataGraphs, err := h.listAllDataGraphs(ctx, &hasExternalID)
	if err != nil {
		return nil, err
	}

	// For each data graph, fetch all relationships with external IDs
	var allRelationships []*dgModel.RemoteRelationship
	for _, dg := range dataGraphs {
		relationships, err := h.listAllRelationshipsForDataGraph(ctx, dg.ID, &hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("loading relationships for data graph %s: %w", dg.ID, err)
		}
		allRelationships = append(allRelationships, relationships...)
	}

	return allRelationships, nil
}

func (h *HandlerImpl) LoadImportableResources(ctx context.Context) ([]*dgModel.RemoteRelationship, error) {
	hasExternalID := false

	// First, get all data graphs
	dataGraphs, err := h.listAllDataGraphs(ctx, nil)
	if err != nil {
		return nil, err
	}

	// For each data graph, fetch all relationships without external IDs
	var allRelationships []*dgModel.RemoteRelationship
	for _, dg := range dataGraphs {
		relationships, err := h.listAllRelationshipsForDataGraph(ctx, dg.ID, &hasExternalID)
		if err != nil {
			return nil, fmt.Errorf("loading importable relationships for data graph %s: %w", dg.ID, err)
		}
		allRelationships = append(allRelationships, relationships...)
	}

	return allRelationships, nil
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

func (h *HandlerImpl) MapRemoteToState(remote *dgModel.RemoteRelationship, urnResolver handler.URNResolver) (*dgModel.RelationshipResource, *dgModel.RelationshipState, error) {
	// Skip resources without external IDs
	if remote.ExternalID == "" {
		return nil, nil, nil
	}

	// Resolve the data graph's URN from its remote ID
	dataGraphURN, err := urnResolver.GetURNByID("data-graph", remote.DataGraphID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving data graph URN: %w", err)
	}

	// Resolve the from model's URN from its remote ID
	fromModelURN, err := urnResolver.GetURNByID(model.HandlerMetadata.ResourceType, remote.SourceModelID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving from model URN: %w", err)
	}

	// Resolve the to model's URN from its remote ID
	toModelURN, err := urnResolver.GetURNByID(model.HandlerMetadata.ResourceType, remote.TargetModelID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving to model URN: %w", err)
	}

	// Create PropertyRefs using the resolved URNs
	dataGraphRef := CreateDataGraphReference(dataGraphURN)
	fromModelRef := CreateModelReference(fromModelURN)
	toModelRef := CreateModelReference(toModelURN)

	resource := &dgModel.RelationshipResource{
		ID:             remote.ExternalID,
		DisplayName:    remote.Name,
		DataGraphRef:   dataGraphRef,
		SourceModelRef: fromModelRef,
		TargetModelRef: toModelRef,
		SourceJoinKey:  remote.SourceJoinKey,
		TargetJoinKey:  remote.TargetJoinKey,
		Cardinality:    remote.Cardinality,
	}

	state := &dgModel.RelationshipState{
		ID: remote.ID,
	}

	return resource, state, nil
}

func (h *HandlerImpl) Create(ctx context.Context, data *dgModel.RelationshipResource) (*dgModel.RelationshipState, error) {
	dataGraphRemoteID := data.DataGraphRef.Value
	fromModelRemoteID := data.SourceModelRef.Value
	toModelRemoteID := data.TargetModelRef.Value

	req := &dgClient.CreateRelationshipRequest{
		DataGraphID:   dataGraphRemoteID,
		Name:          data.DisplayName,
		Cardinality:   data.Cardinality,
		SourceModelID: fromModelRemoteID,
		TargetModelID: toModelRemoteID,
		SourceJoinKey: data.SourceJoinKey,
		TargetJoinKey: data.TargetJoinKey,
		ExternalID:    data.ID,
	}

	remote, err := h.client.CreateRelationship(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("creating relationship: %w", err)
	}

	return &dgModel.RelationshipState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Update(ctx context.Context, newData *dgModel.RelationshipResource, oldData *dgModel.RelationshipResource, oldState *dgModel.RelationshipState) (*dgModel.RelationshipState, error) {
	dataGraphRemoteID := oldData.DataGraphRef.Value
	fromModelRemoteID := newData.SourceModelRef.Value
	toModelRemoteID := newData.TargetModelRef.Value

	req := &dgClient.UpdateRelationshipRequest{
		DataGraphID:    dataGraphRemoteID,
		RelationshipID: oldState.ID,
		Name:           newData.DisplayName,
		Cardinality:    newData.Cardinality,
		SourceModelID:  fromModelRemoteID,
		TargetModelID:  toModelRemoteID,
		SourceJoinKey:  newData.SourceJoinKey,
		TargetJoinKey:  newData.TargetJoinKey,
	}

	remote, err := h.client.UpdateRelationship(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("updating relationship: %w", err)
	}

	return &dgModel.RelationshipState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Import(ctx context.Context, data *dgModel.RelationshipResource, remoteID string) (*dgModel.RelationshipState, error) {
	dataGraphRemoteID := data.DataGraphRef.Value

	// Set external ID on the remote resource
	_, setErr := h.client.SetRelationshipExternalID(ctx, &dgClient.SetRelationshipExternalIDRequest{
		DataGraphID:    dataGraphRemoteID,
		RelationshipID: remoteID,
		ExternalID:     data.ID,
	})
	if setErr != nil {
		return nil, fmt.Errorf("setting external ID: %w", setErr)
	}

	// Fetch the remote resource to get updated data
	remote, getErr := h.client.GetRelationship(ctx, &dgClient.GetRelationshipRequest{
		DataGraphID:    dataGraphRemoteID,
		RelationshipID: remoteID,
	})
	if getErr != nil {
		return nil, fmt.Errorf("getting relationship: %w", getErr)
	}

	return &dgModel.RelationshipState{
		ID: remote.ID,
	}, nil
}

func (h *HandlerImpl) Delete(ctx context.Context, id string, oldData *dgModel.RelationshipResource, oldState *dgModel.RelationshipState) error {
	dataGraphRemoteID := oldData.DataGraphRef.Value

	err := h.client.DeleteRelationship(ctx, &dgClient.DeleteRelationshipRequest{
		DataGraphID:    dataGraphRemoteID,
		RelationshipID: oldState.ID,
	})
	if err != nil {
		return fmt.Errorf("deleting relationship: %w", err)
	}
	return nil
}

// FormatForExport formats resources for export
// Relationships are exported inline in model specs, so this is not supported
func (h *HandlerImpl) FormatForExport(
	collection map[string]*dgModel.RemoteRelationship,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	// Relationships don't have standalone specs - they're exported inline in model specs
	return nil, nil
}

// CreateModelReference creates a PropertyRef that points to a model's remote ID
func CreateModelReference(urn string) *resources.PropertyRef {
	return handler.CreatePropertyRef(
		urn,
		func(state *dgModel.ModelState) (string, error) {
			return state.ID, nil
		},
	)
}

// CreateDataGraphReference creates a PropertyRef that points to a data graph's remote ID
func CreateDataGraphReference(urn string) *resources.PropertyRef {
	return handler.CreatePropertyRef(
		urn,
		func(state *dgModel.DataGraphState) (string, error) {
			return state.ID, nil
		},
	)
}
