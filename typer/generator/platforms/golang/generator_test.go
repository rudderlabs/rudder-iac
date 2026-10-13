package golang_test

import (
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang/testutils"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The reference plan uses constructs the generator does not support yet; each
// one it leaves out must be reported.
var referenceWarnings = []string{
	`skipping the group rule (section "context.traits"): Go generation does not support group events yet`,
	`skipping the identify rule (section "traits"): Go generation does not support identify events yet`,
	`skipping the page rule (section "properties"): Go generation does not support page events yet`,
	`skipping the screen rule (section "properties"): Go generation does not support screen events yet`,
	`ignoring the variants of track event "Event With Variants": Go generation does not support variants yet`,
	`skipping property "page_context" (custom:page_context) of track event "Event With Variants": Go generation does not support variants yet`,
	`skipping property "feature_config" (custom:feature_config) of track event "User Signed Up": Go generation does not support variants yet`,
	`skipping property "user_access" (custom:user_access) of track event "User Signed Up": Go generation does not support variants yet`,
}

// variantType is a custom type the generator leaves out until it supports
// variants, and every field that needs it with it.
var variantType = plan.CustomType{Name: "kind", Type: plan.PrimitiveTypeObject, Variants: []plan.Variant{{Discriminator: "type"}}}

func TestGenerateGoldens(t *testing.T) {
	wantWarnings := map[string][]string{
		"testdata/validator/ruddertyper/ruddertyper.go": referenceWarnings,
		"testdata/validator/examples/ruddertyper.go":    nil,
	}
	runtimeFile, err := parser.ParseFile(token.NewFileSet(), "internal/runtime/runtime.go", nil, parser.ImportsOnly)
	require.NoError(t, err)

	for _, golden := range testutils.Goldens {
		t.Run(golden.Path, func(t *testing.T) {
			warnings := captureWarnings(t)

			files, err := (&golang.Generator{}).Generate(golden.Plan(), core.GenerateOptions{RudderCLIVersion: "1.0.0"}, golden.Options)
			require.NoError(t, err)
			require.Len(t, files, 1)
			assert.Equal(t, "ruddertyper.go", files[0].Path)
			assert.Regexp(t, regexp.MustCompile(`\A// Code generated .* DO NOT EDIT\.\n`), files[0].Content)

			want, err := os.ReadFile(golden.Path)
			require.NoError(t, err)
			assert.Equal(t, string(want), files[0].Content, "generated content does not match %s; run 'make typer-go-update-testdata' to update the goldens", golden.Path)
			assert.Equal(t, wantWarnings[golden.Path], *warnings)
			// The runtime package's tests cover the generated runtime only
			// while it is emitted unchanged.
			assert.Contains(t, files[0].Content, golang.RuntimeSource)
			// `make test` does not compile the goldens (only `make typer-go-validate`
			// does), so this catches a runtime import the template does not declare.
			for _, imp := range runtimeFile.Imports {
				assert.Contains(t, files[0].Content, imp.Path.Value)
			}
		})
	}
}

// Names are checked before the skip check, so rules and fields the generator
// leaves out for now fail on them too.
func TestGenerateRejectsUnnameableIdentifiers(t *testing.T) {
	rule := func(eventType plan.EventType, event string, section plan.IdentitySection, properties map[string]plan.PropertySchema) *plan.TrackingPlan {
		return &plan.TrackingPlan{Rules: []plan.EventRule{{
			Event:   plan.Event{EventType: eventType, Name: event},
			Section: section,
			Schema:  plan.ObjectSchema{Properties: properties},
		}}}
	}
	trackEvent := func(event string, properties map[string]plan.PropertySchema) *plan.TrackingPlan {
		return rule(plan.EventTypeTrack, event, plan.IdentitySectionProperties, properties)
	}
	property := func(name string, p plan.PropertySchema) map[string]plan.PropertySchema {
		p.Property.Name = name
		return map[string]plan.PropertySchema{name: p}
	}
	var (
		stringSchema  = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeString}}}
		skippedSchema = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{variantType}}}
		customSchema  = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{plan.CustomType{Name: "!!!", Type: plan.PrimitiveTypeString}}}}
		nestedSchema  = plan.PropertySchema{
			Property: plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
			Schema:   &plan.ObjectSchema{Properties: property("$$", stringSchema)},
		}
		objectSchema = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{plan.CustomType{
			Name:   "profile",
			Type:   plan.PrimitiveTypeObject,
			Schema: &plan.ObjectSchema{Properties: property("$$", skippedSchema)},
		}}}}
	)

	tests := []struct {
		name    string
		plan    *plan.TrackingPlan
		wantErr string
	}{
		{"empty track event name", trackEvent("", nil), "a track event has an empty name"},
		{"symbols only", trackEvent("!!!", nil), `naming track event "!!!": name "!!!" has no letters or digits to build a Go identifier from`},
		{"emoji only", trackEvent("🎯", nil), `naming track event "🎯": name "🎯" has no letters or digits to build a Go identifier from`},
		{
			"track event in a skipped section",
			rule(plan.EventTypeTrack, "!!!", plan.IdentitySectionTraits, nil),
			`naming track event "!!!": name "!!!" has no letters or digits to build a Go identifier from`,
		},
		{
			"empty track event in a skipped section",
			rule(plan.EventTypeTrack, "", plan.IdentitySectionTraits, nil),
			"a track event has an empty name",
		},
		{
			"skipped event type",
			rule(plan.EventTypePage, "🎯", plan.IdentitySectionProperties, nil),
			`naming page event "🎯": name "🎯" has no letters or digits to build a Go identifier from`,
		},
		{
			"property without words",
			trackEvent("Some Event", property("$$", stringSchema)),
			`naming property "$$" of track event "Some Event": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			"skipped property without words",
			trackEvent("Some Event", property("$$", skippedSchema)),
			`naming property "$$" of track event "Some Event": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			"custom type without words",
			trackEvent("Some Event", property("kind", customSchema)),
			`naming custom type "!!!" of property "kind": name "!!!" has no letters or digits to build a Go identifier from`,
		},
		{
			"custom type property without words",
			trackEvent("Some Event", property("kind", objectSchema)),
			`naming property "$$" of custom type "profile": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			"custom type property without words in a skipped rule",
			rule(plan.EventTypeIdentify, "", plan.IdentitySectionTraits, property("kind", objectSchema)),
			`naming property "$$" of custom type "profile": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			"nested property without words",
			trackEvent("Some Event", property("context", nestedSchema)),
			`naming property "$$" of track event "Some Event": name "$$" has no letters or digits to build a Go identifier from`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&golang.Generator{}).Generate(tt.plan, core.GenerateOptions{}, nil)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestGenerateQuickStart(t *testing.T) {
	var (
		stringProperty  = plan.PropertySchema{Property: plan.Property{Name: "name", Types: []plan.PropertyType{plan.PrimitiveTypeString}}}
		skippedProperty = plan.PropertySchema{Property: plan.Property{Name: "kind", Types: []plan.PropertyType{variantType}}}
	)
	rule := func(event string, properties map[string]plan.PropertySchema) plan.EventRule {
		return plan.EventRule{
			Event:   plan.Event{EventType: plan.EventTypeTrack, Name: event},
			Section: plan.IdentitySectionProperties,
			Schema:  plan.ObjectSchema{Properties: properties},
		}
	}

	tests := []struct {
		name  string
		rules []plan.EventRule
		want  string
	}{
		{
			"payload that declares only skipped properties before one with an emitted field",
			[]plan.EventRule{rule("A", map[string]plan.PropertySchema{"kind": skippedProperty}), rule("B", map[string]plan.PropertySchema{"name": stringProperty})},
			"//\terr := rt.TrackA(\n",
		},
		{
			"payload that declares a property before one without properties",
			[]plan.EventRule{rule("A", nil), rule("B", map[string]plan.PropertySchema{"kind": skippedProperty})},
			"//\terr := rt.TrackB(\n",
		},
		{
			"first method when no payload declares a property",
			[]plan.EventRule{rule("A", nil), rule("B", nil)},
			"//\terr := rt.TrackA(\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureWarnings(t)

			files, err := (&golang.Generator{}).Generate(&plan.TrackingPlan{Rules: tt.rules}, core.GenerateOptions{}, nil)
			require.NoError(t, err)
			require.Len(t, files, 1)

			assert.Contains(t, files[0].Content, tt.want)
		})
	}
}

// A skipped property is still declared, so AdditionalProperties cannot supply it.
func TestGenerateReservesSkippedDeclaredKeys(t *testing.T) {
	warnings := captureWarnings(t)
	p := &plan.TrackingPlan{Rules: []plan.EventRule{{
		Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Some Event"},
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			AdditionalProperties: true,
			Properties: map[string]plan.PropertySchema{
				"name": {Property: plan.Property{Name: "name", Types: []plan.PropertyType{plan.PrimitiveTypeString}}},
				"kind": {Property: plan.Property{Name: "kind", Types: []plan.PropertyType{variantType}}},
			},
		},
	}}}

	files, err := (&golang.Generator{}).Generate(p, core.GenerateOptions{}, nil)
	require.NoError(t, err)
	require.Len(t, files, 1)

	assert.Contains(t, files[0].Content, "\tm := withAdditional(v.AdditionalProperties, \"kind\", \"name\")\n")
	assert.Equal(t, []string{`skipping property "kind" (custom:kind) of track event "Some Event": Go generation does not support variants yet`}, *warnings)
}

// The skip check applies at every level, and the warning names where the
// field lives. An array custom type needs what its items need.
func TestGenerateSkipsUnsupportedFieldsAtEveryLevel(t *testing.T) {
	warnings := captureWarnings(t)
	var (
		kind   = plan.PropertySchema{Property: plan.Property{Name: "kind", Types: []plan.PropertyType{variantType}}}
		object = func(name string, properties map[string]plan.PropertySchema) plan.PropertySchema {
			return plan.PropertySchema{
				Property: plan.Property{Name: name, Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
				Schema:   &plan.ObjectSchema{Properties: properties},
			}
		}
		profile = plan.CustomType{Name: "profile", Type: plan.PrimitiveTypeObject, Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{"kind": kind}}}
		kinds   = plan.CustomType{Name: "kinds", Type: plan.PrimitiveTypeArray, ItemType: variantType}
	)
	p := golang.TrackPlan(
		plan.Property{Name: "profile", Types: []plan.PropertyType{profile}},
		plan.Property{Name: "kinds", Types: []plan.PropertyType{kinds}},
	)
	p.Rules[0].Schema.Properties["context"] = object("context", map[string]plan.PropertySchema{
		"inner": object("inner", map[string]plan.PropertySchema{"kind": kind}),
	})

	_, err := (&golang.Generator{}).Generate(p, core.GenerateOptions{}, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{
		`skipping property "context.inner.kind" (custom:kind) of track event "Some Event": Go generation does not support variants yet`,
		`skipping property "kinds" (custom:kinds) of track event "Some Event": Go generation does not support variants yet`,
		`skipping property "kind" (custom:kind) of custom type "profile": Go generation does not support variants yet`,
	}, *warnings)
}

func captureWarnings(t *testing.T) *[]string {
	t.Helper()
	var warnings []string
	original := core.Warn
	core.Warn = func(msg string) { warnings = append(warnings, msg) }
	t.Cleanup(func() { core.Warn = original })
	return &warnings
}

// A custom type cannot be a union member, as in Kotlin and Swift, whether the
// union holds the property or its array items, and even when an enum beats
// the union, since the enum would drop the values only the custom type
// allows. With one non-null type it is a nullable custom type, or an array of
// them.
func TestGenerateCustomTypeInUnion(t *testing.T) {
	var (
		email   = plan.CustomType{Name: "email", Type: plan.PrimitiveTypeString}
		array   = []plan.PropertyType{plan.PrimitiveTypeArray}
		wantErr = `mapping property "contact" of track event "Some Event": custom type "email" cannot be a member of a multi-type union`
	)

	tests := []struct {
		name    string
		types   []plan.PropertyType
		items   []plan.PropertyType
		config  *plan.PropertyConfig
		wantErr string
		want    string
	}{
		{name: "union member", types: []plan.PropertyType{plan.PrimitiveTypeString, email}, wantErr: wantErr},
		{name: "union member next to null", types: []plan.PropertyType{email, plan.PrimitiveTypeInteger, plan.PrimitiveTypeNull}, wantErr: wantErr},
		{name: "union member with custom item types", types: []plan.PropertyType{plan.PrimitiveTypeArray, email}, items: []plan.PropertyType{email}, wantErr: wantErr},
		{name: "item union member", types: array, items: []plan.PropertyType{plan.PrimitiveTypeString, email}, wantErr: wantErr},
		{
			name:    "union member with an enum",
			types:   []plan.PropertyType{plan.PrimitiveTypeInteger, email},
			config:  &plan.PropertyConfig{Enum: []any{"x@y", 1}},
			wantErr: wantErr,
		},
		{name: "nullable custom type", types: []plan.PropertyType{email, plan.PrimitiveTypeNull}, want: "type PropertyContact = Nullable[CustomTypeEmail]\n"},
		{name: "custom item type", types: array, items: []plan.PropertyType{email}, want: "type PropertyContact = []CustomTypeEmail\n"},
		{name: "nullable custom item type", types: array, items: []plan.PropertyType{email, plan.PrimitiveTypeNull}, want: "type PropertyContact = []Nullable[CustomTypeEmail]\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := (&golang.Generator{}).Generate(golang.TrackPlan(plan.Property{Name: "contact", Types: tt.types, ItemTypes: tt.items, Config: tt.config}), core.GenerateOptions{}, nil)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Len(t, files, 1)
			assert.Contains(t, files[0].Content, tt.want)
		})
	}
}

// A custom type that refers to itself, through any path, fails generation
// with an error naming the cycle, found by the first reference in rule-key and
// wire-key order. Cycles are checked before the skip check, so one that only
// a rule or field left out for now reaches fails too.
func TestGenerateRejectsRecursiveCustomTypes(t *testing.T) {
	object := func(name string) *plan.CustomType {
		return &plan.CustomType{Name: name, Type: plan.PrimitiveTypeObject, Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}}}
	}
	refer := func(from *plan.CustomType, key string, ps plan.PropertySchema) {
		ps.Property.Name = key
		from.Schema.Properties[key] = ps
	}
	typed := func(types ...plan.PropertyType) plan.PropertySchema {
		return plan.PropertySchema{Property: plan.Property{Types: types}}
	}
	schema := func(key string, ps plan.PropertySchema) *plan.ObjectSchema {
		ps.Property.Name = key
		return &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{key: ps}}
	}

	tests := []struct {
		name    string
		plan    func() *plan.TrackingPlan
		wantErr string
	}{
		{
			name: "directly",
			plan: func() *plan.TrackingPlan {
				a := object("a")
				refer(a, "self", typed(a, plan.PrimitiveTypeNull))
				return golang.TrackPlan(plan.Property{Name: "value", Types: []plan.PropertyType{a}})
			},
			wantErr: `custom type "a" refers to itself: a → a`,
		},
		{
			name: "through other custom types",
			plan: func() *plan.TrackingPlan {
				a, b, c := object("a"), object("b"), object("c")
				refer(a, "b", typed(b))
				refer(b, "c", typed(c))
				refer(c, "b", typed(b))
				return golang.TrackPlan(plan.Property{Name: "value", Types: []plan.PropertyType{a}})
			},
			wantErr: `custom type "b" refers to itself: b → c → b`,
		},
		{
			name: "through an array custom type",
			plan: func() *plan.TrackingPlan {
				a := object("a")
				list := &plan.CustomType{Name: "list", Type: plan.PrimitiveTypeArray, ItemType: a}
				refer(a, "children", typed(list))
				return golang.TrackPlan(plan.Property{Name: "value", Types: []plan.PropertyType{list}})
			},
			wantErr: `custom type "list" refers to itself: list → a → list`,
		},
		{
			name: "through array items in a nested schema",
			plan: func() *plan.TrackingPlan {
				a := object("a")
				nested := typed(plan.PrimitiveTypeObject)
				nested.Schema = &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}}
				refer(a, "nested", nested)
				items := typed(plan.PrimitiveTypeArray)
				items.Property.Name, items.Property.ItemTypes = "items", []plan.PropertyType{a}
				nested.Schema.Properties["items"] = items
				return golang.TrackPlan(plan.Property{Name: "value", Types: []plan.PropertyType{a}})
			},
			wantErr: `custom type "a" refers to itself: a → a`,
		},
		{
			name: "only from an identify rule",
			plan: func() *plan.TrackingPlan {
				a, b := object("a"), object("b")
				refer(a, "b", typed(b))
				refer(b, "a", typed(a))
				p := golang.TrackPlan(plan.Property{Name: "value", Types: []plan.PropertyType{a}})
				p.Rules[0].Event, p.Rules[0].Section = plan.Event{EventType: plan.EventTypeIdentify}, plan.IdentitySectionTraits
				return p
			},
			wantErr: `custom type "a" refers to itself: a → b → a`,
		},
		{
			name: "through a variant custom type",
			plan: func() *plan.TrackingPlan {
				a, kind := object("a"), object("kind")
				kind.Variants = []plan.Variant{{Discriminator: "type", Cases: []plan.VariantCase{{Match: []any{"x"}, Schema: *schema("a", typed(a))}}}}
				refer(a, "kind", typed(kind))
				return golang.TrackPlan(plan.Property{Name: "value", Types: []plan.PropertyType{kind}})
			},
			wantErr: `custom type "kind" refers to itself: kind → a → kind`,
		},
		{
			name: "only from a rule's variant",
			plan: func() *plan.TrackingPlan {
				a := object("a")
				refer(a, "self", typed(a))
				p := golang.TrackPlan()
				p.Rules[0].Variants = []plan.Variant{{Discriminator: "type", DefaultSchema: schema("value", typed(a))}}
				return p
			},
			wantErr: `custom type "a" refers to itself: a → a`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := (&golang.Generator{}).Generate(tt.plan(), core.GenerateOptions{}, nil)
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}

// Each value helper is emitted only when the generated types use it, under a
// section header emitted only with one of them. TrackPlan's fields are
// optional, so Ptr comes with every type that cannot hold nil. Null also comes
// with an untyped property, which takes Null{} to send null.
func TestGenerateValueHelpers(t *testing.T) {
	type helpers struct{ header, ptr, null, nullable bool }
	property := func(name string, types []plan.PropertyType, items ...plan.PropertyType) plan.Property {
		return plan.Property{Name: name, Types: types, ItemTypes: items}
	}
	var (
		array    = []plan.PropertyType{plan.PrimitiveTypeArray}
		nullable = []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNull}
	)

	tests := []struct {
		name       string
		properties []plan.Property
		want       helpers
	}{
		{
			name:       "only types that hold nil",
			properties: []plan.Property{property("object", []plan.PropertyType{plan.PrimitiveTypeObject})},
		},
		{
			name: "no null, untyped or nullable property",
			properties: []plan.Property{
				property("string", []plan.PropertyType{plan.PrimitiveTypeString}),
				property("object", []plan.PropertyType{plan.PrimitiveTypeObject}),
				property("untyped_items", array),
				property("union", []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger}),
				// A union's null member is its zero value, not Null.
				property("union_with_null", []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeInteger, plan.PrimitiveTypeNull}),
				property("item_union", array, plan.PrimitiveTypeString, plan.PrimitiveTypeInteger),
			},
			want: helpers{header: true, ptr: true},
		},
		{
			name:       "null property",
			properties: []plan.Property{property("null", []plan.PropertyType{plan.PrimitiveTypeNull})},
			want:       helpers{header: true, ptr: true, null: true},
		},
		{
			name:       "untyped property",
			properties: []plan.Property{property("any", nil)},
			want:       helpers{header: true, null: true},
		},
		{
			name:       "null items",
			properties: []plan.Property{property("nulls", array, plan.PrimitiveTypeNull)},
			want:       helpers{header: true, null: true},
		},
		{
			name:       "nullable property",
			properties: []plan.Property{property("nullable", nullable)},
			want:       helpers{header: true, ptr: true, nullable: true},
		},
		{
			name:       "nullable items",
			properties: []plan.Property{property("nullable_items", array, nullable...)},
			want:       helpers{header: true, nullable: true},
		},
		{
			name:       "enum whose only value is null",
			properties: []plan.Property{{Name: "null_enum", Types: nullable, Config: &plan.PropertyConfig{Enum: []any{nil}}}},
			want:       helpers{header: true, ptr: true, null: true},
		},
		{
			name:       "nullable enum",
			properties: []plan.Property{{Name: "nullable_enum", Types: nullable, Config: &plan.PropertyConfig{Enum: []any{"a", nil}}}},
			want:       helpers{header: true, ptr: true, nullable: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := (&golang.Generator{}).Generate(golang.TrackPlan(tt.properties...), core.GenerateOptions{}, nil)
			require.NoError(t, err)
			require.Len(t, files, 1)

			content := files[0].Content
			assert.Equal(t, tt.want, helpers{
				header:   strings.Contains(content, "\n// --- Value helpers ---\n"),
				ptr:      strings.Contains(content, "\nfunc Ptr[T any](v T) *T { return &v }\n"),
				null:     strings.Contains(content, "\ntype Null struct{}\n"),
				nullable: strings.Contains(content, "\ntype Nullable[T any] struct {\n"),
			})
		})
	}
}
