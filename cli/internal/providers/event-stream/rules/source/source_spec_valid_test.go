package source

import (
	"testing"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	esSource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// trigger pattern registration
	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestSourceSpecSyntaxValidRule_Metadata(t *testing.T) {
	rule := NewSourceSpecSyntaxValidRule()

	assert.Equal(t, "event-stream/source/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "event stream source spec syntax must be valid", rule.Description())
	assert.Equal(t, prules.LegacyVersionPatterns("event-stream-source"), rule.AppliesTo())
}

func TestSourceSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec esSource.SourceSpec
	}{
		{
			name: "minimal source",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: "javascript",
			},
		},
		{
			name: "source with governance",
			spec: esSource.SourceSpec{
				LocalID:          "src-2",
				Name:             "My Source",
				SourceDefinition: "node",
				Governance: &esSource.SourceGovernanceSpec{
					TrackingPlan: &esSource.TrackingPlanSpec{
						Ref:    "#/tp/my-group/tp-1",
						Config: &esSource.TrackingPlanConfigSpec{},
					},
				},
			},
		},
		{
			name: "nil governance",
			spec: esSource.SourceSpec{
				LocalID:          "src-3",
				Name:             "My Source",
				SourceDefinition: "python",
				Governance:       nil,
			},
		},
		{
			name: "empty governance with nil validations",
			spec: esSource.SourceSpec{
				LocalID:          "src-4",
				Name:             "My Source",
				SourceDefinition: "go",
				Governance:       &esSource.SourceGovernanceSpec{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := validateSourceSpec("", "", nil, tt.spec)
			assert.Empty(t, results, "expected no validation errors")
		})
	}
}

func TestSourceSpecSyntaxValidRule_AllSourceTypes(t *testing.T) {
	t.Parallel()

	sourceTypes := []string{
		"java", "dotnet", "php", "flutter", "cordova", "rust",
		"react_native", "python", "ios", "android", "javascript",
		"go", "node", "ruby", "unity",
	}

	for _, st := range sourceTypes {
		t.Run(st, func(t *testing.T) {
			t.Parallel()
			spec := esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: st,
			}
			results := validateSourceSpec("", "", nil, spec)
			assert.Empty(t, results, "source type %q should be valid", st)
		})
	}
}

func TestSourceSpecSyntaxValidRule_InvalidSpecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         esSource.SourceSpec
		wantMessages []string
	}{
		{
			name: "missing id",
			spec: esSource.SourceSpec{
				Name:             "My Source",
				SourceDefinition: "javascript",
			},
			wantMessages: []string{"'id' is required"},
		},
		{
			name: "missing name",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				SourceDefinition: "javascript",
			},
			wantMessages: []string{"'name' is required"},
		},
		{
			name: "missing type",
			spec: esSource.SourceSpec{
				LocalID: "src-1",
				Name:    "My Source",
			},
			wantMessages: []string{"'type' is required"},
		},
		{
			name: "invalid type enum",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: "invalid_type",
			},
			wantMessages: []string{
				"'type' must be one of [java dotnet php flutter cordova rust react_native python ios android javascript go node ruby unity]",
			},
		},
		{
			name: "invalid tracking plan ref format",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: "javascript",
				Governance: &esSource.SourceGovernanceSpec{
					TrackingPlan: &esSource.TrackingPlanSpec{
						Ref:    "not-a-valid-ref",
						Config: &esSource.TrackingPlanConfigSpec{},
					},
				},
			},
			wantMessages: []string{
				"'tracking_plan' is not valid: must be of pattern #/tp/<group>/<id>",
			},
		},
		{
			name: "new format tracking plan ref rejected",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: "javascript",
				Governance: &esSource.SourceGovernanceSpec{
					TrackingPlan: &esSource.TrackingPlanSpec{
						Ref:    "#tracking-plan:tp-1",
						Config: &esSource.TrackingPlanConfigSpec{},
					},
				},
			},
			wantMessages: []string{
				"'tracking_plan' is not valid: must be of pattern #/tp/<group>/<id>",
			},
		},
		{
			name: "missing config when validations present",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: "javascript",
				Governance: &esSource.SourceGovernanceSpec{
					TrackingPlan: &esSource.TrackingPlanSpec{
						Ref: "#/tp/my-group/tp-1",
					},
				},
			},
			wantMessages: []string{"'config' is required"},
		},
		{
			name: "missing tracking_plan when validations present",
			spec: esSource.SourceSpec{
				LocalID:          "src-1",
				Name:             "My Source",
				SourceDefinition: "javascript",
				Governance: &esSource.SourceGovernanceSpec{
					TrackingPlan: &esSource.TrackingPlanSpec{
						Config: &esSource.TrackingPlanConfigSpec{},
					},
				},
			},
			wantMessages: []string{"'tracking_plan' is required"},
		},
		{
			name: "all required fields missing",
			spec: esSource.SourceSpec{},
			wantMessages: []string{
				"'id' is required",
				"'name' is required",
				"'type' is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := validateSourceSpec("", "", nil, tt.spec)
			require.Len(t, results, len(tt.wantMessages))

			var gotMessages []string
			for _, r := range results {
				gotMessages = append(gotMessages, r.Message)
			}
			assert.ElementsMatch(t, tt.wantMessages, gotMessages)
		})
	}
}
