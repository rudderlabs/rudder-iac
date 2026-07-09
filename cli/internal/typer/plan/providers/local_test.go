package providers_test

import (
	"context"
	"encoding/json"
	"os"
	"sort"
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

// TestLocalCatalogPlanProvider_MatchesRemoteConventions covers the three
// constructs buildCatalog omits, each a place the local converter previously
// diverged from the remote JSON-Schema parser: multi-type array/object
// properties, and object custom types (additionalProperties + description).
func TestLocalCatalogPlanProvider_MatchesRemoteConventions(t *testing.T) {
	dc := localcatalog.New()
	dc.Properties = []localcatalog.PropertyV1{
		{LocalID: "child", Name: "childProp", Type: "integer"},
		{LocalID: "mtarr", Name: "mtArr", Types: []string{"array", "null"}, ItemType: "string"},
		{LocalID: "mtobj", Name: "mtObj", Types: []string{"object", "null"}},
		{LocalID: "addrp", Name: "addrProp", Type: "#custom-type:addr"},
	}
	dc.CustomTypes = []localcatalog.CustomTypeV1{
		{
			LocalID:     "addr",
			Name:        "Address",
			Type:        "object",
			Description: "a postal address",
			Config:      map[string]any{"additional_properties": false},
			Properties:  []localcatalog.CustomTypePropertyV1{{Property: "#property:child", Required: true}},
		},
	}
	dc.Events = []localcatalog.EventV1{{LocalID: "evt", Name: "My Event", Type: "track"}}
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
						{Property: "#property:mtarr"},
						{Property: "#property:mtobj", Properties: []*localcatalog.TPRulePropertyV1{
							{Property: "#property:child", Required: true},
						}},
						{Property: "#property:addrp"},
					},
				},
			},
		},
	}

	got, err := providers.NewLocalCatalogPlanProvider(dc, "tp").GetTrackingPlan(context.Background())
	require.NoError(t, err)
	props := got.Rules[0].Schema.Properties

	// Multi-type array property still resolves ItemTypes (previously dropped).
	assert.Equal(t, []plan.PropertyType{plan.PrimitiveTypeArray, plan.PrimitiveTypeNull}, props["mtArr"].Property.Types)
	assert.Equal(t, []plan.PropertyType{plan.PrimitiveTypeString}, props["mtArr"].Property.ItemTypes)

	// Multi-type object property still builds a nested schema (previously dropped).
	require.NotNil(t, props["mtObj"].Schema)
	assert.Contains(t, props["mtObj"].Schema.Properties, "childProp")

	// Object custom type derives additionalProperties from config (was hardcoded
	// true) and does not carry Description (the remote parser never sets it).
	ct, ok := props["addrProp"].Property.Types[0].(*plan.CustomType)
	require.True(t, ok)
	assert.Empty(t, ct.Description)
	require.NotNil(t, ct.Schema)
	assert.False(t, ct.Schema.AdditionalProperties)
}

// TestLocalAndRemoteProduceEquivalentPlan is the guard that justifies having two
// independent adapters onto plan.TrackingPlan: for the same tracking plan, the
// local converter and the remote JSON-Schema parser must produce the same rules.
//
// Both sides are driven from a shared on-disk fixture rather than hand-authored
// structs, so a genuine divergence between the two code paths fails here instead
// of reaching users:
//   - local: the real sample project testdata/project loaded through the real
//     offline provider (LoadSpec -> ExpandRefs -> converter);
//   - remote: a real GetTrackingPlanWithSchemas response captured from a workspace
//     after applying that same project (regenerate with `go run capture_golden.go`).
//
// Metadata differs by design (the local plan has no remote URL/version), so only
// Rules are compared. Rule order is not part of the contract — local follows spec
// order, remote follows API order — so they are compared by event name.
func TestLocalAndRemoteProduceEquivalentPlan(t *testing.T) {
	ctx := context.Background()

	local, err := providers.NewLocalCatalogPlanProviderForProject("../testdata/project", "typer-test-tracking-plan")
	require.NoError(t, err)
	localPlan, err := local.GetTrackingPlan(ctx)
	require.NoError(t, err)

	data, err := os.ReadFile("testdata/remote_tracking_plan.golden.json")
	require.NoError(t, err)
	var golden catalog.TrackingPlanWithSchemas
	require.NoError(t, json.Unmarshal(data, &golden))
	remotePlan, err := providers.NewJSONSchemaPlanProvider("tp", &mockTrackingPlanStore{trackingPlanWithSchemas: &golden}).GetTrackingPlan(ctx)
	require.NoError(t, err)

	byEvent := func(rules []plan.EventRule) {
		sort.Slice(rules, func(i, j int) bool { return rules[i].Event.Name < rules[j].Event.Name })
	}
	byEvent(localPlan.Rules)
	byEvent(remotePlan.Rules)

	// Known, documented asymmetry: a variant case's DisplayName/Description are
	// carried by the local specs but not by the remote JSON-Schema wire format
	// (the allOf/if/then envelope has nowhere to put them — see jsonschema.go's
	// "could be extracted from metadata if available"). Local is the richer side
	// here; rather than degrade it to match, we blank these two fields on both
	// sides before comparing, so the test still guards every structural field.
	// Everything else must be byte-identical.
	blankVariantLabels(localPlan.Rules)
	blankVariantLabels(remotePlan.Rules)

	assert.Equal(t, remotePlan.Rules, localPlan.Rules)
}

// blankVariantLabels clears the two variant-case fields the remote wire format
// cannot carry (DisplayName, Description) everywhere they can appear — event-rule
// variants and variants nested inside object custom types — so the equivalence
// assertion compares only what both paths can represent. See the call site for
// the rationale. Custom types are pointer-shared and can be cyclic, so a visited
// set bounds the walk.
func blankVariantLabels(rules []plan.EventRule) {
	seen := map[*plan.CustomType]bool{}
	var walkSchema func(*plan.ObjectSchema)
	var walkVariants func([]plan.Variant)
	var walkType func(plan.PropertyType)

	walkType = func(t plan.PropertyType) {
		ct := plan.AsCustomType(t)
		if ct == nil || seen[ct] {
			return
		}
		seen[ct] = true
		walkSchema(ct.Schema)
		walkType(ct.ItemType)
		walkVariants(ct.Variants)
	}
	walkSchema = func(s *plan.ObjectSchema) {
		if s == nil {
			return
		}
		for _, ps := range s.Properties {
			for _, t := range ps.Property.Types {
				walkType(t)
			}
			for _, t := range ps.Property.ItemTypes {
				walkType(t)
			}
			walkSchema(ps.Schema)
		}
	}
	walkVariants = func(vs []plan.Variant) {
		for i := range vs {
			for j := range vs[i].Cases {
				vs[i].Cases[j].DisplayName = ""
				vs[i].Cases[j].Description = ""
				walkSchema(&vs[i].Cases[j].Schema)
			}
			walkSchema(vs[i].DefaultSchema)
		}
	}
	for i := range rules {
		walkSchema(&rules[i].Schema)
		walkVariants(rules[i].Variants)
	}
}
