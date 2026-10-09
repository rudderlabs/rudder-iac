package golang

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeSignature(t *testing.T) {
	tests := []struct {
		name      string
		property  plan.Property
		signature string
	}{
		{"untyped", plan.Property{}, ";items:;enum:"},
		{"string", plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeString}}, "string;items:;enum:"},
		{"integer", plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeInteger}}, "integer;items:;enum:"},
		{"number", plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeNumber}}, "number;items:;enum:"},
		{"boolean", plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeBoolean}}, "boolean;items:;enum:"},
		{"object", plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeObject}}, "object;items:;enum:"},
		{"array without item types", plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeArray}}, "array;items:;enum:"},
		{
			"types are sorted",
			plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNull, plan.PrimitiveTypeInteger}},
			"integer|null|string;items:;enum:",
		},
		{
			"item types are sorted",
			plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeArray}, ItemTypes: []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger}},
			"array;items:integer|string;enum:",
		},
		{
			"custom types by name",
			plan.Property{Types: []plan.PropertyType{plan.CustomType{Name: "email", Type: plan.PrimitiveTypeString}}},
			"custom:email;items:;enum:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.signature, typeSignature(tt.property))
		})
	}
}

func TestPropertyTypeOrder(t *testing.T) {
	property := func(name, description string, typ plan.PrimitiveType) plan.PropertySchema {
		return plan.PropertySchema{Property: plan.Property{Name: name, Description: description, Types: []plan.PropertyType{typ}}}
	}
	rule := func(event string, properties ...plan.PropertySchema) plan.EventRule {
		schema := plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}}
		for _, p := range properties {
			schema.Properties[p.Property.Name] = p
		}
		return plan.EventRule{
			Event:   plan.Event{EventType: plan.EventTypeTrack, Name: event},
			Section: plan.IdentitySectionProperties,
			Schema:  schema,
		}
	}
	p := &plan.TrackingPlan{Rules: []plan.EventRule{
		rule("B Event", property("amount", "amount as text", plan.PrimitiveTypeString), property("user_id", "", plan.PrimitiveTypeString)),
		rule("A Event", property("amount", "amount as a number", plan.PrimitiveTypeNumber), property("userId", "", plan.PrimitiveTypeString)),
		rule("C Event", property("amount", "described later", plan.PrimitiveTypeNumber)),
	}}

	rules, err := trackRules(p)
	require.NoError(t, err)
	ctx := &GoContext{}
	_, err = addPropertyTypes(ctx, rules, core.NewNameRegistry(core.DefaultCollisionHandler))
	require.NoError(t, err)

	// Ordered by name, then signature ("number" < "string"); one type per
	// (name, signature), described by the first rule in rule-key order.
	assert.Equal(t, []GoPropertyType{
		{Alias: &GoTypeAlias{Name: "PropertyAmount", Doc: "PropertyAmount represents the property \"amount\".\n\namount as a number", Type: "float64"}},
		{Alias: &GoTypeAlias{Name: "PropertyAmount1", Doc: "PropertyAmount1 represents the property \"amount\".\n\namount as text", Type: "string"}},
		{Alias: &GoTypeAlias{Name: "PropertyUserID", Doc: "PropertyUserID represents the property \"userId\".", Type: "string"}},
		{Alias: &GoTypeAlias{Name: "PropertyUserID1", Doc: "PropertyUserID1 represents the property \"user_id\".", Type: "string"}},
	}, ctx.PropertyTypes)
}

// An item union is registered right after its owner, so it keeps
// {Owner}Item and a property sorted later that pascal-cases to the same name
// takes the suffix.
func TestItemUnionNameOrder(t *testing.T) {
	p := &plan.TrackingPlan{Rules: []plan.EventRule{{
		Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Some Event"},
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"foo": {Property: plan.Property{
				Name:      "foo",
				Types:     []plan.PropertyType{plan.PrimitiveTypeArray},
				ItemTypes: []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger},
			}},
			"fooItem": {Property: plan.Property{Name: "fooItem", Types: []plan.PropertyType{plan.PrimitiveTypeString}}},
		}},
	}}}

	rules, err := trackRules(p)
	require.NoError(t, err)
	ctx := &GoContext{}
	_, err = addPropertyTypes(ctx, rules, core.NewNameRegistry(core.DefaultCollisionHandler))
	require.NoError(t, err)

	assert.Equal(t, []GoPropertyType{
		{Union: &GoUnion{
			Name: "PropertyFooItem",
			Doc: "PropertyFooItem is one item of PropertyFoo: a string or integer.\n" +
				"Build it with one of the NewPropertyFooItem* functions.\n" +
				"Null is not a member, so the zero value cannot be sent.",
			Members: []GoUnionMember{
				{Name: "NewPropertyFooItemString", Doc: "returns a string value.", Type: "string"},
				{Name: "NewPropertyFooItemInteger", Doc: "returns an integer value.", Type: "int64"},
			},
		}},
		{Alias: &GoTypeAlias{Name: "PropertyFoo", Doc: "PropertyFoo represents the property \"foo\".", Type: "[]PropertyFooItem"}},
		{Alias: &GoTypeAlias{Name: "PropertyFooItem1", Doc: "PropertyFooItem1 represents the property \"fooItem\".", Type: "string"}},
	}, ctx.PropertyTypes)
}
