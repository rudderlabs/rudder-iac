package validations

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveAccountIDs(t *testing.T) {
	graph := resources.NewGraph()

	dgURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	dgRes := &dgModel.DataGraphResource{ID: "my-dg", AccountID: "acc-456"}
	graph.AddResource(resources.NewResource("my-dg", datagraph.HandlerMetadata.ResourceType,
		resources.ResourceData{"AccountID": "acc-456"}, nil, resources.WithRawData(dgRes)))

	runner := &Runner{graph: graph}

	plan := &ValidationPlan{
		Units: []*ValidationUnit{
			{
				ResourceType: "model",
				ID:           "user",
				Resource: &dgModel.ModelResource{
					ID:           "user",
					DataGraphRef: &resources.PropertyRef{URN: dgURN},
				},
			},
		},
	}

	err := runner.resolveAccountIDs(plan)
	require.NoError(t, err)
	assert.Equal(t, "acc-456", plan.Units[0].AccountID)
}

func TestResolveAccountIDs_NotInGraph(t *testing.T) {
	graph := resources.NewGraph()
	runner := &Runner{graph: graph}

	dgURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)

	plan := &ValidationPlan{
		Units: []*ValidationUnit{
			{
				ResourceType: "model",
				ID:           "user",
				Resource: &dgModel.ModelResource{
					ID:           "user",
					DataGraphRef: &resources.PropertyRef{URN: dgURN},
				},
			},
		},
	}

	err := runner.resolveAccountIDs(plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in local graph")
}

func TestFindDataGraphURN_Model(t *testing.T) {
	runner := &Runner{}
	dgURN := "data-graph:my-dg"

	unit := &ValidationUnit{
		ResourceType: "model",
		Resource: &dgModel.ModelResource{
			DataGraphRef: &resources.PropertyRef{URN: dgURN},
		},
	}

	assert.Equal(t, dgURN, runner.findDataGraphURN(unit))
}

func TestFindDataGraphURN_Relationship(t *testing.T) {
	runner := &Runner{}
	dgURN := "data-graph:my-dg"

	unit := &ValidationUnit{
		ResourceType: "relationship",
		Resource: &dgModel.RelationshipResource{
			DataGraphRef: &resources.PropertyRef{URN: dgURN},
		},
	}

	assert.Equal(t, dgURN, runner.findDataGraphURN(unit))
}
