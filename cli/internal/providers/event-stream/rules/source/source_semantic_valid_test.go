package source

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestSourceSemanticValidRule_Metadata(t *testing.T) {
	rule := NewSourceSemanticValidRule()

	assert.Equal(t, "event-stream/source/semantic-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "event stream source references must resolve to existing resources", rule.Description())
	assert.Equal(t, []string{"event-stream-source"}, rule.AppliesTo())
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