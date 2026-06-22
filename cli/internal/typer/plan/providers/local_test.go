package providers_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCatalog assembles a DataCatalog covering the constructs the v1 renderer
// supports: primitives, enum config, arrays, nested objects, and a custom type.
func buildCatalog() *localcatalog.DataCatalog {
	dc := localcatalog.New()
	dc.Properties = []localcatalog.PropertyV1{
		{LocalID: "str", Name: "strProp", Type: "string", Description: "a string"},
		{LocalID: "enm", Name: "enumProp", Type: "string", Config: map[string]any{"enum": []any{"a", "b"}}},
		{LocalID: "arr", Name: "arrProp", Type: "array", ItemType: "string"},
		{LocalID: "obj", Name: "objProp", Type: "object"},
		{LocalID: "child", Name: "childProp", Type: "integer"},
		{LocalID: "cust", Name: "custProp", Type: "#custom-type:email"},
	}
	dc.CustomTypes = []localcatalog.CustomTypeV1{
		{LocalID: "email", Name: "Email", Type: "string", Config: map[string]any{"enum": []any{"x"}}},
	}
	dc.Events = []localcatalog.EventV1{
		{LocalID: "evt", Name: "My Event", Type: "track", Description: "an event"},
	}
	dc.TrackingPlans = []*localcatalog.TrackingPlanV1{
		{
			LocalID: "tp",
			Name:    "TP",
			Rules: []*localcatalog.TPRuleV1{
				{
					Type:            "event_rule",
					LocalID:         "r1",
					Event:           "#event:evt",
					IdentitySection: "properties",
					Properties: []*localcatalog.TPRulePropertyV1{
						{Property: "#property:str", Required: true},
						{Property: "#property:enm"},
						{Property: "#property:arr"},
						{Property: "#property:cust"},
						{Property: "#property:obj", Properties: []*localcatalog.TPRulePropertyV1{
							{Property: "#property:child", Required: true},
						}},
					},
				},
			},
		},
	}
	return dc
}

func TestLocalCatalogPlanProvider_GetTrackingPlan(t *testing.T) {
	p := providers.NewLocalCatalogPlanProvider(buildCatalog(), "tp")

	got, err := p.GetTrackingPlan(context.Background())
	require.NoError(t, err)

	want := &plan.TrackingPlan{
		Name: "TP",
		Metadata: plan.PlanMetadata{
			TrackingPlanID: "tp",
			URL:            "https://app.rudderstack.com/trackingPlans/tp",
		},
		Rules: []plan.EventRule{
			{
				Event:   plan.Event{Name: "My Event", EventType: plan.EventTypeTrack, Description: "an event"},
				Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{
					AdditionalProperties: false,
					Properties: map[string]plan.PropertySchema{
						"strProp": {
							Property: plan.Property{Name: "strProp", Types: []plan.PropertyType{plan.PrimitiveTypeString}},
							Required: true,
						},
						"enumProp": {
							Property: plan.Property{
								Name:   "enumProp",
								Types:  []plan.PropertyType{plan.PrimitiveTypeString},
								Config: &plan.PropertyConfig{Enum: []any{"a", "b"}},
							},
						},
						"arrProp": {
							Property: plan.Property{
								Name:      "arrProp",
								Types:     []plan.PropertyType{plan.PrimitiveTypeArray},
								ItemTypes: []plan.PropertyType{plan.PrimitiveTypeString},
							},
						},
						"custProp": {
							Property: plan.Property{
								Name: "custProp",
								Types: []plan.PropertyType{&plan.CustomType{
									Name:   "Email",
									Type:   plan.PrimitiveTypeString,
									Config: &plan.PropertyConfig{Enum: []any{"x"}},
								}},
							},
						},
						"objProp": {
							Property: plan.Property{Name: "objProp", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
							Schema: &plan.ObjectSchema{
								AdditionalProperties: true,
								Properties: map[string]plan.PropertySchema{
									"childProp": {
										Property: plan.Property{Name: "childProp", Types: []plan.PropertyType{plan.PrimitiveTypeInteger}},
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, want, got)
}

func TestLocalCatalogPlanProvider_FailsLoudOnVariants(t *testing.T) {
	dc := buildCatalog()
	dc.TrackingPlans[0].Rules[0].Variants = localcatalog.VariantsV1{
		{Type: "discriminator", Discriminator: "#property:str"},
	}

	_, err := providers.NewLocalCatalogPlanProvider(dc, "tp").GetTrackingPlan(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "variants")
}

func TestLocalCatalogPlanProvider_TrackingPlanNotFound(t *testing.T) {
	_, err := providers.NewLocalCatalogPlanProvider(buildCatalog(), "missing").GetTrackingPlan(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
