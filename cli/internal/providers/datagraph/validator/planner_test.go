package validator

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGraph(models []*dgModel.ModelResource, relationships []*dgModel.RelationshipResource) *resources.Graph {
	g := resources.NewGraph()

	for _, m := range models {
		data := resources.ResourceData{
			"Type":  m.Type,
			"Table": m.Table,
		}
		r := resources.NewResource(m.ID, model.HandlerMetadata.ResourceType, data, nil, resources.WithRawData(m))
		g.AddResource(r)
	}

	for _, rel := range relationships {
		data := resources.ResourceData{
			"Cardinality": rel.Cardinality,
		}
		r := resources.NewResource(rel.ID, relationship.HandlerMetadata.ResourceType, data, nil, resources.WithRawData(rel))
		g.AddResource(r)
	}

	return g
}

func TestPlanAll(t *testing.T) {
	models := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
		{ID: "purchase", Type: "event", Table: "cat.sch.purchases", Timestamp: "ts"},
	}
	relationships := []*dgModel.RelationshipResource{
		{ID: "user-purchases", Cardinality: "one-to-many"},
	}

	graph := newTestGraph(models, relationships)

	plan, err := PlanAll(graph)
	require.NoError(t, err)

	assert.ElementsMatch(t, []*ValidationUnit{
		{
			ResourceType: "model",
			ID:           "user",
			URN:          "data-graph-model:user",
			Resource:     models[0],
		},
		{
			ResourceType: "model",
			ID:           "purchase",
			URN:          "data-graph-model:purchase",
			Resource:     models[1],
		},
		{
			ResourceType: "relationship",
			ID:           "user-purchases",
			URN:          "data-graph-relationship:user-purchases",
			Resource:     relationships[0],
		},
	}, plan.Units)
}

func TestPlanAll_Empty(t *testing.T) {
	graph := resources.NewGraph()

	plan, err := PlanAll(graph)
	require.NoError(t, err)
	assert.Empty(t, plan.Units)
}

func TestPlanSingle_Model(t *testing.T) {
	models := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}

	graph := newTestGraph(models, nil)

	plan, err := PlanSingle(graph, "model", "user")
	require.NoError(t, err)

	assert.Equal(t, []*ValidationUnit{
		{
			ResourceType: "model",
			ID:           "user",
			URN:          "data-graph-model:user",
			Resource:     models[0],
		},
	}, plan.Units)
}

func TestPlanSingle_Relationship(t *testing.T) {
	relationships := []*dgModel.RelationshipResource{
		{ID: "user-orders", Cardinality: "one-to-many"},
	}

	graph := newTestGraph(nil, relationships)

	plan, err := PlanSingle(graph, "relationship", "user-orders")
	require.NoError(t, err)

	assert.Equal(t, []*ValidationUnit{
		{
			ResourceType: "relationship",
			ID:           "user-orders",
			URN:          "data-graph-relationship:user-orders",
			Resource:     relationships[0],
		},
	}, plan.Units)
}

func TestPlanSingle_NotFound(t *testing.T) {
	graph := resources.NewGraph()

	_, err := PlanSingle(graph, "model", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPlanSingle_InvalidType(t *testing.T) {
	graph := resources.NewGraph()

	_, err := PlanSingle(graph, "invalid", "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestPlanModified(t *testing.T) {
	localModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
		{ID: "order", Type: "entity", Table: "cat.sch.orders", PrimaryID: "id"},
	}
	localGraph := newTestGraph(localModels, nil)

	remoteModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}
	remoteGraph := newTestGraph(remoteModels, nil)

	plan, err := PlanModified(localGraph, remoteGraph, differ.DiffOptions{})
	require.NoError(t, err)

	// Only "order" should be in the plan (new resource)
	assert.Equal(t, []*ValidationUnit{
		{
			ResourceType: "model",
			ID:           "order",
			URN:          "data-graph-model:order",
			Resource:     localModels[1],
		},
	}, plan.Units)
}

func TestPlanModified_Updated(t *testing.T) {
	localModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users_v2", PrimaryID: "id"},
	}
	localGraph := newTestGraph(localModels, nil)

	remoteModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}
	remoteGraph := newTestGraph(remoteModels, nil)

	plan, err := PlanModified(localGraph, remoteGraph, differ.DiffOptions{})
	require.NoError(t, err)

	assert.Equal(t, []*ValidationUnit{
		{
			ResourceType: "model",
			ID:           "user",
			URN:          "data-graph-model:user",
			Resource:     localModels[0],
		},
	}, plan.Units)
}

func TestPlanModified_NoChanges(t *testing.T) {
	models := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}
	graph := newTestGraph(models, nil)

	plan, err := PlanModified(graph, graph, differ.DiffOptions{})
	require.NoError(t, err)

	assert.Empty(t, plan.Units)
}
