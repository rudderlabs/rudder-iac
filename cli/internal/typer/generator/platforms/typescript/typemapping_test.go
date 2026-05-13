package typescript

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry() *core.NameRegistry {
	return core.NewNameRegistry(typescriptCollisionHandler)
}

func TestResolvePropertyType_Primitives(t *testing.T) {
	tests := []struct {
		name     string
		types    []plan.PropertyType
		expected string
	}{
		{"empty types → unknown", []plan.PropertyType{}, "unknown"},
		{"string", []plan.PropertyType{plan.PrimitiveTypeString}, "string"},
		{"integer → number", []plan.PropertyType{plan.PrimitiveTypeInteger}, "number"},
		{"number", []plan.PropertyType{plan.PrimitiveTypeNumber}, "number"},
		{"boolean", []plan.PropertyType{plan.PrimitiveTypeBoolean}, "boolean"},
		{"null", []plan.PropertyType{plan.PrimitiveTypeNull}, "null"},
		{"object → open record", []plan.PropertyType{plan.PrimitiveTypeObject}, "Record<string, unknown>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePropertyType(&plan.Property{Types: tt.types}, "", "", newTestRegistry())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolvePropertyType_Arrays(t *testing.T) {
	tests := []struct {
		name      string
		itemTypes []plan.PropertyType
		expected  string
	}{
		{"untyped array → unknown[]", nil, "unknown[]"},
		{"empty itemTypes → unknown[]", []plan.PropertyType{}, "unknown[]"},
		{"string array", []plan.PropertyType{plan.PrimitiveTypeString}, "string[]"},
		{"integer array → number[]", []plan.PropertyType{plan.PrimitiveTypeInteger}, "number[]"},
		{"boolean array", []plan.PropertyType{plan.PrimitiveTypeBoolean}, "boolean[]"},
		{
			"mixed item types → Array<...>",
			[]plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger},
			"Array<string | number>",
		},
		{
			"string-or-null array → Array<string | null>",
			[]plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNull},
			"Array<string | null>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePropertyType(&plan.Property{
				Types:     []plan.PropertyType{plan.PrimitiveTypeArray},
				ItemTypes: tt.itemTypes,
			}, "", "", newTestRegistry())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolvePropertyType_MultiType(t *testing.T) {
	tests := []struct {
		name     string
		types    []plan.PropertyType
		expected string
	}{
		{
			"string or null",
			[]plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNull},
			"string | null",
		},
		{
			"string, integer, boolean",
			[]plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger, plan.PrimitiveTypeBoolean},
			"string | number | boolean",
		},
		{
			"number or null",
			[]plan.PropertyType{plan.PrimitiveTypeNumber, plan.PrimitiveTypeNull},
			"number | null",
		},
		{
			// integer and number both collapse to TS `number`; the union must
			// dedupe so the output reads `number | boolean`, not `number | number | boolean`.
			"integer and number dedupe",
			[]plan.PropertyType{plan.PrimitiveTypeInteger, plan.PrimitiveTypeNumber, plan.PrimitiveTypeBoolean},
			"number | boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePropertyType(&plan.Property{Types: tt.types}, "", "", newTestRegistry())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolvePropertyType_CustomType(t *testing.T) {
	// Custom types resolve to a registered name, not the underlying primitive.
	// resolvePropertyType registers the name on demand if it hasn't been
	// processed already, so callers don't have to pre-walk the plan.
	emailCT := plan.CustomType{
		Name: "email",
		Type: plan.PrimitiveTypeString,
	}
	openObjectCT := plan.CustomType{
		Name:   "page_data",
		Type:   plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{},
	}
	stringArrayCT := plan.CustomType{
		Name:     "tags",
		Type:     plan.PrimitiveTypeArray,
		ItemType: plan.PrimitiveTypeString,
	}

	tests := []struct {
		name     string
		prop     *plan.Property
		expected string
	}{
		{
			"primitive custom type → registered alias",
			&plan.Property{Types: []plan.PropertyType{emailCT}},
			"CustomTypeEmail",
		},
		{
			"object custom type → registered alias",
			&plan.Property{Types: []plan.PropertyType{openObjectCT}},
			"CustomTypePageData",
		},
		{
			"array custom type → registered alias",
			&plan.Property{Types: []plan.PropertyType{stringArrayCT}},
			"CustomTypeTags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePropertyType(tt.prop, "", "", newTestRegistry())
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolvePropertyType_EnumOverride(t *testing.T) {
	// When the surrounding interface builder has already resolved an enum alias
	// for the property, that alias takes precedence over inline type
	// resolution. The plan-level types are irrelevant in this case.
	got, err := resolvePropertyType(
		&plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeString}},
		"PropertyDeviceType",
		"",
		newTestRegistry(),
	)
	require.NoError(t, err)
	assert.Equal(t, "PropertyDeviceType", got)
}

func TestResolvePropertyType_NestedOverride(t *testing.T) {
	// When the property has an inline nested-object schema, the caller hoists
	// it into a top-level interface and passes the registered name. That name
	// short-circuits the underlying `object` primitive resolution.
	got, err := resolvePropertyType(
		&plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
		"",
		"TrackUserSignedUpContext",
		newTestRegistry(),
	)
	require.NoError(t, err)
	assert.Equal(t, "TrackUserSignedUpContext", got)
}

func TestBuildInterfaceWithNested_OptionalMultiType(t *testing.T) {
	// Optional + multi-type must render as `prop?: A | B`, not
	// `prop: A | B | undefined`. Optionality lives on the field marker, not in
	// the type union — `null` belongs in the union when the plan says so,
	// `undefined` does not. Matters under exactOptionalPropertyTypes.
	schema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"value": {
				Property: plan.Property{
					Name:  "value",
					Types: []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNumber},
				},
			},
		},
	}

	ctx := &TSContext{}
	iface, err := buildInterfaceWithNested("Parent", "", schema, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSInterface{
		Name: "Parent",
		Properties: []TSInterfaceProperty{
			{Name: "value", Type: "string | number", Optional: true},
		},
	}, iface)
}

func TestBuildInterfaceWithNested_MultiTypeInsideNestedObject(t *testing.T) {
	// A multi-type property inside a hoisted nested-object schema must resolve
	// to a union on the nested interface, not collapse to `unknown` or fall
	// back to the underlying object primitive.
	schema := &plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"details": {
				Property: plan.Property{
					Name:  "details",
					Types: []plan.PropertyType{plan.PrimitiveTypeObject},
				},
				Required: true,
				Schema: &plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"value": {
							Property: plan.Property{
								Name:  "value",
								Types: []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNumber},
							},
							Required: true,
						},
					},
				},
			},
		},
	}

	ctx := &TSContext{}
	iface, err := buildInterfaceWithNested("Parent", "", schema, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSInterface{
		Name: "Parent",
		Properties: []TSInterfaceProperty{
			{Name: "details", Type: "ParentDetails"},
		},
	}, iface)

	assert.Equal(t, []TSInterface{
		{
			Name: "ParentDetails",
			Properties: []TSInterfaceProperty{
				{Name: "value", Type: "string | number"},
			},
		},
	}, ctx.NestedInterfaces)
}

func TestIsValidTSIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"foo", true},
		{"_foo", true},
		{"$foo", true},
		{"foo_bar", true},
		{"foo123", true},
		{"foo$bar", true},
		// invalid: starts with digit
		{"1foo", false},
		// invalid: contains non-ident char
		{"foo-bar", false},
		{"foo bar", false},
		{"用户名", false},
		// edge cases
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, isValidTSIdentifier(tt.input))
		})
	}
}
