package validations

import (
	"context"
	"fmt"
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/testutils"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateModel_Success(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ValidateModelFunc: func(ctx context.Context, req *dgClient.ValidateModelRequest) (*dgClient.ValidationReport, error) {
			assert.Equal(t, "dg-remote-123", req.DataGraphID)
			assert.Equal(t, "entity", req.Type)
			assert.Equal(t, "cat.sch.users", req.TableRef)
			return &dgClient.ValidationReport{
				Issues: []dgClient.ValidationIssue{
					{Rule: "model/table-exists", Severity: "warning", Message: "Table has no recent data"},
				},
			}, nil
		},
	}

	unit := &ValidationUnit{
		ResourceType: "model",
		ID:           "user",
		DataGraphID:  "dg-remote-123",
		Resource: &dgModel.ModelResource{
			ID:          "user",
			DisplayName: "User",
			Type:        "entity",
			Table:       "cat.sch.users",
			PrimaryID:   "id",
		},
	}

	result := executeValidation(context.Background(), mockClient, nil, unit)

	require.NoError(t, result.Err)
	assert.Equal(t, "user", result.ID)
	assert.Equal(t, "User", result.DisplayName)
	assert.Equal(t, "model", result.ResourceType)
	assert.Len(t, result.Issues, 1)
	assert.Equal(t, "warning", result.Issues[0].Severity)
}

func TestValidateModel_APIError(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ValidateModelFunc: func(ctx context.Context, req *dgClient.ValidateModelRequest) (*dgClient.ValidationReport, error) {
			return nil, fmt.Errorf("api error")
		},
	}

	unit := &ValidationUnit{
		ResourceType: "model",
		ID:           "user",
		DataGraphID:  "dg-remote-123",
		Resource: &dgModel.ModelResource{
			ID:          "user",
			DisplayName: "User",
			Type:        "entity",
			Table:       "cat.sch.users",
			PrimaryID:   "id",
		},
	}

	result := executeValidation(context.Background(), mockClient, nil, unit)
	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "api error")
}

func TestValidateRelationship_Success(t *testing.T) {
	// Build a graph with the source and target models
	sourceModel := &dgModel.ModelResource{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"}
	targetModel := &dgModel.ModelResource{ID: "order", Type: "entity", Table: "cat.sch.orders", PrimaryID: "id"}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("user", model.HandlerMetadata.ResourceType,
		resources.ResourceData{"Type": "entity"}, nil, resources.WithRawData(sourceModel)))
	graph.AddResource(resources.NewResource("order", model.HandlerMetadata.ResourceType,
		resources.ResourceData{"Type": "entity"}, nil, resources.WithRawData(targetModel)))

	mockClient := &testutils.MockDataGraphClient{
		ValidateRelationshipFunc: func(ctx context.Context, req *dgClient.ValidateRelationshipRequest) (*dgClient.ValidationReport, error) {
			assert.Equal(t, "dg-remote-123", req.DataGraphID)
			assert.Equal(t, "one-to-many", req.Cardinality)
			assert.Equal(t, "cat.sch.users", req.SourceModel.TableRef)
			assert.Equal(t, "id", req.SourceModel.JoinKey)
			assert.Equal(t, "cat.sch.orders", req.TargetModel.TableRef)
			assert.Equal(t, "user_id", req.TargetModel.JoinKey)
			return &dgClient.ValidationReport{Issues: []dgClient.ValidationIssue{}}, nil
		},
	}

	sourceURN := resources.URN("user", model.HandlerMetadata.ResourceType)
	targetURN := resources.URN("order", model.HandlerMetadata.ResourceType)

	unit := &ValidationUnit{
		ResourceType: "relationship",
		ID:           "user-orders",
		DataGraphID:  "dg-remote-123",
		Resource: &dgModel.RelationshipResource{
			ID:             "user-orders",
			DisplayName:    "User Orders",
			Cardinality:    "one-to-many",
			SourceModelRef: &resources.PropertyRef{URN: sourceURN},
			TargetModelRef: &resources.PropertyRef{URN: targetURN},
			SourceJoinKey:  "id",
			TargetJoinKey:  "user_id",
		},
	}

	result := executeValidation(context.Background(), mockClient, graph, unit)

	require.NoError(t, result.Err)
	assert.Equal(t, "user-orders", result.ID)
	assert.Equal(t, "relationship", result.ResourceType)
	assert.Empty(t, result.Issues)
}

func TestValidateRelationship_ModelNotInGraph(t *testing.T) {
	graph := resources.NewGraph() // empty graph

	mockClient := &testutils.MockDataGraphClient{}

	unit := &ValidationUnit{
		ResourceType: "relationship",
		ID:           "user-orders",
		DataGraphID:  "dg-remote-123",
		Resource: &dgModel.RelationshipResource{
			ID:             "user-orders",
			DisplayName:    "User Orders",
			Cardinality:    "one-to-many",
			SourceModelRef: &resources.PropertyRef{URN: "data-graph-model:user"},
			TargetModelRef: &resources.PropertyRef{URN: "data-graph-model:order"},
			SourceJoinKey:  "id",
			TargetJoinKey:  "user_id",
		},
	}

	result := executeValidation(context.Background(), mockClient, graph, unit)
	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "resolving source model table ref")
}

func TestRunValidationTasks_Concurrent(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ValidateModelFunc: func(ctx context.Context, req *dgClient.ValidateModelRequest) (*dgClient.ValidationReport, error) {
			return &dgClient.ValidationReport{Issues: []dgClient.ValidationIssue{}}, nil
		},
	}

	units := []*ValidationUnit{
		{
			ResourceType: "model",
			ID:           "model-1",
			DataGraphID:  "dg-123",
			Resource:     &dgModel.ModelResource{ID: "model-1", DisplayName: "M1", Type: "entity", Table: "cat.sch.t1", PrimaryID: "id"},
		},
		{
			ResourceType: "model",
			ID:           "model-2",
			DataGraphID:  "dg-123",
			Resource:     &dgModel.ModelResource{ID: "model-2", DisplayName: "M2", Type: "event", Table: "cat.sch.t2", Timestamp: "ts"},
		},
	}

	results := runValidationTasks(context.Background(), mockClient, nil, units)

	assert.Len(t, results, 2)
	assert.Equal(t, "model-1", results[0].ID)
	assert.Equal(t, "model-2", results[1].ID)
}
