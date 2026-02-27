package source

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceSemanticValidRule_Metadata(t *testing.T) {
	rule := NewSourceSemanticValidRule()

	assert.Equal(t, "event-stream/source/semantic-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "event stream source references must resolve to existing resources", rule.Description())
	assert.Equal(t, []string{"event-stream-source"}, rule.AppliesToKinds())
}

func TestSourceSemanticValid(t *testing.T) {
	t.Parallel()

	t.Run("no governance is valid", func(t *testing.T) {
		t.Parallel()

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "My Source",
			SourceDefinition: "javascript",
		}

		graph := resources.NewGraph()
		results := validateSourceSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("valid tracking plan ref found in graph", func(t *testing.T) {
		t.Parallel()

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "My Source",
			SourceDefinition: "javascript",
			Governance: &esSource.SourceGovernanceSpec{
				TrackingPlan: &esSource.TrackingPlanSpec{
					Ref:    "#/tp/my-group/tp-1",
					Config: &esSource.TrackingPlanConfigSpec{},
				},
			},
		}

		graph := funcs.GraphWith("tp-1", "tracking-plan")
		results := validateSourceSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("tracking plan ref not found in graph", func(t *testing.T) {
		t.Parallel()

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "My Source",
			SourceDefinition: "javascript",
			Governance: &esSource.SourceGovernanceSpec{
				TrackingPlan: &esSource.TrackingPlanSpec{
					Ref:    "#/tp/my-group/tp-missing",
					Config: &esSource.TrackingPlanConfigSpec{},
				},
			},
		}

		graph := resources.NewGraph()
		results := validateSourceSemantic("", "", nil, spec, graph)
		assert.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "not found in the project")
		assert.Equal(t, "/governance/validations/tracking_plan", results[0].Reference)
	})

	t.Run("nil validations inside governance is valid", func(t *testing.T) {
		t.Parallel()

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "My Source",
			SourceDefinition: "javascript",
			Governance:       &esSource.SourceGovernanceSpec{},
		}

		graph := resources.NewGraph()
		results := validateSourceSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})
}

func sourceResource(id, name string) *resources.Resource {
	return resources.NewResource(id, esSource.ResourceType, resources.ResourceData{"name": name}, nil)
}

func TestSourceSemanticValid_NameUniqueness(t *testing.T) {
	t.Parallel()

	t.Run("unique source names", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(sourceResource("src-1", "Source Alpha"))
		graph.AddResource(sourceResource("src-2", "Source Beta"))

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "Source Alpha",
			SourceDefinition: "javascript",
		}

		results := validateSourceSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})

	t.Run("duplicate source name detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(sourceResource("src-1", "Same Name"))
		graph.AddResource(sourceResource("src-2", "Same Name"))

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "Same Name",
			SourceDefinition: "javascript",
		}

		results := validateSourceSemantic("", "", nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/name", results[0].Reference)
		assert.Contains(t, results[0].Message, "duplicate name 'Same Name' within kind 'event-stream-source'")
	})

	t.Run("single source in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(sourceResource("src-1", "Only Source"))

		spec := esSource.SourceSpec{
			LocalID:          "src-1",
			Name:             "Only Source",
			SourceDefinition: "javascript",
		}

		results := validateSourceSemantic("", "", nil, spec, graph)
		assert.Empty(t, results)
	})
}
