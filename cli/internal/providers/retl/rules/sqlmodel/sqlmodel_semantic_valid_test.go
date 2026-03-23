package sqlmodel

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sqlModelResource(id, displayName string) *resources.Resource {
	return resources.NewResource(id, sqlmodel.ResourceType, resources.ResourceData{
		sqlmodel.DisplayNameKey: displayName,
	}, nil)
}

func TestSQLModelSemanticValidRule_Metadata(t *testing.T) {
	rule := NewSQLModelSemanticValidRule()

	expectedPatterns := append(
		prules.LegacyVersionPatterns("retl-source-sql-model"),
		rules.MatchKindVersion(sqlmodel.ResourceKind, specs.SpecVersionV1),
	)

	assert.Equal(t, "retl/sqlmodel/semantic-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "retl sql model semantic constraints must be satisfied", rule.Description())
	assert.Equal(t, expectedPatterns, rule.AppliesTo())
}

func TestSQLModelSemanticValid_DisplayNameUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("unique display names", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(sqlModelResource("model-1", "Model Alpha"))
		graph.AddResource(sqlModelResource("model-2", "Model Beta"))

		spec := sqlmodel.SQLModelSpec{
			ID:          "model-1",
			DisplayName: "Model Alpha",
		}

		results := validateSQLModelSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("duplicate display name detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(sqlModelResource("model-1", "Same Name"))
		graph.AddResource(sqlModelResource("model-2", "Same Name"))

		spec := sqlmodel.SQLModelSpec{
			ID:          "model-1",
			DisplayName: "Same Name",
		}

		results := validateSQLModelSemantic("", "", nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/display_name", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate display_name 'Same Name' within kind 'retl-source-sql-model'")
	})

	t.Run("single resource — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(sqlModelResource("model-1", "Only Model"))

		spec := sqlmodel.SQLModelSpec{
			ID:          "model-1",
			DisplayName: "Only Model",
		}

		results := validateSQLModelSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})
}
