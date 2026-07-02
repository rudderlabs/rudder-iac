package providers_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildCatalog assembles a DataCatalog covering primitives, enum config, arrays,
// nested objects, and a custom type.
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
		Name:     "TP",
		Metadata: plan.PlanMetadata{TrackingPlanID: "tp"},
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

func TestLocalCatalogPlanProvider_Variants(t *testing.T) {
	dc := buildCatalog()
	dc.TrackingPlans[0].Rules[0].Variants = localcatalog.VariantsV1{
		{
			Type:          "discriminator",
			Discriminator: "#property:str", // resolves to the property name "strProp"
			Cases: []localcatalog.VariantCaseV1{
				{
					DisplayName: "Case 1",
					Description: "first case",
					Match:       []any{"a"},
					Properties:  []localcatalog.PropertyReferenceV1{{Property: "#property:child", Required: true}},
				},
			},
			Default: localcatalog.DefaultPropertiesV1{
				Properties: []localcatalog.PropertyReferenceV1{{Property: "#property:enm"}},
			},
		},
	}

	got, err := providers.NewLocalCatalogPlanProvider(dc, "tp").GetTrackingPlan(context.Background())
	require.NoError(t, err)

	want := []plan.Variant{
		{
			Type:          "discriminator",
			Discriminator: "strProp",
			Cases: []plan.VariantCase{
				{
					DisplayName: "Case 1",
					Description: "first case",
					Match:       []any{"a"},
					Schema: plan.ObjectSchema{
						Properties: map[string]plan.PropertySchema{
							"childProp": {
								Property: plan.Property{Name: "childProp", Types: []plan.PropertyType{plan.PrimitiveTypeInteger}},
								Required: true,
							},
						},
					},
				},
			},
			DefaultSchema: &plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"enumProp": {
						Property: plan.Property{
							Name:   "enumProp",
							Types:  []plan.PropertyType{plan.PrimitiveTypeString},
							Config: &plan.PropertyConfig{Enum: []any{"a", "b"}},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, want, got.Rules[0].Variants)
}

func TestLocalCatalogPlanProvider_TrackingPlanNotFound(t *testing.T) {
	_, err := providers.NewLocalCatalogPlanProvider(buildCatalog(), "missing").GetTrackingPlan(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestLocalAndRemoteProduceEquivalentPlan is the guard that justifies having two
// independent adapters onto plan.TrackingPlan: for the same logical plan, the
// local converter and the remote JSON-Schema parser must produce the same rules.
// (Metadata differs by design — the local plan has no remote URL/version.)
func TestLocalAndRemoteProduceEquivalentPlan(t *testing.T) {
	ctx := context.Background()

	localPlan, err := providers.NewLocalCatalogPlanProvider(buildCatalog(), "tp").GetTrackingPlan(ctx)
	require.NoError(t, err)

	// Hand-authored JSON-Schema representation of the same plan buildCatalog builds.
	ev := catalog.TrackingPlanEventSchema{
		Name:            "My Event",
		Description:     "an event",
		EventType:       "track",
		IdentitySection: "properties",
	}
	ev.Rules.Type = "object"
	ev.Rules.Properties = map[string]any{
		"properties": map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"strProp":  map[string]any{"type": "string"},
				"enumProp": map[string]any{"type": "string", "enum": []any{"a", "b"}},
				"arrProp":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"custProp": map[string]any{"$ref": "#/$defs/Email"},
				"objProp": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
					"properties":           map[string]any{"childProp": map[string]any{"type": "integer"}},
					"required":             []any{"childProp"},
				},
			},
			"required": []any{"strProp"},
		},
	}
	ev.Rules.Defs = map[string]any{
		"Email": map[string]any{"type": "string", "enum": []any{"x"}},
	}

	remoteResp := constructTrackingPlanWithSchemas("tp", "TP", []catalog.TrackingPlanEventSchema{ev})
	remotePlan, err := providers.NewJSONSchemaPlanProvider("tp", &mockTrackingPlanStore{trackingPlanWithSchemas: remoteResp}).GetTrackingPlan(ctx)
	require.NoError(t, err)

	assert.Equal(t, remotePlan.Rules, localPlan.Rules)
}
