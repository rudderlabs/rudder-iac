package golang

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enumProperty(name string, types []plan.PropertyType, values ...any) plan.Property {
	return plan.Property{Name: name, Types: types, Config: &plan.PropertyConfig{Enum: values}}
}

var (
	stringType  = []plan.PropertyType{plan.PrimitiveTypeString}
	integerType = []plan.PropertyType{plan.PrimitiveTypeInteger}
	numberType  = []plan.PropertyType{plan.PrimitiveTypeNumber}
)

func TestResolveEnum(t *testing.T) {
	tests := []struct {
		name     string
		property plan.Property
		want     *effectiveEnum
	}{
		{name: "no enum", property: plan.Property{Types: stringType}},
		{
			name:     "strings",
			property: enumProperty("p", stringType, "a", "b"),
			want: &effectiveEnum{base: "string", signature: `["a","b"]`, values: []GoEnumValue{
				{Name: "A", Text: `"a"`, Expr: `"a"`},
				{Name: "B", Text: `"b"`, Expr: `"b"`},
			}},
		},
		{
			name:     "integral numbers with integer declared",
			property: enumProperty("p", integerType, 200, -1),
			want: &effectiveEnum{base: "int64", signature: "[200,-1]", values: []GoEnumValue{
				{Name: "200", Text: "200", Expr: "int64(200)"},
				{Name: "Neg1", Text: "-1", Expr: "int64(-1)"},
			}},
		},
		{
			name:     "integral numbers with integer and number declared",
			property: enumProperty("p", []plan.PropertyType{plan.PrimitiveTypeInteger, plan.PrimitiveTypeNumber}, 1, 2),
			want: &effectiveEnum{base: "float64", signature: "[1,2]", values: []GoEnumValue{
				{Name: "1", Text: "1", Expr: "float64(1)"},
				{Name: "2", Text: "2", Expr: "float64(2)"},
			}},
		},
		{
			name:     "numbers with number declared",
			property: enumProperty("p", numberType, 1.5, 5.0),
			want: &effectiveEnum{base: "float64", signature: "[1.5,5]", values: []GoEnumValue{
				{Name: "1_5", Text: "1.5", Expr: "1.5"},
				{Name: "5", Text: "5", Expr: "float64(5)"},
			}},
		},
		{
			name:     "integral numbers without declared types",
			property: enumProperty("p", nil, 1),
			want: &effectiveEnum{base: "float64", signature: "[1]", values: []GoEnumValue{
				{Name: "1", Text: "1", Expr: "float64(1)"},
			}},
		},
		{
			name:     "booleans",
			property: enumProperty("p", []plan.PropertyType{plan.PrimitiveTypeBoolean}, true, false),
			want: &effectiveEnum{base: "bool", signature: "[true,false]", values: []GoEnumValue{
				{Name: "True", Text: "true", Expr: "true"},
				{Name: "False", Text: "false", Expr: "false"},
			}},
		},
		{
			name:     "mixed JSON types: integers that fit are int64, other numbers float64",
			property: enumProperty("p", nil, "active", 1, true, 2.5, 1e20),
			want: &effectiveEnum{signature: `["active",1,true,2.5,100000000000000000000]`, values: []GoEnumValue{
				{Name: "Active", Text: `"active"`, Expr: `"active"`},
				{Name: "1", Text: "1", Expr: "int64(1)"},
				{Name: "True", Text: "true", Expr: "true"},
				{Name: "2_5", Text: "2.5", Expr: "2.5"},
				{Name: "100000000000000000000", Text: "100000000000000000000", Expr: "float64(100000000000000000000)"},
			}},
		},
		{
			name:     "null allowed by the declared types",
			property: enumProperty("p", []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNull}, nil, "a"),
			want: &effectiveEnum{base: "string", null: true, signature: `[null,"a"]`, values: []GoEnumValue{
				{Name: "A", Text: `"a"`, Expr: `"a"`},
			}},
		},
		{
			name:     "null without declared types",
			property: enumProperty("p", nil, "a", nil),
			want: &effectiveEnum{base: "string", null: true, signature: `["a",null]`, values: []GoEnumValue{
				{Name: "A", Text: `"a"`, Expr: `"a"`},
			}},
		},
		{
			name:     "only null",
			property: enumProperty("p", []plan.PropertyType{plan.PrimitiveTypeInteger, plan.PrimitiveTypeNull}, nil),
			want:     &effectiveEnum{null: true, signature: "[null]"},
		},
		{
			name:     "values the declared types do not allow are left out",
			property: enumProperty("p", integerType, "a", nil, 1.5, 1, true),
			want: &effectiveEnum{base: "int64", signature: "[1]", values: []GoEnumValue{
				{Name: "1", Text: "1", Expr: "int64(1)"},
			}},
		},
		{
			name:     "equal values deduplicated after normalization, keeping the first",
			property: enumProperty("p", numberType, 1, 2.5, 1.0, json.Number("1e0"), int64(1), json.Number("2.50")),
			want: &effectiveEnum{base: "float64", signature: "[1,2.5]", values: []GoEnumValue{
				{Name: "1", Text: "1", Expr: "float64(1)"},
				{Name: "2_5", Text: "2.5", Expr: "2.5"},
			}},
		},
		{
			name:     "integers keep their precision",
			property: enumProperty("p", integerType, json.Number("9007199254740993"), json.Number("9007199254740992")),
			want: &effectiveEnum{base: "int64", signature: "[9007199254740993,9007199254740992]", values: []GoEnumValue{
				{Name: "9007199254740993", Text: "9007199254740993", Expr: "int64(9007199254740993)"},
				{Name: "9007199254740992", Text: "9007199254740992", Expr: "int64(9007199254740992)"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveEnum(tt.property)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Every row of the spec's Enum Constant Names table.
func TestEnumConstantNames(t *testing.T) {
	tests := []struct {
		value any
		types []plan.PropertyType
		want  string
	}{
		{"GET", stringType, "GET"},
		{"smartTV", stringType, "SmartTV"},
		{"IoT-Device", stringType, "IoTDevice"},
		{"активный", stringType, "Активный"},
		{"已完成", stringType, "已完成"},
		{"🎯", stringType, "U1F3AF"},
		{"!!!", stringType, "U21U21U21"},
		{"", stringType, "Empty"},
		{200, integerType, "200"},
		{-1, integerType, "Neg1"},
		{1.5, numberType, "1_5"},
		{5.0, numberType, "5"},
		{-0.25, numberType, "Neg0_25"},
		{true, nil, "True"},
		{false, nil, "False"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.value), func(t *testing.T) {
			e, err := resolveEnum(enumProperty("p", tt.types, tt.value))
			require.NoError(t, err)
			require.Len(t, e.values, 1)
			assert.Equal(t, tt.want, e.values[0].Name)
		})
	}
}

// Enum constants and {Type}Values functions are registered after every type
// name, and every {Type}Values before any constant, so a colliding constant
// takes the suffix.
func TestEnumNameCollisions(t *testing.T) {
	stringEnum := func(name, property string, values ...GoEnumValue) GoPropertyType {
		return GoPropertyType{Enum: &GoEnum{
			Name:       name,
			Doc:        name + " represents the property " + strconv.Quote(property) + ".",
			Base:       "string",
			ConstDoc:   "Allowed values of " + name + ".",
			ValuesFunc: name + "Values",
			Values:     values,
		}}
	}
	status := func(values ...GoEnumValue) GoPropertyType {
		return stringEnum("PropertyStatus", "status", values...)
	}
	tests := []struct {
		name       string
		properties []plan.Property
		want       []GoPropertyType
	}{
		{
			name:       "values with the same name",
			properties: []plan.Property{enumProperty("status", stringType, "Active", "active")},
			want: []GoPropertyType{status(
				GoEnumValue{Name: "PropertyStatusActive", Text: `"Active"`, Expr: `"Active"`},
				GoEnumValue{Name: "PropertyStatusActive1", Text: `"active"`, Expr: `"active"`},
			)},
		},
		{
			name:       "a value named values",
			properties: []plan.Property{enumProperty("status", stringType, "values")},
			want: []GoPropertyType{status(
				GoEnumValue{Name: "PropertyStatusValues1", Text: `"values"`, Expr: `"values"`},
			)},
		},
		{
			name: "a property type named like a constant",
			properties: []plan.Property{
				enumProperty("status", stringType, "active"),
				{Name: "statusActive", Types: stringType},
			},
			want: []GoPropertyType{
				status(GoEnumValue{Name: "PropertyStatusActive1", Text: `"active"`, Expr: `"active"`}),
				{Alias: &GoTypeAlias{
					Name: "PropertyStatusActive",
					Doc:  `PropertyStatusActive represents the property "statusActive".`,
					Type: "string",
				}},
			},
		},
		{
			name: "a value named like another enum's Values function",
			properties: []plan.Property{
				enumProperty("a", stringType, "bValues"),
				enumProperty("aB", stringType, "x"),
			},
			want: []GoPropertyType{
				stringEnum("PropertyA", "a", GoEnumValue{Name: "PropertyABValues1", Text: `"bValues"`, Expr: `"bValues"`}),
				stringEnum("PropertyAB", "aB", GoEnumValue{Name: "PropertyABX", Text: `"x"`, Expr: `"x"`}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := newContext(trackPlan(tt.properties...), "", "ruddertyper")
			require.NoError(t, err)
			assert.Equal(t, tt.want, ctx.PropertyTypes)
		})
	}
}

func TestGenerateRejectsInvalidEnums(t *testing.T) {
	tests := []struct {
		name     string
		property plan.Property
		wantErr  string
	}{
		{
			name:     "no value of the declared types",
			property: enumProperty("kind", stringType, 1, 2),
			wantErr:  `mapping property "kind": no enum value has a declared type (string)`,
		},
		{
			name:     "no integral value for an integer enum",
			property: enumProperty("kind", integerType, 1.5),
			wantErr:  `mapping property "kind": no enum value has a declared type (integer)`,
		},
		{
			name:     "object value",
			property: enumProperty("kind", nil, "a", map[string]any{"b": 1}),
			wantErr:  `mapping property "kind": enum value map[b:1] is not a string, finite number, boolean or null`,
		},
		{
			name:     "array value",
			property: enumProperty("kind", nil, []any{1}),
			wantErr:  `mapping property "kind": enum value [1] is not a string, finite number, boolean or null`,
		},
		{
			name:     "NaN",
			property: enumProperty("kind", numberType, math.NaN()),
			wantErr:  `mapping property "kind": enum value NaN is not a string, finite number, boolean or null`,
		},
		{
			name:     "infinity",
			property: enumProperty("kind", numberType, math.Inf(1)),
			wantErr:  `mapping property "kind": enum value +Inf is not a string, finite number, boolean or null`,
		},
		{
			name:     "two values one float64 represents",
			property: enumProperty("kind", numberType, json.Number("9007199254740992"), json.Number("9007199254740993")),
			wantErr:  `mapping property "kind": enum values 9007199254740992 and 9007199254740993 are the same float64`,
		},
		{
			name:     "an int64 and a float64 of a mixed enum written alike",
			property: enumProperty("kind", nil, "a", json.Number("9007199254740992"), json.Number("9007199254740992.5")),
			wantErr:  `mapping property "kind": enum values 9007199254740992 and 9007199254740992.5 are the same float64`,
		},
		{
			name:     "integer beyond int64",
			property: enumProperty("kind", integerType, json.Number("9223372036854775808")),
			wantErr:  `mapping property "kind": enum value 9223372036854775808 does not fit in int64`,
		},
		{
			name:     "number beyond float64",
			property: enumProperty("kind", numberType, json.Number("1e400")),
			wantErr:  `mapping property "kind": enum value 1e400 does not fit in float64`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&Generator{}).Generate(trackPlan(tt.property), core.GenerateOptions{}, nil)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
