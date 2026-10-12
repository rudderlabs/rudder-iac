package golang

import (
	"encoding/json"
	"testing"

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
		{
			"enum values normalized and deduplicated, in plan order",
			plan.Property{
				Types:  []plan.PropertyType{plan.PrimitiveTypeNumber, plan.PrimitiveTypeInteger},
				Config: &plan.PropertyConfig{Enum: []any{2, 1.0, json.Number("1e0"), 0.5}},
			},
			"integer|number;items:;enum:[2,1,0.5]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := resolveEnum(tt.property)
			require.NoError(t, err)
			assert.Equal(t, tt.signature, typeSignature(tt.property, e))
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

	ctx, err := newContext(p, "", "ruddertyper")
	require.NoError(t, err)

	// Ordered by name, then signature ("number" < "string"); one type per
	// (name, signature), described by the first rule in rule-key order.
	assert.Equal(t, []GoDecl{
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

	ctx, err := newContext(p, "", "ruddertyper")
	require.NoError(t, err)

	assert.Equal(t, []GoDecl{
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

// Every custom type name is registered before the structs custom types
// hoist, so a custom type named UserProfileAddress keeps its name and
// UserProfile's hoisted address takes the suffix.
func TestCustomTypeNameBeatsHoistedName(t *testing.T) {
	var (
		street      = plan.PropertySchema{Property: plan.Property{Name: "street", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true}
		userProfile = &plan.CustomType{
			Name: "UserProfile",
			Type: plan.PrimitiveTypeObject,
			Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"address": {
					Property: plan.Property{Name: "address", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
					Schema:   &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{"street": street}},
				},
			}},
		}
		userProfileAddress = &plan.CustomType{Name: "UserProfileAddress", Type: plan.PrimitiveTypeString}
	)
	p := trackPlan(
		plan.Property{Name: "profile", Types: []plan.PropertyType{userProfile}},
		plan.Property{Name: "address", Types: []plan.PropertyType{userProfileAddress}},
	)

	ctx, err := newContext(p, "", "ruddertyper")
	require.NoError(t, err)

	assert.Equal(t, []GoDecl{
		{Struct: &GoStruct{
			Name:         "CustomTypeUserProfile",
			Doc:          "CustomTypeUserProfile represents the custom type \"UserProfile\".",
			DeclaredKeys: []string{"address"},
			Fields:       []GoField{{Name: "Address", Key: "address", Type: "*CustomTypeUserProfileAddress1", Pointer: true}},
		}},
		{Struct: &GoStruct{
			Name:         "CustomTypeUserProfileAddress1",
			Doc:          "CustomTypeUserProfileAddress1 is the object in CustomTypeUserProfile.Address.",
			DeclaredKeys: []string{"street"},
			Fields:       []GoField{{Name: "Street", Key: "street", Type: "PropertyStreet", Required: true}},
		}},
		{Alias: &GoTypeAlias{
			Name: "CustomTypeUserProfileAddress",
			Doc:  "CustomTypeUserProfileAddress represents the custom type \"UserProfileAddress\".",
			Type: "string",
		}},
	}, ctx.CustomTypes)
}

// Hoisted names spell the path, so foo.barBaz and fooBar.baz both make
// {Parent}FooBarBaz; the one registered second, depth-first in wire-key
// order, takes the suffix.
func TestHoistedNameCollision(t *testing.T) {
	object := func(name string, properties map[string]plan.PropertySchema) plan.PropertySchema {
		return plan.PropertySchema{
			Property: plan.Property{Name: name, Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
			Schema:   &plan.ObjectSchema{Properties: properties},
		}
	}
	leaf := map[string]plan.PropertySchema{"x": {Property: plan.Property{Name: "x", Types: []plan.PropertyType{plan.PrimitiveTypeString}}}}
	p := trackPlan()
	p.Rules[0].Schema.Properties = map[string]plan.PropertySchema{
		"foo":    object("foo", map[string]plan.PropertySchema{"barBaz": object("barBaz", leaf)}),
		"fooBar": object("fooBar", map[string]plan.PropertySchema{"baz": object("baz", leaf)}),
	}

	ctx, err := newContext(p, "", "ruddertyper")
	require.NoError(t, err)

	x := []GoField{{Name: "X", Key: "x", Type: "*PropertyX", Pointer: true}}
	assert.Equal(t, []GoDecl{
		{Struct: &GoStruct{
			Name:         "TrackSomeEventProperties",
			Doc:          "TrackSomeEventProperties holds the properties of \"Some Event\".",
			Payload:      "Properties",
			DeclaredKeys: []string{"foo", "fooBar"},
			Fields: []GoField{
				{Name: "Foo", Key: "foo", Type: "*TrackSomeEventPropertiesFoo", Pointer: true},
				{Name: "FooBar", Key: "fooBar", Type: "*TrackSomeEventPropertiesFooBar", Pointer: true},
			},
		}},
		{Struct: &GoStruct{
			Name:         "TrackSomeEventPropertiesFoo",
			Doc:          "TrackSomeEventPropertiesFoo is the object in TrackSomeEventProperties.Foo.",
			DeclaredKeys: []string{"barBaz"},
			Fields:       []GoField{{Name: "BarBaz", Key: "barBaz", Type: "*TrackSomeEventPropertiesFooBarBaz", Pointer: true}},
		}},
		{Struct: &GoStruct{
			Name:         "TrackSomeEventPropertiesFooBarBaz",
			Doc:          "TrackSomeEventPropertiesFooBarBaz is the object in TrackSomeEventPropertiesFoo.BarBaz.",
			DeclaredKeys: []string{"x"},
			Fields:       x,
		}},
		{Struct: &GoStruct{
			Name:         "TrackSomeEventPropertiesFooBar",
			Doc:          "TrackSomeEventPropertiesFooBar is the object in TrackSomeEventProperties.FooBar.",
			DeclaredKeys: []string{"baz"},
			Fields:       []GoField{{Name: "Baz", Key: "baz", Type: "*TrackSomeEventPropertiesFooBarBaz1", Pointer: true}},
		}},
		{Struct: &GoStruct{
			Name:         "TrackSomeEventPropertiesFooBarBaz1",
			Doc:          "TrackSomeEventPropertiesFooBarBaz1 is the object in TrackSomeEventPropertiesFooBar.Baz.",
			DeclaredKeys: []string{"x"},
			Fields:       x,
		}},
	}, ctx.Payloads)
}

// A hoisted property's description documents the field's type: the struct of
// an object, or the union of object and another type, whose Object struct is
// only one member and does not repeat it.
func TestHoistedDescription(t *testing.T) {
	var (
		leaf = &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{"x": {Property: plan.Property{Name: "x", Types: []plan.PropertyType{plan.PrimitiveTypeString}}}}}
		p    = trackPlan()
	)
	p.Rules[0].Schema.Properties = map[string]plan.PropertySchema{
		"foo": {
			Property: plan.Property{Name: "foo", Description: "the foo object", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
			Schema:   leaf,
		},
		"bar": {
			Property: plan.Property{Name: "bar", Description: "the bar value", Types: []plan.PropertyType{plan.PrimitiveTypeObject, plan.PrimitiveTypeString}},
			Schema:   leaf,
		},
	}

	ctx, err := newContext(p, "", "ruddertyper")
	require.NoError(t, err)

	x := []GoField{{Name: "X", Key: "x", Type: "*PropertyX", Pointer: true}}
	assert.Equal(t, []GoDecl{
		{Struct: &GoStruct{
			Name:         "TrackSomeEventProperties",
			Doc:          "TrackSomeEventProperties holds the properties of \"Some Event\".",
			Payload:      "Properties",
			DeclaredKeys: []string{"bar", "foo"},
			Fields: []GoField{
				{Name: "Bar", Key: "bar", Type: "*TrackSomeEventPropertiesBar", Pointer: true},
				{Name: "Foo", Key: "foo", Type: "*TrackSomeEventPropertiesFoo", Pointer: true},
			},
		}},
		{Union: &GoUnion{
			Name: "TrackSomeEventPropertiesBar",
			Doc: "TrackSomeEventPropertiesBar is the value in TrackSomeEventProperties.Bar: a string or object.\n" +
				"Build it with one of the NewTrackSomeEventPropertiesBar* functions.\n" +
				"Null is not a member, so the zero value cannot be sent.\n\n" +
				"the bar value",
			Members: []GoUnionMember{
				{Name: "NewTrackSomeEventPropertiesBarString", Doc: "returns a string value.", Type: "string"},
				{Name: "NewTrackSomeEventPropertiesBarObject", Doc: "returns an object value.", Type: "TrackSomeEventPropertiesBarObject"},
			},
		}},
		{Struct: &GoStruct{
			Name:         "TrackSomeEventPropertiesBarObject",
			Doc:          "TrackSomeEventPropertiesBarObject is the object in TrackSomeEventProperties.Bar.",
			DeclaredKeys: []string{"x"},
			Fields:       x,
		}},
		{Struct: &GoStruct{
			Name:         "TrackSomeEventPropertiesFoo",
			Doc:          "TrackSomeEventPropertiesFoo is the object in TrackSomeEventProperties.Foo.\n\nthe foo object",
			DeclaredKeys: []string{"x"},
			Fields:       x,
		}},
	}, ctx.Payloads)
}
