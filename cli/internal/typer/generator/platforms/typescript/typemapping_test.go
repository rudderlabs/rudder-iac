package typescript

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			got, err := resolvePropertyType(&plan.Property{Types: tt.types})
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
			})
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
			got, err := resolvePropertyType(&plan.Property{Types: tt.types})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestResolvePropertyType_CustomType(t *testing.T) {
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
			"custom string-backed type → string",
			&plan.Property{Types: []plan.PropertyType{emailCT}},
			"string",
		},
		{
			"custom object type → open record (named aliases deferred)",
			&plan.Property{Types: []plan.PropertyType{openObjectCT}},
			"Record<string, unknown>",
		},
		{
			"custom array type with primitive item",
			&plan.Property{Types: []plan.PropertyType{stringArrayCT}},
			"string[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePropertyType(tt.prop)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
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
