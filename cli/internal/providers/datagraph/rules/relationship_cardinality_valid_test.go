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

func TestNewRelationshipCardinalityValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewRelationshipCardinalityValidRule()

	assert.Equal(t, "datagraph/data-graph/relationship-cardinality-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())

	assert.Equal(t, prules.V1VersionPatterns("data-graph"), rule.AppliesTo())
}

// buildCardinalityTest creates a DataGraphSpec and a graph with the target model for cardinality testing
func buildCardinalityTest(sourceType, targetType, cardinality string) (dgModel.DataGraphSpec, *resources.Graph) {
	sourceModel := dgModel.ModelSpec{
		ID:          "source-model",
		DisplayName: "Source Model",
		Type:        sourceType,
		Table:       "db.schema.source_table",
		Relationships: []dgModel.RelationshipSpec{
			{
				ID:            "test-rel",
				DisplayName:   "Test Relationship",
				Cardinality:   cardinality,
				Target:        "#data-graph-model:target-model",
				SourceJoinKey: "join_key",
				TargetJoinKey: "join_key",
			},
		},
	}

	if sourceType == "entity" {
		sourceModel.PrimaryID = "id"
	} else {
		sourceModel.Timestamp = "created_at"
	}

	spec := dgModel.DataGraphSpec{
		ID:        "test-dg",
		AccountID: "wh-123",
		Models:    []dgModel.ModelSpec{sourceModel},
	}

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("target-model", modelHandler.HandlerMetadata.ResourceType,
		resources.ResourceData{}, nil,
		resources.WithRawData(&dgModel.ModelResource{
			ID:   "target-model",
			Type: targetType,
		}),
	))

	return spec, graph
}

func TestRelationshipCardinalityValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		sourceType    string
		targetType    string
		cardinality   string
		shouldSucceed bool
		errorContains string
	}{
		{
			name:          "event to event - rejected",
			sourceType:    "event",
			targetType:    "event",
			cardinality:   "many-to-one",
			shouldSucceed: false,
			errorContains: "event models cannot be connected to other event models",
		},
		{
			name:          "event to entity - valid many-to-one",
			sourceType:    "event",
			targetType:    "entity",
			cardinality:   "many-to-one",
			shouldSucceed: true,
		},
		{
			name:          "event to entity - invalid one-to-many",
			sourceType:    "event",
			targetType:    "entity",
			cardinality:   "one-to-many",
			shouldSucceed: false,
			errorContains: "must have cardinality 'many-to-one'",
		},
		{
			name:          "event to entity - invalid one-to-one",
			sourceType:    "event",
			targetType:    "entity",
			cardinality:   "one-to-one",
			shouldSucceed: false,
			errorContains: "must have cardinality 'many-to-one'",
		},
		{
			name:          "entity to event - valid one-to-many",
			sourceType:    "entity",
			targetType:    "event",
			cardinality:   "one-to-many",
			shouldSucceed: true,
		},
		{
			name:          "entity to event - invalid many-to-one",
			sourceType:    "entity",
			targetType:    "event",
			cardinality:   "many-to-one",
			shouldSucceed: false,
			errorContains: "must have cardinality 'one-to-many'",
		},
		{
			name:          "entity to event - invalid one-to-one",
			sourceType:    "entity",
			targetType:    "event",
			cardinality:   "one-to-one",
			shouldSucceed: false,
			errorContains: "must have cardinality 'one-to-many'",
		},
		{
			name:          "entity to entity - valid one-to-one",
			sourceType:    "entity",
			targetType:    "entity",
			cardinality:   "one-to-one",
			shouldSucceed: true,
		},
		{
			name:          "entity to entity - valid one-to-many",
			sourceType:    "entity",
			targetType:    "entity",
			cardinality:   "one-to-many",
			shouldSucceed: true,
		},
		{
			name:          "entity to entity - valid many-to-one",
			sourceType:    "entity",
			targetType:    "entity",
			cardinality:   "many-to-one",
			shouldSucceed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, graph := buildCardinalityTest(tt.sourceType, tt.targetType, tt.cardinality)

			results := validateRelationshipCardinality("data-graph", specs.SpecVersionV1, nil, spec, graph)

			if tt.shouldSucceed {
				assert.Empty(t, results, "Valid cardinality should not produce errors")
			} else {
				require.Len(t, results, 1)
				assert.Equal(t, "/models/0/relationships/0/cardinality", results[0].Reference)
				assert.Contains(t, results[0].Message, tt.errorContains)
			}
		})
	}
}

