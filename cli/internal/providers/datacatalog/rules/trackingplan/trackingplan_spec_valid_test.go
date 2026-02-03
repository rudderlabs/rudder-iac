package trackingplan

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestTrackingPlanSpecSyntaxValidRule_Metadata(t *testing.T) {
	rule := NewTrackingPlanSpecSyntaxValidRule()

	assert.Equal(t, "datacatalog/tracking-plans/spec-syntax-valid", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "tracking plan spec syntax must be valid", rule.Description())
	assert.Equal(t, []string{"tp"}, rule.AppliesTo())

	examples := rule.Examples()
	assert.NotEmpty(t, examples.Valid)
	assert.NotEmpty(t, examples.Invalid)
}

func TestTrackingPlanSpecSyntaxValidRule_ValidSpecs(t *testing.T) {
	tests := []struct {
		name string
		spec localcatalog.TrackingPlan
	}{
		{
			name: "single variant with one case",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/page_type",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Search",
										Description: "Search page",
										Properties: []localcatalog.PropertyReference{
											{Ref: "#/properties/group/search_term", Required: true},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "variant with multiple cases and default",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule2",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/event_category",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Ecommerce",
										Description: "E-commerce events",
										Properties: []localcatalog.PropertyReference{
											{Ref: "#/properties/group/product_id"},
											{Ref: "#/properties/group/price", Required: true},
										},
									},
									{
										DisplayName: "Content",
										Properties: []localcatalog.PropertyReference{
											{Ref: "#/properties/group/article_id"},
										},
									},
								},
								Default: []localcatalog.PropertyReference{
									{Ref: "#/properties/group/user_id", Required: true},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "no variants (optional)",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID:  "rule3",
						Variants: nil,
					},
				},
			},
		},
		{
			name: "empty variants array (optional)",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID:  "rule4",
						Variants: localcatalog.Variants{},
					},
				},
			},
		},
		{
			name: "multiple rules with variants",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule5",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/status",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Active",
										Properties: []localcatalog.PropertyReference{
											{Ref: "#/properties/group/active_field"},
										},
									},
								},
							},
						},
					},
					{
						LocalID: "rule6",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/type",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Type A",
										Properties: []localcatalog.PropertyReference{
											{Ref: "#/properties/group/field_a"},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateTrackingPlanSpec("tp", "rudder/v1", map[string]any{}, tt.spec)
			assert.Empty(t, results, "Valid spec should produce no errors")
		})
	}
}

func TestTrackingPlanSpecSyntaxValidRule_InvalidVariants(t *testing.T) {
	tests := []struct {
		name         string
		spec         localcatalog.TrackingPlan
		expectedRefs []string
		expectedMsgs []string
	}{
		{
			name: "type not discriminator",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "invalid_type",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/type"},
			expectedMsgs: []string{"'type' must equal 'discriminator'"},
		},
		{
			name: "discriminator empty",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/discriminator"},
			expectedMsgs: []string{"'discriminator' is required"},
		},
		{
			name: "discriminator invalid format",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "not_a_reference",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/discriminator"},
			expectedMsgs: []string{"'discriminator' is not a valid reference format"},
		},
		{
			name: "cases array empty",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases:         []localcatalog.VariantCase{},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/cases"},
			expectedMsgs: []string{"'cases' length must be greater than or equal to 1"},
		},
		{
			name: "case missing display_name",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/cases/0/display_name"},
			expectedMsgs: []string{"'display_name' is required"},
		},
		{
			name: "case properties empty",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/cases/0/properties"},
			expectedMsgs: []string{"'properties' length must be greater than or equal to 1"},
		},
		{
			name: "property reference missing $ref",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: ""}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/cases/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is required"},
		},
		{
			name: "property reference invalid format",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "invalid_ref"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/cases/0/properties/0/$ref"},
			expectedMsgs: []string{"'$ref' is not a valid reference format"},
		},
		{
			name: "default property reference missing $ref",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
								Default: []localcatalog.PropertyReference{{Ref: ""}},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/default/0/$ref"},
			expectedMsgs: []string{"'$ref' is required"},
		},
		{
			name: "default property reference invalid format",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
								Default: []localcatalog.PropertyReference{{Ref: "invalid"}},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants/0/default/0/$ref"},
			expectedMsgs: []string{"'$ref' is not a valid reference format"},
		},
		{
			name: "more than one variant",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field1",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field2",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case2",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p2"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{"/rules/0/variants"},
			expectedMsgs: []string{"'variants' length must be less than or equal to 1"},
		},
		{
			name: "multiple errors in same variant",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "wrong",
								Discriminator: "",
								Cases:         []localcatalog.VariantCase{},
							},
						},
					},
				},
			},
			expectedRefs: []string{
				"/rules/0/variants/0/type",
				"/rules/0/variants/0/discriminator",
				"/rules/0/variants/0/cases",
			},
			expectedMsgs: []string{
				"'type' must equal 'discriminator'",
				"'discriminator' is required",
				"'cases' length must be greater than or equal to 1",
			},
		},
		{
			name: "multiple rules with variant errors",
			spec: localcatalog.TrackingPlan{
				LocalID: "test_tp",
				Name:    "Test TP",
				Rules: []*localcatalog.TPRule{
					{
						LocalID: "rule1",
						Variants: localcatalog.Variants{
							{
								Type:          "wrong",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "Case1",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
						},
					},
					{
						LocalID: "rule2",
						Variants: localcatalog.Variants{
							{
								Type:          "discriminator",
								Discriminator: "#/properties/group/field",
								Cases: []localcatalog.VariantCase{
									{
										DisplayName: "",
										Properties:  []localcatalog.PropertyReference{{Ref: "#/properties/group/p1"}},
									},
								},
							},
						},
					},
				},
			},
			expectedRefs: []string{
				"/rules/0/variants/0/type",
				"/rules/1/variants/0/cases/0/display_name",
			},
			expectedMsgs: []string{
				"'type' must equal 'discriminator'",
				"'display_name' is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := validateTrackingPlanSpec("tp", "rudder/v1", map[string]any{}, tt.spec)

			assert.Len(t, results, len(tt.expectedRefs), "Expected %d errors, got %d", len(tt.expectedRefs), len(results))

			actualRefs := extractRefs(results)
			actualMsgs := extractMsgs(results)

			assert.ElementsMatch(t, tt.expectedRefs, actualRefs, "References don't match")
			assert.ElementsMatch(t, tt.expectedMsgs, actualMsgs, "Messages don't match")
		})
	}
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
