package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRelationshipRefsValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewRelationshipRefsValidRule()

	assert.Equal(t, "datagraph/data-graph/relationship-refs-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())

	expectedPatterns := append(
		prules.LegacyVersionPatterns("data-graph"),
		prules.V1VersionPatterns("data-graph")...,
	)
	assert.Equal(t, expectedPatterns, rule.AppliesTo())
}

func TestRelationshipRefsValid_TargetExistsLocally(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        "test-dg",
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
						ID:            "user-account",
						DisplayName:   "User Account",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "account_id",
					},
				},
			},
			{
				ID:          "account",
				DisplayName: "Account",
				Type:        "entity",
				Table:       "db.schema.accounts",
				PrimaryID:   "id",
			},
		},
	}

	graph := resources.NewGraph()
	results := validateRelationshipRefs("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results, "Target exists locally — should not produce errors")
}

func TestRelationshipRefsValid_TargetExistsInGraph(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        "test-dg",
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
						ID:            "user-account",
						DisplayName:   "User Account",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "account_id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("account", modelHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
	))

	results := validateRelationshipRefs("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results, "Target exists in graph — should not produce errors")
}

func TestRelationshipRefsValid_TargetMissing(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        "test-dg",
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
						ID:            "user-account",
						DisplayName:   "User Account",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:non-existent",
						SourceJoinKey: "account_id",
						TargetJoinKey: "account_id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	results := validateRelationshipRefs("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 1)
	assert.Equal(t, "/models/0/relationships/0/target", results[0].Reference)
	assert.Contains(t, results[0].Message, "non-existent")
	assert.Contains(t, results[0].Message, "does not exist")
}

func TestRelationshipRefsValid_MultipleFailures(t *testing.T) {
	t.Parallel()

	spec := dgModel.DataGraphSpec{
		ID:        "test-dg",
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
						ID:            "rel-1",
						DisplayName:   "Rel 1",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:missing-a",
						SourceJoinKey: "key",
						TargetJoinKey: "key",
					},
					{
						ID:            "rel-2",
						DisplayName:   "Rel 2",
						Cardinality:   "one-to-many",
						Target:        "#data-graph-model:missing-b",
						SourceJoinKey: "key",
						TargetJoinKey: "key",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	results := validateRelationshipRefs("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 2)
	assert.Equal(t, "/models/0/relationships/0/target", results[0].Reference)
	assert.Equal(t, "/models/0/relationships/1/target", results[1].Reference)
}
