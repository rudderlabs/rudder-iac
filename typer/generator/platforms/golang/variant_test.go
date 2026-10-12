package golang

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// variantPlan is a plan with one track event, "Some Event", whose base schema
// holds the required discriminator kind.
func variantPlan(kind plan.Property, variants ...plan.Variant) *plan.TrackingPlan {
	kind.Name = "kind"
	p := trackPlan()
	p.Rules[0].Schema.Properties["kind"] = plan.PropertySchema{Property: kind, Required: true}
	p.Rules[0].Variants = variants
	return p
}

// discriminate is a variant on kind with a case per match list.
func discriminate(matches ...[]any) plan.Variant {
	v := plan.Variant{Type: "discriminator", Discriminator: "kind"}
	for _, m := range matches {
		v.Cases = append(v.Cases, plan.VariantCase{Match: m})
	}
	return v
}

func typed(types ...plan.PropertyType) plan.Property { return plan.Property{Types: types} }

func nullable(types ...plan.PropertyType) []plan.PropertyType {
	return append(types, plan.PrimitiveTypeNull)
}

// someEventVariant generates p and returns the variant interface of "Some
// Event" and the structs that implement it, in file order.
func someEventVariant(t *testing.T, p *plan.TrackingPlan) (*GoVariant, []*GoStruct) {
	t.Helper()
	ctx, err := newContext(p, "", "ruddertyper")
	require.NoError(t, err)

	var (
		variant *GoVariant
		structs []*GoStruct
	)
	for _, d := range ctx.Payloads {
		if d.Variant != nil {
			variant = d.Variant
		}
		if d.Struct != nil && d.Struct.Variant != nil {
			structs = append(structs, d.Struct)
		}
	}
	require.NotNil(t, variant)
	return variant, structs
}

func TestGenerateVariantPlanShapes(t *testing.T) {
	var (
		object = func(properties map[string]plan.PropertySchema) plan.PropertySchema {
			return plan.PropertySchema{Property: plan.Property{Name: "meta", Types: []plan.PropertyType{plan.PrimitiveTypeObject}}, Schema: &plan.ObjectSchema{Properties: properties}}
		}
		withCase = func(v plan.Variant, schema plan.ObjectSchema) plan.Variant {
			v.Cases[0].Schema = schema
			return v
		}
		redefine = func(p plan.Property) plan.ObjectSchema {
			p.Name = "kind"
			return plan.ObjectSchema{Properties: map[string]plan.PropertySchema{"kind": {Property: p}}}
		}
		nestedConflict = variantPlan(typed(onlyString...), withCase(discriminate([]any{"a"}), plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"meta": object(map[string]plan.PropertySchema{"b": {Property: plan.Property{Name: "b", Types: onlyString}}}),
		}}))
		defaultConflict = variantPlan(typed(onlyString...), discriminate([]any{"a"}))
		variantType     = plan.CustomType{Name: "meta", Type: plan.PrimitiveTypeObject, Variants: []plan.Variant{discriminate([]any{"a"})}, Schema: &plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"$$":   {Property: plan.Property{Name: "$$", Types: onlyString}},
				"kind": {Property: plan.Property{Name: "kind", Types: onlyString}, Required: true},
			},
		}}
	)
	nestedConflict.Rules[0].Schema.Properties["meta"] = object(map[string]plan.PropertySchema{"a": {Property: plan.Property{Name: "a", Types: onlyString}}})
	defaultConflict.Rules[0].Variants[0].DefaultSchema = &plan.ObjectSchema{Properties: redefine(typed(onlyInteger...)).Properties}

	tests := []struct {
		name         string
		plan         *plan.TrackingPlan
		wantErr      string
		wantWarnings []string
	}{
		{
			name:    "a case redefines a base property's type",
			plan:    variantPlan(typed(onlyString...), withCase(discriminate([]any{"a"}), redefine(typed(onlyInteger...)))),
			wantErr: `mapping the variants of track event "Some Event": case ["a"]: property "kind" conflicts with its definition in the base schema`,
		},
		{
			name:    "a case redefines a base property's enum",
			plan:    variantPlan(typed(onlyString...), withCase(discriminate([]any{"a"}), redefine(enumProperty("kind", onlyString, "a")))),
			wantErr: `mapping the variants of track event "Some Event": case ["a"]: property "kind" conflicts with its definition in the base schema`,
		},
		{
			name:    "a case redefines a base property with an invalid enum",
			plan:    variantPlan(typed(onlyString...), withCase(discriminate([]any{"a"}), redefine(plan.Property{Types: onlyString, Config: &plan.PropertyConfig{Enum: []any{}}}))),
			wantErr: `mapping the variants of track event "Some Event": case ["a"]: mapping property "kind": enum has no values`,
		},
		{
			name:    "a case redefines a base property's nested schema",
			plan:    nestedConflict,
			wantErr: `mapping the variants of track event "Some Event": case ["a"]: property "meta" conflicts with its definition in the base schema`,
		},
		{
			name:    "the default schema redefines a base property",
			plan:    defaultConflict,
			wantErr: `mapping the variants of track event "Some Event": the default schema: property "kind" conflicts with its definition in the base schema`,
		},
		{
			name:    "a match value in two cases",
			plan:    variantPlan(typed(onlyString...), discriminate([]any{"a"}, []any{"b", "a"})),
			wantErr: `mapping the variants of track event "Some Event": case ["b","a"]: match value "a" appears more than once`,
		},
		{
			name:    "a match value equal to another once normalized",
			plan:    variantPlan(typed(onlyInteger...), discriminate([]any{1, json.Number("1.0")})),
			wantErr: `mapping the variants of track event "Some Event": case [1,1.0]: match value 1 appears more than once`,
		},
		{
			name:    "a match value outside the declared types",
			plan:    variantPlan(typed(onlyString...), discriminate([]any{1})),
			wantErr: `mapping the variants of track event "Some Event": case [1]: match value 1 is not a value of discriminator "kind"`,
		},
		{
			name:    "a match value outside the enum",
			plan:    variantPlan(enumProperty("kind", onlyString, "a", "b"), discriminate([]any{"archived"})),
			wantErr: `mapping the variants of track event "Some Event": case ["archived"]: match value "archived" is not a value of discriminator "kind"`,
		},
		{
			name:    "a null match value",
			plan:    variantPlan(typed(plan.PrimitiveTypeString, plan.PrimitiveTypeNull), discriminate([]any{nil})),
			wantErr: `mapping the variants of track event "Some Event": case [null]: match value null is not a string, integer or boolean`,
		},
		{
			name:    "a number match value",
			plan:    variantPlan(typed(plan.PrimitiveTypeNumber, plan.PrimitiveTypeString), discriminate([]any{1.5})),
			wantErr: `mapping the variants of track event "Some Event": case [1.5]: match value 1.5 is not a string, integer or boolean`,
		},
		{
			name:    "an integral match value outside int64",
			plan:    variantPlan(typed(onlyInteger...), discriminate([]any{json.Number("1e20")})),
			wantErr: `mapping the variants of track event "Some Event": case [1e20]: match value 1e20 does not fit in int64`,
		},
		{
			name:    "a discriminator typed only number",
			plan:    variantPlan(typed(onlyNumber...), discriminate([]any{1})),
			wantErr: `mapping the variants of track event "Some Event": discriminator "kind" (number) allows none of string, integer and boolean`,
		},
		{
			name:    "a discriminator without types",
			plan:    variantPlan(enumProperty("kind", nil, "a"), discriminate([]any{"a"})),
			wantErr: `mapping the variants of track event "Some Event": discriminator "kind" (any) allows none of string, integer and boolean`,
		},
		{
			name:    "a discriminator of an object custom type",
			plan:    variantPlan(typed(plan.CustomType{Name: "meta", Type: plan.PrimitiveTypeObject}), discriminate([]any{"a"})),
			wantErr: `mapping the variants of track event "Some Event": discriminator "kind" (custom:meta) allows none of string, integer and boolean`,
		},
		{
			name:    "a discriminator the base schema does not declare",
			plan:    variantPlan(typed(plan.PrimitiveTypeBoolean), plan.Variant{Discriminator: "other"}),
			wantErr: `mapping the variants of track event "Some Event": discriminator "other" is not a property of the base schema`,
		},
		{
			name:    "two variants entries",
			plan:    variantPlan(typed(plan.PrimitiveTypeBoolean), discriminate([]any{true}), discriminate([]any{false})),
			wantErr: `mapping the variants of track event "Some Event": found 2 variants entries; a rule or custom type may have one`,
		},
		{
			name: "a case property without words",
			plan: variantPlan(typed(onlyString...), withCase(discriminate([]any{"a"}), plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"$$": {Property: plan.Property{Name: "$$", Types: onlyString}},
			}})),
			wantErr: `naming property "$$" of case ["a"] of the variants of track event "Some Event": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			name:    "a variant custom type's base property without words",
			plan:    trackPlan(plan.Property{Name: "meta", Types: []plan.PropertyType{variantType}}),
			wantErr: `naming property "$$" of custom type "meta": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			name:         "an open case under a closed base",
			plan:         variantPlan(typed(onlyString...), withCase(discriminate([]any{"a"}), plan.ObjectSchema{AdditionalProperties: true})),
			wantWarnings: []string{`generating case ["a"] of the variants of track event "Some Event" closed: its base schema does not allow additional properties`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				warnings []string
				warn     = core.Warn
			)
			core.Warn = func(msg string) { warnings = append(warnings, msg) }
			t.Cleanup(func() { core.Warn = warn })

			files, err := (&Generator{}).Generate(tt.plan, core.GenerateOptions{}, nil)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantWarnings, warnings)
			// Every base here is closed, so no branch may be open.
			assert.NotContains(t, files[0].Content, "AdditionalProperties")
		})
	}
}

// Default is left out only when the cases match every value the
// discriminator can take, which a nullable one never allows. Its
// discriminator's note names the values it stands for.
func TestGenerateVariantDefault(t *testing.T) {
	tests := []struct {
		name    string
		kind    plan.Property
		matches [][]any
		want    []string
		note    string
	}{
		{"string", typed(onlyString...), [][]any{{"a"}, {"b"}}, []string{"CaseA", "CaseB", "Default"}, `not "a" or "b"`},
		{"nullable string", typed(nullable(plan.PrimitiveTypeString)...), [][]any{{"a"}}, []string{"CaseA", "Default"}, `not "a", or null`},
		{"boolean with both values matched", typed(plan.PrimitiveTypeBoolean), [][]any{{true}, {false}}, []string{"CaseTrue", "CaseFalse"}, ""},
		{"boolean with one value matched", typed(plan.PrimitiveTypeBoolean), [][]any{{true}}, []string{"CaseTrue", "Default"}, "not true"},
		{
			"nullable boolean with both values matched",
			typed(nullable(plan.PrimitiveTypeBoolean)...),
			[][]any{{true, false}},
			[]string{"CaseTrue", "CaseFalse", "Default"},
			"only null",
		},
		{
			"boolean custom type with both values matched",
			typed(plan.CustomType{Name: "flag", Type: plan.PrimitiveTypeBoolean}),
			[][]any{{false}, {true}},
			[]string{"CaseFalse", "CaseTrue"},
			"",
		},
		{"enum with every value matched", enumProperty("kind", onlyString, "a", "b"), [][]any{{"b"}, {"a"}}, []string{"CaseB", "CaseA"}, ""},
		{"enum with a value left", enumProperty("kind", onlyString, "a", "b"), [][]any{{"a"}}, []string{"CaseA", "Default"}, "only PropertyKindB"},
		{"integer enum matched as written otherwise", enumProperty("kind", onlyInteger, 1, 2), [][]any{{json.Number("2.0"), 1.0}}, []string{"Case2", "Case1"}, ""},
		{
			"enum custom type with every value matched",
			typed(plan.CustomType{Name: "status", Type: plan.PrimitiveTypeString, Config: &plan.PropertyConfig{Enum: []any{"on", "off"}}}),
			[][]any{{"on"}, {"off"}},
			[]string{"CaseOn", "CaseOff"},
			"",
		},
		{
			"nullable enum with every other value matched",
			enumProperty("kind", nullable(plan.PrimitiveTypeString), "a", nil),
			[][]any{{"a"}},
			[]string{"CaseA", "Default"},
			"only null",
		},
		{
			"boolean and string union with both booleans matched",
			typed(plan.PrimitiveTypeBoolean, plan.PrimitiveTypeString),
			[][]any{{true}, {false}},
			[]string{"CaseTrue", "CaseFalse", "Default"},
			"not false or true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variant, structs := someEventVariant(t, variantPlan(tt.kind, discriminate(tt.matches...)))
			var (
				got  = make([]string, len(variant.Implementations))
				note string
			)
			for i, name := range variant.Implementations {
				got[i] = strings.TrimPrefix(name, "TrackSomeEventProperties")
			}
			for _, st := range structs {
				if st.Name == "TrackSomeEventPropertiesDefault" {
					note = st.Fields[0].Note // the discriminator, its only field
				}
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.note, note)
		})
	}
}

// An optional discriminator adds Absent and makes every Discriminator return
// ok as well; a case then writes its value in toMap itself.
func TestGenerateVariantOptionalDiscriminator(t *testing.T) {
	p := variantPlan(typed(onlyString...), discriminate([]any{"a"}))
	kind := p.Rules[0].Schema.Properties["kind"]
	kind.Required = false
	p.Rules[0].Schema.Properties["kind"] = kind

	variant, structs := someEventVariant(t, p)
	assert.Equal(t, &GoVariant{
		Name: "TrackSomeEventProperties",
		Doc: `TrackSomeEventProperties holds the properties of "Some Event", a discriminated union on the optional "kind". ` +
			"It is implemented by TrackSomeEventPropertiesAbsent, TrackSomeEventPropertiesCaseA and TrackSomeEventPropertiesDefault.",
		Key:              "kind",
		Result:           "(value PropertyKind, ok bool)",
		DiscriminatorDoc: `Discriminator returns the value of "kind"; ok is false when it is absent.`,
		Payload:          true,
		Implementations:  []string{"TrackSomeEventPropertiesAbsent", "TrackSomeEventPropertiesCaseA", "TrackSomeEventPropertiesDefault"},
	}, variant)
	assert.Equal(t, []*GoStruct{
		{
			Name:             "TrackSomeEventPropertiesAbsent",
			Doc:              `TrackSomeEventPropertiesAbsent is the case where "kind" is absent. It holds the base properties only.`,
			Payload:          true,
			DeclaredKeys:     []string{"kind"},
			Variant:          variant,
			DiscriminatorDoc: `Discriminator reports that "kind" is absent.`,
			Returns:          "value, false",
		},
		{
			Name:             "TrackSomeEventPropertiesCaseA",
			Doc:              `TrackSomeEventPropertiesCaseA is the case where kind is "a".`,
			Payload:          true,
			DeclaredKeys:     []string{"kind"},
			Variant:          variant,
			DiscriminatorDoc: `Discriminator returns "a".`,
			Returns:          `"a", true`,
			Sends:            `"a"`,
		},
		{
			Name:             "TrackSomeEventPropertiesDefault",
			Doc:              `TrackSomeEventPropertiesDefault is the default case of TrackSomeEventProperties, for any kind other than "a".`,
			Payload:          true,
			DeclaredKeys:     []string{"kind"},
			Fields:           []GoField{{Name: "Kind", Key: "kind", Type: "PropertyKind", Required: true, Note: `not "a"`}},
			Variant:          variant,
			DiscriminatorDoc: "Discriminator returns Kind.",
			Returns:          "v.Kind, true",
			ReadsReceiver:    true,
		},
	}, structs)
}

// One row per row of the spec's Discriminator Values table: what a case's
// Discriminator returns for each resolved discriminator type.
func TestGenerateVariantDiscriminatorValues(t *testing.T) {
	status := plan.CustomType{Name: "status", Type: plan.PrimitiveTypeString, Config: &plan.PropertyConfig{Enum: []any{"a", "b"}}}

	tests := []struct {
		name  string
		kind  plan.Property
		match any
		want  string
	}{
		{"alias of string", typed(onlyString...), "a", `"a"`},
		{"alias of int64", typed(onlyInteger...), 1, "1"},
		{"alias of bool", typed(plan.PrimitiveTypeBoolean), true, "true"},
		{"primitive custom type", typed(plan.CustomType{Name: "code", Type: plan.PrimitiveTypeInteger}), -2, "-2"},
		{"enum", enumProperty("kind", onlyString, "a", "b"), "a", "PropertyKindA"},
		{"enum custom type", typed(status), "b", "CustomTypeStatusB"},
		{"mixed enum", enumProperty("kind", []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger}, "a", 1), 1, "PropertyKind1()"},
		{"union, string member", typed(plan.PrimitiveTypeBoolean, plan.PrimitiveTypeString), "beta", `NewPropertyKindString("beta")`},
		{"union, boolean member", typed(plan.PrimitiveTypeBoolean, plan.PrimitiveTypeString), true, "NewPropertyKindBoolean(true)"},
		{"union, integral match with an Integer member", typed(plan.PrimitiveTypeInteger, plan.PrimitiveTypeNumber, plan.PrimitiveTypeString), 1, "NewPropertyKindInteger(1)"},
		{"union, integral match without an Integer member", typed(plan.PrimitiveTypeNumber, plan.PrimitiveTypeString), 1, "NewPropertyKindNumber(1)"},
		{"Nullable of string", typed(nullable(plan.PrimitiveTypeString)...), "a", `NewNullable[string]("a")`},
		{"Nullable of int64", typed(nullable(plan.PrimitiveTypeInteger)...), 1, "NewNullable[int64](1)"},
		{"Nullable of an enum", enumProperty("kind", nullable(plan.PrimitiveTypeString), "a", nil), "a", "NewNullable[PropertyKindValue](PropertyKindValueA)"},
		{"Nullable of an enum custom type", typed(nullable(status)...), "a", "NewNullable[CustomTypeStatus](CustomTypeStatusA)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, structs := someEventVariant(t, variantPlan(tt.kind, discriminate([]any{tt.match})))
			assert.Equal(t, tt.want, structs[0].Returns)
		})
	}
}
