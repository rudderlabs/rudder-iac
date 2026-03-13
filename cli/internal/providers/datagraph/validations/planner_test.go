package validations

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

func TestPlanner_PlanAll(t *testing.T) {
	models := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
		{ID: "purchase", Type: "event", Table: "cat.sch.purchases", Timestamp: "ts"},
	}
	relationships := []*dgModel.RelationshipResource{
		{ID: "user-purchases", Cardinality: "one-to-many"},
	}

	graph := newTestGraph(models, relationships)
	planner := NewPlanner(graph)

	plan, err := planner.BuildPlan(nil, ModeAll, "", "", differ.DiffOptions{})
	require.NoError(t, err)

	assert.Len(t, plan.Units, 3)

	var modelCount, relCount int
	for _, u := range plan.Units {
		switch u.ResourceType {
		case "model":
			modelCount++
		case "relationship":
			relCount++
		}
	}
	assert.Equal(t, 2, modelCount)
	assert.Equal(t, 1, relCount)
}

func TestPlanner_PlanAll_Empty(t *testing.T) {
	graph := resources.NewGraph()
	planner := NewPlanner(graph)

	plan, err := planner.BuildPlan(nil, ModeAll, "", "", differ.DiffOptions{})
	require.NoError(t, err)
	assert.Empty(t, plan.Units)
}

func TestPlanner_PlanSingle_Model(t *testing.T) {
	models := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}

	graph := newTestGraph(models, nil)
	planner := NewPlanner(graph)

	plan, err := planner.BuildPlan(nil, ModeSingle, "model", "user", differ.DiffOptions{})
	require.NoError(t, err)

	require.Len(t, plan.Units, 1)
	assert.Equal(t, "model", plan.Units[0].ResourceType)
	assert.Equal(t, "user", plan.Units[0].ID)
}

func TestPlanner_PlanSingle_Relationship(t *testing.T) {
	relationships := []*dgModel.RelationshipResource{
		{ID: "user-orders", Cardinality: "one-to-many"},
	}

	graph := newTestGraph(nil, relationships)
	planner := NewPlanner(graph)

	plan, err := planner.BuildPlan(nil, ModeSingle, "relationship", "user-orders", differ.DiffOptions{})
	require.NoError(t, err)

	require.Len(t, plan.Units, 1)
	assert.Equal(t, "relationship", plan.Units[0].ResourceType)
	assert.Equal(t, "user-orders", plan.Units[0].ID)
}

func TestPlanner_PlanSingle_NotFound(t *testing.T) {
	graph := resources.NewGraph()
	planner := NewPlanner(graph)

	_, err := planner.BuildPlan(nil, ModeSingle, "model", "nonexistent", differ.DiffOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPlanner_PlanSingle_InvalidType(t *testing.T) {
	graph := resources.NewGraph()
	planner := NewPlanner(graph)

	_, err := planner.BuildPlan(nil, ModeSingle, "invalid", "id", differ.DiffOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestPlanner_PlanModified(t *testing.T) {
	// Local graph has two models; remote only has one (same data)
	localModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
		{ID: "order", Type: "entity", Table: "cat.sch.orders", PrimaryID: "id"},
	}
	localGraph := newTestGraph(localModels, nil)

	remoteModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}
	remoteGraph := newTestGraph(remoteModels, nil)

	planner := NewPlanner(localGraph)
	plan, err := planner.BuildPlan(remoteGraph, ModeModified, "", "", differ.DiffOptions{})
	require.NoError(t, err)

	// Only "order" should be in the plan (new resource)
	require.Len(t, plan.Units, 1)
	assert.Equal(t, "order", plan.Units[0].ID)
}

func TestPlanner_PlanModified_Updated(t *testing.T) {
	// Same ID, different data
	localModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users_v2", PrimaryID: "id"},
	}
	localGraph := newTestGraph(localModels, nil)

	remoteModels := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}
	remoteGraph := newTestGraph(remoteModels, nil)

	planner := NewPlanner(localGraph)
	plan, err := planner.BuildPlan(remoteGraph, ModeModified, "", "", differ.DiffOptions{})
	require.NoError(t, err)

	require.Len(t, plan.Units, 1)
	assert.Equal(t, "user", plan.Units[0].ID)
}

func TestPlanner_PlanModified_NoChanges(t *testing.T) {
	models := []*dgModel.ModelResource{
		{ID: "user", Type: "entity", Table: "cat.sch.users", PrimaryID: "id"},
	}
	graph := newTestGraph(models, nil)

	planner := NewPlanner(graph)
	plan, err := planner.BuildPlan(graph, ModeModified, "", "", differ.DiffOptions{})
	require.NoError(t, err)

	assert.Empty(t, plan.Units)
}
