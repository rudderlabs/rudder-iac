package trackingplan

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

// propertyResourceWithType creates a property resource with a specific type.
func propertyResourceWithType(id, name, typ string) *resources.Resource {
	data := resources.ResourceData{"name": name, "type": typ}
	return resources.NewResource(id, "property", data, nil)
}

func trackingPlanResource(id, name string) *resources.Resource {
	data := resources.ResourceData{"name": name}
	return resources.NewResource(id, "tracking-plan", data, nil)
}

func TestTrackingPlanSemanticValid_ReferenceResolution(t *testing.T) {
	t.Parallel()

	t.Run("all refs found", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith(
			"signup", "event",
			"email", "property",
			"method", "property",
			"user_id", "property",
		)

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email",
									Properties: []localcatalog.PropertyReference{
										{Ref: "#property:email"},
									},
								},
							},
							Default: []localcatalog.PropertyReference{
								{Ref: "#property:user_id"},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)
		assert.Empty(t, results, "all refs exist in graph — no errors expected")
	})

	t.Run("missing property refs", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("signup", "event")

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
						{
							Ref: "#property:address",
							Properties: []*localcatalog.TPRuleProperty{
								{Ref: "#property:zip"},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 3)

		expectedRefs := []string{
			"/rules/0/properties/0/$ref",
			"/rules/0/properties/1/$ref",
			"/rules/0/properties/1/properties/0/$ref",
		}

		actualRefs := make([]string, len(results))
		for i, r := range results {
			actualRefs[i] = r.Reference
		}
		assert.ElementsMatch(t, expectedRefs, actualRefs)

		for _, r := range results {
			assert.Contains(t, r.Message, "not found in resource graph")
		}
	})

	t.Run("missing event ref", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:nonexistent"},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/event/$ref", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced event 'nonexistent' not found")
	})

	t.Run("variant discriminator ref missing", func(t *testing.T) {
		t.Parallel()

		graph := funcs.GraphWith("signup", "event", "email", "property")

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:missing_disc",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case1",
									Properties: []localcatalog.PropertyReference{
										{Ref: "#property:email"},
									},
								},
							},
						},
					},
				},
			},
		}

		results := funcs.ValidateReferences(spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "referenced property 'missing_disc' not found")
	})
}

func TestTrackingPlanSemanticValid_VariantDiscriminator(t *testing.T) {
	t.Parallel()

	t.Run("valid discriminator type — string", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("signup_method", "Signup Method", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
						{Ref: "#property:signup_method"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:signup_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Email",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "string discriminator in own properties — no errors")
	})

	t.Run("valid discriminator type — integer", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("level", "Level", "integer"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
						{Ref: "#property:level"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:level",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Basic",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "integer discriminator in own properties — no errors")
	})

	t.Run("invalid discriminator type — object", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
						{Ref: "#property:address"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:address",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Home",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "discriminator property type 'object' must contain one of: string, integer, boolean")
	})

	t.Run("custom type ref discriminator — allowed", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(resources.NewResource("payment_method", "property", resources.ResourceData{
			"name": "Payment Method",
			"type": resources.PropertyRef{URN: "custom-type:PaymentType", Property: "name"},
		}, nil))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
						{Ref: "#property:payment_method"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:payment_method",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Credit",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "custom type ref discriminator should be allowed")
	})

	t.Run("discriminator not in rule's own properties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("external_prop", "External Prop", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
					},
					Variants: localcatalog.Variants{
						{
							Type:          "discriminator",
							Discriminator: "#property:external_prop",
							Cases: []localcatalog.VariantCase{
								{
									DisplayName: "Case1",
									Properties:  []localcatalog.PropertyReference{{Ref: "#property:email"}},
								},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/variants/0/discriminator", results[0].Reference)
		assert.Contains(t, results[0].Message, "must reference a property defined in the parent's own properties")
	})

	t.Run("rule without variants — no variant errors", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "no variants means no variant errors")
	})
}

func TestTrackingPlanSemanticValid_PropertyNesting(t *testing.T) {
	t.Parallel()

	boolTrue := true

	t.Run("object type allows nesting", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))
		graph.AddResource(propertyResourceWithType("city", "City", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref: "#property:address",
							Properties: []*localcatalog.TPRuleProperty{
								{Ref: "#property:city"},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "object type should allow nesting")
	})

	t.Run("string type does not allow nesting", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))
		graph.AddResource(propertyResourceWithType("domain", "Domain", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref: "#property:email",
							Properties: []*localcatalog.TPRuleProperty{
								{Ref: "#property:domain"},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "nested properties are not allowed for property 'email'")
	})

	t.Run("string type does not allow additionalProperties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref:                  "#property:email",
							AdditionalProperties: &boolTrue,
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/properties/0/additionalProperties", results[0].Reference)
		assert.Contains(t, results[0].Message, "additional_properties is not allowed for property 'email'")
	})

	t.Run("object type allows additionalProperties", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("address", "Address", "object"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref:                  "#property:address",
							AdditionalProperties: &boolTrue,
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "object type should allow additionalProperties")
	})

	t.Run("array type with object item_types allows nesting", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(resources.NewResource("items", "property", resources.ResourceData{
			"name":   "Items",
			"type":   "array",
			"config": map[string]any{"item_types": []any{"object"}},
		}, nil))
		graph.AddResource(propertyResourceWithType("name", "Name", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref: "#property:items",
							Properties: []*localcatalog.TPRuleProperty{
								{Ref: "#property:name"},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "array with object item_types should allow nesting")
	})

	t.Run("array type with string item_types does not allow nesting", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(resources.NewResource("tags", "property", resources.ResourceData{
			"name":   "Tags",
			"type":   "array",
			"config": map[string]any{"item_types": []any{"string"}},
		}, nil))
		graph.AddResource(propertyResourceWithType("label", "Label", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{
							Ref: "#property:tags",
							Properties: []*localcatalog.TPRuleProperty{
								{Ref: "#property:label"},
							},
						},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		require.Len(t, results, 1)
		assert.Equal(t, "/rules/0/properties/0", results[0].Reference)
		assert.Contains(t, results[0].Message, "nested properties are not allowed for property 'tags'")
	})

	t.Run("property without nesting or additionalProperties produces no error", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{}, nil))
		graph.AddResource(propertyResourceWithType("email", "Email", "string"))

		spec := localcatalog.TrackingPlan{
			LocalID: "test_tp",
			Name:    "Test TP",
			Rules: []*localcatalog.TPRule{
				{
					LocalID: "rule1",
					Event:   &localcatalog.TPRuleEvent{Ref: "#event:signup"},
					Properties: []*localcatalog.TPRuleProperty{
						{Ref: "#property:email"},
					},
				},
			},
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results)
	})
}

func TestTrackingPlanSemanticValid_Uniqueness(t *testing.T) {
	t.Parallel()

	t.Run("unique tracking plans", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(trackingPlanResource("tp1", "Onboarding Plan"))
		graph.AddResource(trackingPlanResource("tp2", "Checkout Plan"))

		spec := localcatalog.TrackingPlan{
			LocalID: "tp1",
			Name:    "Onboarding Plan",
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "unique names — no errors expected")
	})

	t.Run("duplicate detected", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(trackingPlanResource("tp1", "Onboarding Plan"))
		graph.AddResource(trackingPlanResource("tp2", "Onboarding Plan"))

		spec := localcatalog.TrackingPlan{
			LocalID: "tp1",
			Name:    "Onboarding Plan",
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)

		require.Len(t, results, 1)
		assert.Equal(t, "/display_name", results[0].Reference)
		assert.Contains(t, results[0].Message, "tracking plan with name 'Onboarding Plan' is not unique")
	})

	t.Run("single in graph — no false positive", func(t *testing.T) {
		t.Parallel()

		graph := resources.NewGraph()
		graph.AddResource(trackingPlanResource("tp1", "Onboarding Plan"))

		spec := localcatalog.TrackingPlan{
			LocalID: "tp1",
			Name:    "Onboarding Plan",
		}

		results := validateTrackingPlanSemantic(localcatalog.KindTrackingPlans, specs.SpecVersionV0_1, nil, spec, graph)
		assert.Empty(t, results, "single entry should not trigger uniqueness error")
	})
}
