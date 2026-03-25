package validator

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
			assert.Equal(t, "acc-123", req.AccountID)
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
		URN:          "data-graph-model:user",
		AccountID:    "acc-123",
		Resource: &dgModel.ModelResource{
			ID:          "user",
			DisplayName: "User",
			Type:        "entity",
			Table:       "cat.sch.users",
			PrimaryID:   "id",
		},
	}

	result := executeValidation(context.Background(), mockClient, nil, unit)

	assert.Equal(t, &ResourceValidation{
		ID:           "user",
		URN:          "data-graph-model:user",
		DisplayName:  "User",
		ResourceType: "model",
		Issues: []dgClient.ValidationIssue{
			{Rule: "model/table-exists", Severity: "warning", Message: "Table has no recent data"},
		},
	}, result)
}

func TestValidateModel_APIError(t *testing.T) {
	apiErr := fmt.Errorf("api error")
	mockClient := &testutils.MockDataGraphClient{
		ValidateModelFunc: func(ctx context.Context, req *dgClient.ValidateModelRequest) (*dgClient.ValidationReport, error) {
			return nil, apiErr
		},
	}

	unit := &ValidationUnit{
		ResourceType: "model",
		ID:           "user",
		URN:          "data-graph-model:user",
		AccountID:    "acc-123",
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
	sourceModel := &dgModel.ModelResource{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"}
	targetModel := &dgModel.ModelResource{ID: "order", Type: "entity", Table: "cat.sch.orders", PrimaryID: "id"}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("user", model.HandlerMetadata.ResourceType,
		resources.ResourceData{"Type": "entity"}, nil, resources.WithRawData(sourceModel)))
	graph.AddResource(resources.NewResource("order", model.HandlerMetadata.ResourceType,
		resources.ResourceData{"Type": "entity"}, nil, resources.WithRawData(targetModel)))

	mockClient := &testutils.MockDataGraphClient{
		ValidateRelationshipFunc: func(ctx context.Context, req *dgClient.ValidateRelationshipRequest) (*dgClient.ValidationReport, error) {
			assert.Equal(t, "acc-123", req.AccountID)
			assert.Equal(t, "one-to-many", req.Cardinality)
			assert.Equal(t, "cat.sch.users", req.SourceModel.TableRef)
			assert.Equal(t, "id", req.SourceModel.JoinKey)
			assert.Equal(t, "cat.sch.orders", req.TargetModel.TableRef)
			assert.Equal(t, "user_id", req.TargetModel.JoinKey)
			return &dgClient.ValidationReport{Issues: []dgClient.ValidationIssue{}}, nil
		},
	}

	var (
		sourceURN = resources.URN("user", model.HandlerMetadata.ResourceType)
		targetURN = resources.URN("order", model.HandlerMetadata.ResourceType)
	)

	unit := &ValidationUnit{
		ResourceType: "relationship",
		ID:           "user-orders",
		URN:          "data-graph-relationship:user-orders",
		AccountID:    "acc-123",
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

	assert.Equal(t, &ResourceValidation{
		ID:           "user-orders",
		URN:          "data-graph-relationship:user-orders",
		DisplayName:  "User Orders",
		ResourceType: "relationship",
		Issues:       []dgClient.ValidationIssue{},
	}, result)
}

func TestValidateRelationship_ModelNotInGraph(t *testing.T) {
	graph := resources.NewGraph()

	mockClient := &testutils.MockDataGraphClient{}

	unit := &ValidationUnit{
		ResourceType: "relationship",
		ID:           "user-orders",
		URN:          "data-graph-relationship:user-orders",
		AccountID:    "acc-123",
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
			URN:          "data-graph-model:model-1",
			AccountID:    "acc-123",
			Resource:     &dgModel.ModelResource{ID: "model-1", DisplayName: "M1", Type: "entity", Table: "cat.sch.t1", PrimaryID: "id"},
		},
		{
			ResourceType: "model",
			ID:           "model-2",
			URN:          "data-graph-model:model-2",
			AccountID:    "acc-123",
			Resource:     &dgModel.ModelResource{ID: "model-2", DisplayName: "M2", Type: "event", Table: "cat.sch.t2", Timestamp: "ts"},
		},
	}

	results := runValidationTasks(context.Background(), mockClient, nil, units, noopReporter{}, 4)

	assert.Len(t, results, 2)
	assert.Equal(t, "model-1", results[0].ID)
	assert.Equal(t, "model-2", results[1].ID)
}

func TestRunValidationTasks_ReporterCalled(t *testing.T) {
	mockClient := &testutils.MockDataGraphClient{
		ValidateModelFunc: func(ctx context.Context, req *dgClient.ValidateModelRequest) (*dgClient.ValidationReport, error) {
			return &dgClient.ValidationReport{Issues: []dgClient.ValidationIssue{}}, nil
		},
	}

	units := []*ValidationUnit{
		{
			ResourceType: "model",
			ID:           "user",
			URN:          "data-graph-model:user",
			AccountID:    "acc-123",
			Resource:     &dgModel.ModelResource{ID: "user", DisplayName: "User", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
		},
	}

	reporter := &mockReporter{}
	runValidationTasks(context.Background(), mockClient, nil, units, reporter, 4)

	assert.Equal(t, []string{"data-graph-model:user"}, reporter.started)
	assert.Equal(t, []string{"data-graph-model:user"}, reporter.completed)
}

type mockReporter struct {
	started   []string
	completed []string
}

func (m *mockReporter) TaskStarted(id, _ string)       { m.started = append(m.started, id) }
func (m *mockReporter) TaskCompleted(id, _ string, _ error) { m.completed = append(m.completed, id) }
