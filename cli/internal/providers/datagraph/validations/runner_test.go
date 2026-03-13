package validations

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveDataGraphIDs(t *testing.T) {
	runner := &Runner{graph: resources.NewGraph()}

	dgURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)

	remoteState := state.EmptyState()
	remoteState.AddResource(&state.ResourceState{
		ID:        "my-dg",
		Type:      datagraph.HandlerMetadata.ResourceType,
		OutputRaw: &dgModel.DataGraphState{ID: "dg-remote-456"},
	})

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

	err := runner.resolveDataGraphIDs(plan, remoteState)
	require.NoError(t, err)
	assert.Equal(t, "dg-remote-456", plan.Units[0].DataGraphID)
}

func TestResolveDataGraphIDs_NotSynced(t *testing.T) {
	runner := &Runner{graph: resources.NewGraph()}

	dgURN := resources.URN("my-dg", datagraph.HandlerMetadata.ResourceType)
	remoteState := state.EmptyState()

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

	err := runner.resolveDataGraphIDs(plan, remoteState)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has not been synced yet")
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
