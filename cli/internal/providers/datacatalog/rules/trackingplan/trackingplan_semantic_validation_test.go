package trackingplan

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"

	_ "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

func TestTrackingPlanSemanticValid_AllRefsFound(t *testing.T) {
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
}

func TestTrackingPlanSemanticValid_MissingRefs(t *testing.T) {
	t.Parallel()

	// graph has only "signup" event — all property refs will fail
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

	var (
		actualRefs = make([]string, len(results))
		actualMsgs = make([]string, len(results))
	)
	for i, r := range results {
		actualRefs[i] = r.Reference
		actualMsgs[i] = r.Message
	}

	expectedRefs := []string{
		"/rules/0/properties/0/$ref",
		"/rules/0/properties/1/$ref",
		"/rules/0/properties/1/properties/0/$ref",
	}

	assert.Len(t, results, 3)
	assert.ElementsMatch(t, expectedRefs, actualRefs)
	for _, msg := range actualMsgs {
		assert.Contains(t, msg, "not found in resource graph")
	}
}

func TestTrackingPlanSemanticValid_MissingEventRef(t *testing.T) {
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

	assert.Len(t, results, 1)
	assert.Equal(t, "/rules/0/event/$ref", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced event 'nonexistent' not found")
}

func TestTrackingPlanSemanticValid_VariantDiscriminatorMissing(t *testing.T) {
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

	assert.Len(t, results, 1)
	assert.Equal(t, "/rules/0/variants/0/discriminator", results[0].Reference)
	assert.Contains(t, results[0].Message, "referenced property 'missing_disc' not found")
}
