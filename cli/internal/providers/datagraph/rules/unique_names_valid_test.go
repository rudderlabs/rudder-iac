package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDataGraphID = "test-dg"

var testDataGraphURN = resources.URN(testDataGraphID, "data-graph")

func addModelToGraph(graph *resources.Graph, id, displayName string) {
	graph.AddResource(resources.NewResource(id, modelHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.ModelResource{
			ID:          id,
			DisplayName: displayName,
			DataGraphRef: &resources.PropertyRef{
				URN: testDataGraphURN,
			},
		}),
	))
}

func addRelationshipToGraph(graph *resources.Graph, id, displayName string) {
	graph.AddResource(resources.NewResource(id, relHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:          id,
			DisplayName: displayName,
			DataGraphRef: &resources.PropertyRef{
				URN: testDataGraphURN,
			},
		}),
	))
}

func TestNewUniqueNamesValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewUniqueNamesValidRule()

	assert.Equal(t, "datagraph/data-graph/unique-names-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, prules.V1VersionPatterns("data-graph"), rule.AppliesTo())
}

func TestUniqueNamesValid_AllUnique(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
				Relationships: []dgModel.RelationshipSpec{
					{
						ID:            "user-order",
						DisplayName:   "User Order",
						Cardinality:   "one-to-many",
						Target:        "#data-graph-model:order",
						SourceJoinKey: "id",
						TargetJoinKey: "user_id",
					},
				},
			},
			{
				ID:          "order",
				DisplayName: "Order",
				Type:        "entity",
				Table:       "db.schema.orders",
				PrimaryID:   "id",
			},
		},
	}

	graph := resources.NewGraph()
	addModelToGraph(graph, "user", "User")
	addModelToGraph(graph, "order", "Order")
	addRelationshipToGraph(graph, "user-order", "User Order")

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results)
}

func TestUniqueNamesValid_DuplicateModelDisplayNameWithinSpec(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
			},
			{
				ID:          "user-v2",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users_v2",
				PrimaryID:   "id",
			},
		},
	}

	graph := resources.NewGraph()
	addModelToGraph(graph, "user", "User")
	addModelToGraph(graph, "user-v2", "User")

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 2)
	assert.Equal(t, "/models/0/display_name", results[0].Reference)
	assert.Contains(t, results[0].Message, `duplicate model display name "User"`)
	assert.Equal(t, "/models/1/display_name", results[1].Reference)
	assert.Contains(t, results[1].Message, `duplicate model display name "User"`)
}

func TestUniqueNamesValid_DuplicateRelationshipDisplayNameWithinSpec(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
				Relationships: []dgModel.RelationshipSpec{
					{
						ID:            "user-order",
						DisplayName:   "Links To",
						Cardinality:   "one-to-many",
						Target:        "#data-graph-model:order",
						SourceJoinKey: "id",
						TargetJoinKey: "user_id",
					},
				},
			},
			{
				ID:          "order",
				DisplayName: "Order",
				Type:        "entity",
				Table:       "db.schema.orders",
				PrimaryID:   "id",
				Relationships: []dgModel.RelationshipSpec{
					{
						ID:            "order-product",
						DisplayName:   "Links To",
						Cardinality:   "one-to-many",
						Target:        "#data-graph-model:product",
						SourceJoinKey: "id",
						TargetJoinKey: "order_id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	addModelToGraph(graph, "user", "User")
	addModelToGraph(graph, "order", "Order")
	addRelationshipToGraph(graph, "user-order", "Links To")
	addRelationshipToGraph(graph, "order-product", "Links To")

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 2)
	assert.Equal(t, "/models/0/relationships/0/display_name", results[0].Reference)
	assert.Contains(t, results[0].Message, `duplicate relationship display name "Links To"`)
	assert.Equal(t, "/models/1/relationships/0/display_name", results[1].Reference)
	assert.Contains(t, results[1].Message, `duplicate relationship display name "Links To"`)
}

func TestUniqueNamesValid_CrossSpecModelDuplicate(t *testing.T) {
	t.Parallel()

	// Spec defines a model with display name "User" but different ID
	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user-v2",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users_v2",
				PrimaryID:   "id",
			},
		},
	}

	// Graph already has a model with display name "User" from another spec
	graph := resources.NewGraph()
	addModelToGraph(graph, "user", "User")
	addModelToGraph(graph, "user-v2", "User")

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 1)
	assert.Equal(t, "/models/0/display_name", results[0].Reference)
	assert.Contains(t, results[0].Message, `duplicate model display name "User"`)
}

func TestUniqueNamesValid_CrossSpecRelationshipDuplicate(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "order",
				DisplayName: "Order",
				Type:        "entity",
				Table:       "db.schema.orders",
				PrimaryID:   "id",
				Relationships: []dgModel.RelationshipSpec{
					{
						ID:            "order-user",
						DisplayName:   "Belongs To",
						Cardinality:   "many-to-one",
						Target:        "#data-graph-model:user",
						SourceJoinKey: "user_id",
						TargetJoinKey: "id",
					},
				},
			},
		},
	}

	// Graph has an existing relationship with the same display name from another spec
	graph := resources.NewGraph()
	addModelToGraph(graph, "order", "Order")
	addRelationshipToGraph(graph, "product-user", "Belongs To")
	addRelationshipToGraph(graph, "order-user", "Belongs To")

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 1)
	assert.Equal(t, "/models/0/relationships/0/display_name", results[0].Reference)
	assert.Contains(t, results[0].Message, `duplicate relationship display name "Belongs To"`)
}

func TestUniqueNamesValid_SelfMatchNotFlagged(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
			},
		},
	}

	// Graph contains the same model (same URN) — should not flag as duplicate
	graph := resources.NewGraph()
	addModelToGraph(graph, "user", "User")

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results)
}

func TestUniqueNamesValid_DifferentDataGraphNotFlagged(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        testDataGraphID,
		AccountID: "wh-123",
		Models: []dgModel.ModelSpec{
			{
				ID:          "user",
				DisplayName: "User",
				Type:        "entity",
				Table:       "db.schema.users",
				PrimaryID:   "id",
			},
		},
	}

	graph := resources.NewGraph()
	addModelToGraph(graph, "user", "User")
	// Model with same display name but belonging to a different data graph
	graph.AddResource(resources.NewResource("other-user", modelHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.ModelResource{
			ID:          "other-user",
			DisplayName: "User",
			DataGraphRef: &resources.PropertyRef{
				URN: resources.URN("other-dg", "data-graph"),
			},
		}),
	))

	results := validateUniqueNames("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results)
}
