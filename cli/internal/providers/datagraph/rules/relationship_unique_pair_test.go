package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	modelHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relationshipHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRelationshipUniquePairRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewRelationshipUniquePairRule()

	assert.Equal(t, "datagraph/data-graph/relationship-unique-pair", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, prules.V1VersionPatterns("data-graph"), rule.AppliesTo())
}

func TestRelationshipUniquePair_NoRelationships(t *testing.T) {
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
			},
		},
	}

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, resources.NewGraph())
	assert.Empty(t, results)
}

func TestRelationshipUniquePair_SingleRelationship(t *testing.T) {
	t.Parallel()

	var (
		userURN    = resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		accountURN = resources.URN("account", modelHandler.HandlerMetadata.ResourceType)
	)

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
	graph.AddResource(resources.NewResource("user-account", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-account",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
		}),
	))

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results)
}

func TestRelationshipUniquePair_DuplicateInSameModel(t *testing.T) {
	t.Parallel()

	var (
		userURN    = resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		accountURN = resources.URN("account", modelHandler.HandlerMetadata.ResourceType)
	)

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
						ID:            "user-account-v1",
						DisplayName:   "User Account v1",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "account_id",
					},
					{
						ID:            "user-account-v2",
						DisplayName:   "User Account v2",
						Cardinality:   "one-to-many",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("user-account-v1", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-account-v1",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
		}),
	))
	graph.AddResource(resources.NewResource("user-account-v2", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-account-v2",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
		}),
	))

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 1)
	assert.Equal(t, "/models/0/relationships/1", results[0].Reference)
	assert.Contains(t, results[0].Message, "user")
	assert.Contains(t, results[0].Message, "account")
}

func TestRelationshipUniquePair_DifferentTargets(t *testing.T) {
	t.Parallel()

	var (
		userURN    = resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		accountURN = resources.URN("account", modelHandler.HandlerMetadata.ResourceType)
		profileURN = resources.URN("profile", modelHandler.HandlerMetadata.ResourceType)
	)

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
					{
						ID:            "user-profile",
						DisplayName:   "User Profile",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:profile",
						SourceJoinKey: "profile_id",
						TargetJoinKey: "profile_id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("user-account", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-account",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
		}),
	))
	graph.AddResource(resources.NewResource("user-profile", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-profile",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: profileURN},
		}),
	))

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results)
}

func TestRelationshipUniquePair_ReversedPairAllowed(t *testing.T) {
	t.Parallel()

	var (
		userURN    = resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		accountURN = resources.URN("account", modelHandler.HandlerMetadata.ResourceType)
	)

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
						ID:            "user-to-account",
						DisplayName:   "User to Account",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "id",
					},
				},
			},
			{
				ID:          "account",
				DisplayName: "Account",
				Type:        "entity",
				Table:       "db.schema.accounts",
				PrimaryID:   "id",
				Relationships: []dgModel.RelationshipSpec{
					{
						ID:            "account-to-user",
						DisplayName:   "Account to User",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:user",
						SourceJoinKey: "id",
						TargetJoinKey: "account_id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("user-to-account", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-to-account",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
		}),
	))
	graph.AddResource(resources.NewResource("account-to-user", relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "account-to-user",
			SourceModelRef: &resources.PropertyRef{URN: accountURN},
			TargetModelRef: &resources.PropertyRef{URN: userURN},
		}),
	))

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, graph)
	assert.Empty(t, results, "A->B and B->A are distinct pairs and should both be allowed")
}

func TestRelationshipUniquePair_ConflictWithExistingGraphRelationship(t *testing.T) {
	t.Parallel()

	var (
		userURN    = resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		accountURN = resources.URN("account", modelHandler.HandlerMetadata.ResourceType)
	)

	graph := resources.NewGraph()
	// Existing relationship from another spec (not in specRelIDs)
	graph.AddResource(resources.NewResource(
		"existing-rel",
		relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{},
		nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "existing-rel",
			DisplayName:    "Existing Rel",
			Cardinality:    "one-to-one",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
			SourceJoinKey:  "account_id",
			TargetJoinKey:  "id",
		}),
	))
	// The spec's own relationship is also in the graph
	graph.AddResource(resources.NewResource(
		"user-account-new",
		relationshipHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{},
		nil,
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:             "user-account-new",
			DisplayName:    "User Account New",
			Cardinality:    "one-to-many",
			SourceModelRef: &resources.PropertyRef{URN: userURN},
			TargetModelRef: &resources.PropertyRef{URN: accountURN},
			SourceJoinKey:  "account_id",
			TargetJoinKey:  "id",
		}),
	))

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
						ID:            "user-account-new",
						DisplayName:   "User Account New",
						Cardinality:   "one-to-many",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "id",
					},
				},
			},
		},
	}

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 1)
	assert.Equal(t, "/models/0/relationships/0", results[0].Reference)
	assert.Contains(t, results[0].Message, "user")
	assert.Contains(t, results[0].Message, "account")
}

func TestRelationshipUniquePair_MultipleDuplicates(t *testing.T) {
	t.Parallel()

	var (
		userURN    = resources.URN("user", modelHandler.HandlerMetadata.ResourceType)
		accountURN = resources.URN("account", modelHandler.HandlerMetadata.ResourceType)
		profileURN = resources.URN("profile", modelHandler.HandlerMetadata.ResourceType)
	)

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
						ID:            "dup-account-1",
						DisplayName:   "Account 1",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "account_id",
					},
					{
						ID:            "dup-account-2",
						DisplayName:   "Account 2",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:account",
						SourceJoinKey: "account_id",
						TargetJoinKey: "account_id",
					},
					{
						ID:            "dup-profile-1",
						DisplayName:   "Profile 1",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:profile",
						SourceJoinKey: "profile_id",
						TargetJoinKey: "profile_id",
					},
					{
						ID:            "dup-profile-2",
						DisplayName:   "Profile 2",
						Cardinality:   "one-to-one",
						Target:        "#data-graph-model:profile",
						SourceJoinKey: "profile_id",
						TargetJoinKey: "profile_id",
					},
				},
			},
		},
	}

	graph := resources.NewGraph()
	for _, relID := range []string{"dup-account-1", "dup-account-2"} {
		graph.AddResource(resources.NewResource(relID, relationshipHandler.HandlerMetadata.ResourceType,
			resources.ResourceData{}, nil,
			resources.WithRawData(&dgModel.RelationshipResource{
				ID:             relID,
				SourceModelRef: &resources.PropertyRef{URN: userURN},
				TargetModelRef: &resources.PropertyRef{URN: accountURN},
			}),
		))
	}
	for _, relID := range []string{"dup-profile-1", "dup-profile-2"} {
		graph.AddResource(resources.NewResource(relID, relationshipHandler.HandlerMetadata.ResourceType,
			resources.ResourceData{}, nil,
			resources.WithRawData(&dgModel.RelationshipResource{
				ID:             relID,
				SourceModelRef: &resources.PropertyRef{URN: userURN},
				TargetModelRef: &resources.PropertyRef{URN: profileURN},
			}),
		))
	}

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, graph)

	require.Len(t, results, 2)
	assert.Equal(t, "/models/0/relationships/1", results[0].Reference)
	assert.Contains(t, results[0].Message, "account")
	assert.Equal(t, "/models/0/relationships/3", results[1].Reference)
	assert.Contains(t, results[1].Message, "profile")
}

func TestRelationshipUniquePair_RelationshipNotInGraph(t *testing.T) {
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
						ID:            "bad-ref",
						DisplayName:   "Bad Ref",
						Cardinality:   "one-to-one",
						Target:        "invalid-format",
						SourceJoinKey: "id",
						TargetJoinKey: "id",
					},
				},
			},
		},
	}

	results := validateRelationshipUniquePair("data-graph", specs.SpecVersionV1, nil, spec, resources.NewGraph())
	assert.Empty(t, results, "relationships not in graph should be skipped")
}
