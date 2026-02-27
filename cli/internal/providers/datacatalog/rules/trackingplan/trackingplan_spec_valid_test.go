package trackingplan

import (
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"

	// Trigger pattern registration (legacy_event_ref, legacy_property_ref, display_name, etc.) from parent rules package
	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestTrackingPlanSpecSyntaxValidRule_Metadata(t *testing.T) {
	rule := NewTrackingPlanSpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/tracking-plans/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "tracking plan spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"tp"}, rule.AppliesToKinds())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}

func TestTrackingPlanSpecSyntaxValidRule_ValidEventAndPropertyRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.TrackingPlan
	}{
		{
			name: "rule with valid event ref",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
					},
				},
			},
		},
		{
			name: "rule with valid event ref and property refs",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: "#/properties/signup-props/email"},
							{Ref: "#/properties/signup-props/name", Required: true},
						},
					},
				},
			},
		},
		{
			name: "rule with nested property refs",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{
								Ref:      "#/properties/signup-props/address",
								Required: true,
								Properties: []*localcatalog.TPRuleProperty{
									{Ref: "#/properties/address-props/city"},
									{Ref: "#/properties/address-props/zip", Required: true},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "rule with identity_section properties",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref:             "#/events/user-events/signup",
							IdentitySection: "properties",
						},
					},
				},
			},
		},
		{
			name: "rule with identity_section traits",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref:             "#/events/user-events/signup",
							IdentitySection: "traits",
						},
					},
				},
			},
		},
		{
			name: "rule with identity_section context.traits",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref:             "#/events/user-events/signup",
							IdentitySection: "context.traits",
						},
					},
				},
			},
		},
		{
			name: "rule with empty identity_section",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
					},
				},
			},
		},
		{
			name: "rule with event ref and nil properties",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
					},
				},
			},
		},
		{
			name: "rule with event ref and empty properties",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/page_view",
						},
						Properties: []*localcatalog.TPRuleProperty{},
					},
				},
			},
		},
		{
			name: "multiple rules with event and property refs",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: "#/properties/signup-props/email"},
						},
					},
					{
						LocalID: "rule2",
						Event: &localcatalog.TPRuleEvent{
							Ref:            "#/events/user-events/login",
							AllowUnplanned: true,
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: "#/properties/login-props/method", Required: true},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, tt.spec)
			assert.Empty(t, results, "Valid spec should produce no errors")
		})
	}
}

func TestTrackingPlanSpecSyntaxValidRule_InvalidEventAndPropertyRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         localcatalog.TrackingPlan
		expectedRefs []string
		expectedMsgs []string
	}{
		{
			name: "rule with invalid identity_section",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref:             "#/events/user-events/signup",
							IdentitySection: "invalid_section",
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/event/identity_section"},
			expectedMsgs: []string{"'identity_section' must be one of [properties traits context.traits]"},
		},
		{
			name: "rule local id missing",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/id"},
			expectedMsgs: []string{"'id' is required"},
		},
		{
			name: "rule event missing",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
					},
				},
			},
			expectedRefs: []string{"/rules/0/event"},
			expectedMsgs: []string{"'event' is required"},
		},
		{
			name: "event ref missing",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "",
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/event/$ref"},
			expectedMsgs: []string{"'$ref' is required"},
		},
		{
			name: "event ref invalid format",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "not_a_valid_ref",
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/event/$ref"},
			expectedMsgs: []string{"'$ref' is not valid: must be of pattern #/events/<group>/<id>"},
		},
		{
			name: "property ref missing",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: ""},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is required"},
		},
		{
			name: "property ref invalid format",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: "invalid_property_ref"},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
		{
			name: "nested property ref invalid format",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{
								Ref: "#/properties/signup-props/address",
								Properties: []*localcatalog.TPRuleProperty{
									{Ref: "bad_nested_ref"},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/properties/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is not valid: must be of pattern #/properties/<group>/<id>"},
		},
		{
			name: "multiple property ref errors across rules",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Event: &localcatalog.TPRuleEvent{
							Ref: "#/events/user-events/signup",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: ""},
							{Ref: "#/properties/signup-props/email"},
						},
					},
					{
						LocalID: "rule2",
						Event: &localcatalog.TPRuleEvent{
							Ref: "bad_event_ref",
						},
						Properties: []*localcatalog.TPRuleProperty{
							{Ref: "bad_prop_ref"},
						},
					},
				},
			},
			expectedRefs: []string{
				"/rules/0/properties/0/$ref",
				"/rules/1/event/$ref",
				"/rules/1/properties/0/$ref",
			},
			expectedMsgs: []string{
				"'$ref' is required",
				"'$ref' is not valid: must be of pattern #/events/<group>/<id>",
				"'$ref' is not valid: must be of pattern #/properties/<group>/<id>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, tt.spec)

			assert.Len(t, results, len(tt.expectedRefs), "Expected %d errors, got %d", len(tt.expectedRefs), len(results))

			actualRefs := extractRefs(results)
			actualMsgs := extractMsgs(results)

			assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "References don't match")
			assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Messages don't match")
		})
	}
}

func TestTrackingPlanSpecSyntaxValidRule_ValidNameAndDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec localcatalog.TrackingPlan
	}{
		{
			name: "name with minimum length (3 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "Abc",
			},
		},
		{
			name: "name with maximum length (65 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "A" + strings.Repeat("b", 64),
			},
		},
		{
			name: "name starting with lowercase letter",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "user tracking plan",
			},
		},
		{
			name: "name starting with underscore",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "_internal plan",
			},
		},
		{
			name: "name with all allowed special chars",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "Plan v2.0, beta - test",
			},
		},
		{
			name: "description with minimum length (3 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID:     "tp1",
				Name:        "Test Plan",
				Description: "abc",
			},
		},
		{
			name: "description omitted is valid",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "Test Plan",
			},
		},
		{
			name: "empty description is valid (omitempty)",
			spec: localcatalog.TrackingPlan{
				LocalID:     "tp1",
				Name:        "Test Plan",
				Description: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, tt.spec)
			assert.Empty(t, results, "Valid spec should produce no errors")
		})
	}
}

func TestTrackingPlanSpecSyntaxValidRule_InvalidNameAndDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         localcatalog.TrackingPlan
		expectedRefs []string
		expectedMsgs []string
	}{
		{
			name: "id missing",
			spec: localcatalog.TrackingPlan{
				Name: "Test Plan",
			},
			expectedRefs: []string{"/id"},
			expectedMsgs: []string{"'id' is required"},
		},
		{
			name: "name missing",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
			},
			expectedRefs: []string{"/display_name"},
			expectedMsgs: []string{"'display_name' is required"},
		},
		{
			name: "name too short (2 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "Ab",
			},
			expectedRefs: []string{"/display_name"},
			expectedMsgs: []string{"'display_name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name too long (66 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "A" + strings.Repeat("b", 65),
			},
			expectedRefs: []string{"/display_name"},
			expectedMsgs: []string{"'display_name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name starting with digit",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "1Invalid Plan",
			},
			expectedRefs: []string{"/display_name"},
			expectedMsgs: []string{"'display_name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "name with disallowed chars",
			spec: localcatalog.TrackingPlan{
				LocalID: "tp1",
				Name:    "Plan@Name",
			},
			expectedRefs: []string{"/display_name"},
			expectedMsgs: []string{"'display_name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"},
		},
		{
			name: "description too short (2 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID:     "tp1",
				Name:        "Test Plan",
				Description: "ab",
			},
			expectedRefs: []string{"/description"},
			expectedMsgs: []string{"'description' length must be greater than or equal to 3"},
		},
		{
			name: "description too long (2001 chars)",
			spec: localcatalog.TrackingPlan{
				LocalID:     "tp1",
				Name:        "Test Plan",
				Description: "a" + strings.Repeat("b", 2000),
			},
			expectedRefs: []string{"/description"},
			expectedMsgs: []string{"'description' length must be less than or equal to 2000"},
		},
		{
			name: "description not starting with letter",
			spec: localcatalog.TrackingPlan{
				LocalID:     "tp1",
				Name:        "Test Plan",
				Description: "123 not starting with letter",
			},
			expectedRefs: []string{"/description"},
			expectedMsgs: []string{"'description' is not valid: must start with a letter [a-zA-Z]"},
		},
		{
			name: "both name and description invalid",
			spec: localcatalog.TrackingPlan{
				LocalID:     "tp1",
				Name:        "#bad",
				Description: "1x",
			},
			expectedRefs: []string{"/display_name", "/description"},
			expectedMsgs: []string{
				"'display_name' is not valid: must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters",
				"'description' length must be greater than or equal to 3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, tt.spec)

			assert.Len(t, results, len(tt.expectedRefs), "Expected %d errors, got %d", len(tt.expectedRefs), len(results))

			actualRefs := extractRefs(results)
			actualMsgs := extractMsgs(results)

			assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "References don't match")
			assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Messages don't match")
		})
	}
}

func TestTrackingPlanSpecSyntaxValidRule_VariantReferencePaths(t *testing.T) {
	t.Parallel()

	t.Run("nil variants is valid", func(t *testing.T) {
		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event: &localcatalog.TPRuleEvent{
						Ref: "#/events/user-events/signup",
					},
					Variants: nil,
				},
			},
		}

		results := validateTrackingPlanSpec(
			localcatalog.KindTrackingPlans,
			specs.SpecVersionV0_1,
			map[string]any{},
			spec,
		)
		assert.Empty(t, results, "Nil variants should produce no errors")
	})
	t.Run("empty variants is valid", func(t *testing.T) {
		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event: &localcatalog.TPRuleEvent{
						Ref: "#/events/user-events/signup",
					},
					Variants: localcatalog.Variants{},
				},
			},
		}

		results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.Empty(t, results, "Empty variants should produce no errors")
	})

	t.Run("valid variant produces no errors", func(t *testing.T) {
		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event: &localcatalog.TPRuleEvent{
						Ref: "#/events/user-events/signup",
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#/properties/signup-props/method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email Signup",
									Match:       []any{"email"},
									Properties: []localcatalog.PropertyReference{
										{Ref: "#/properties/signup-props/email", Required: true},
									},
								},
							},
							Default: []localcatalog.PropertyReference{
								{Ref: "#/properties/common/user_id", Required: true},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.Empty(t, results, "Valid variant should produce no errors")
	})

	t.Run("more than one variant is invalid", func(t *testing.T) {
		validVariant := localcatalog.Variant{
			Type:          "discriminator",
			Discriminator: "#/properties/signup-props/method",
			Cases: []localcatalog.VariantCase{
				{
					DisplayName: "Case1",
					Match:       []any{"value"},
					Properties: []localcatalog.PropertyReference{
						{Ref: "#/properties/signup-props/email", Required: true},
					},
				},
			},
		}

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID:  "rule1",
					Event:    &localcatalog.TPRuleEvent{Ref: "#/events/user-events/signup"},
					Variants: localcatalog.Variants{validVariant, validVariant},
				},
			},
		}

		results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.NotEmpty(t, results)
		assert.Contains(t, extractRefs(results), "/rules/0/variants")
		assert.Contains(t, extractMsgs(results), "'variants' length must be less than or equal to 1")
	})

	t.Run("invalid variant generates correct reference paths", func(t *testing.T) {
		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event: &localcatalog.TPRuleEvent{
						Ref: "#/events/user-events/signup",
					},
					Variants: localcatalog.Variants{
						{
							Type:          "wrong_type",
							Discriminator: "bad_ref",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case1",
									Match:       []any{"value"},
									Properties: []localcatalog.PropertyReference{
										{Ref: "bad_prop_ref"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, spec)

		expectedRefs := []string{
			"/rules/0/variants/0/type",
			"/rules/0/variants/0/discriminator",
			"/rules/0/variants/0/cases/0/properties/0/$ref",
		}
		expectedMsgs := []string{
			"'type' must equal 'discriminator'",
			"'discriminator' is not valid: must be of pattern #/properties/<group>/<id>",
			"'$ref' is not valid: must be of pattern #/properties/<group>/<id>",
		}

		assert.Len(t, results, 3)
		assert.ElementsMatch(t, expectedRefs, extractRefs(results))
		assert.ElementsMatch(t, expectedMsgs, extractMsgs(results))
	})
}

func TestTrackingPlanSpecSyntaxValidRule_NestingDepth(t *testing.T) {
	t.Parallel()

	t.Run("3 levels of nesting is valid", func(t *testing.T) {
		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#/events/user-events/signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref: "#/properties/props/address",
							Properties: []*localcatalog.TPRuleProperty{
								{
									Ref: "#/properties/props/city",
									Properties: []*localcatalog.TPRuleProperty{
										{
											Ref: "#/properties/props/district",
											Properties: []*localcatalog.TPRuleProperty{
												{Ref: "#/properties/props/zone"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.Empty(t, results, "3 levels of nesting should produce no errors")
	})

	t.Run("4 levels of nesting exceeds max depth", func(t *testing.T) {
		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#/events/user-events/signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref: "#/properties/props/address",
							Properties: []*localcatalog.TPRuleProperty{
								{
									Ref: "#/properties/props/city",
									Properties: []*localcatalog.TPRuleProperty{
										{
											Ref: "#/properties/props/district",
											Properties: []*localcatalog.TPRuleProperty{
												{
													Ref: "#/properties/props/zone",
													Properties: []*localcatalog.TPRuleProperty{
														{Ref: "#/properties/props/block"},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSpec(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, map[string]any{}, spec)
		assert.Len(t, results, 1)
		assert.Equal(t, "/rules/0/properties/0", extractRefs(results)[0])
		assert.Equal(t, "maximum property nesting depth of 3 levels exceeded", extractMsgs(results)[0])
	})

}

// Helper functions to extract references and messages from validation results
func extractRefs(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, r := range results {
		refs[i] = r.Reference
	}
	return refs
}

func extractMsgs(results []rules.ValidationResult) []string {
	msgs := make([]string, len(results))
	for i, r := range results {
		msgs[i] = r.Message
	}
	return msgs
}
