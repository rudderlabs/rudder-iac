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
	`skipping property "dollar_field" (string) of track event "$Variable$String": Go generation does not support enums yet`,
	`skipping property "email" (custom:email) of track event "$eventWithNameCamelCase$!": Go generation does not support custom types yet`,
	`ignoring the variants of track event "Event With Variants": Go generation does not support variants yet`,
	`skipping property "device_type" (string) of track event "Event With Variants": Go generation does not support enums yet`,
	`skipping property "page_context" (custom:page_context) of track event "Event With Variants": Go generation does not support custom types yet`,
	`skipping property "profile" (custom:user_profile) of track event "Event With Variants": Go generation does not support custom types yet`,
	`skipping property "status_code" (string) of track event "Product \"Premium\" Clicked": Go generation does not support enums yet`,
	`skipping property "active" (custom:active) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "addresses" (custom:address_list) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "age" (custom:age) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "contacts" (array of custom:email) of track event "User Signed Up": Go generation does not support arrays of custom types yet`,
	`skipping property "context" (object) of track event "User Signed Up": Go generation does not support nested object schemas yet`,
	`skipping property "custom_null_field" (custom:null_type) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "device_type" (string) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "email_list" (custom:email_list) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "empty_object_no_additional_props" (custom:empty_object_no_additional_props) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "empty_object_with_additional_props" (custom:empty_object_with_additional_props) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "enabled" (boolean) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "feature_config" (custom:feature_config) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "mixed_value" (any) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "nested_empty_object" (object) of track event "User Signed Up": Go generation does not support nested object schemas yet`,
	`skipping property "nested_empty_object_no_additional_props" (object) of track event "User Signed Up": Go generation does not support nested object schemas yet`,
	`skipping property "phone_numbers" (array of custom:phone_number) of track event "User Signed Up": Go generation does not support arrays of custom types yet`,
	`skipping property "priority" (integer) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "profile" (custom:user_profile) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "profile_list" (custom:profile_list) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "rating" (number) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "status" (custom:status) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "unicode_custom_type" (custom:типы_данных) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "unicode_enum_field" (string) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "user_access" (custom:user_access) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "active" (custom:active) of track event "eventWithNameCamelCase": Go generation does not support custom types yet`,
}

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
		stringSchema = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeString}}}
		enumSchema   = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeString}, Config: &plan.PropertyConfig{Enum: []any{"a"}}}}
		customSchema = plan.PropertySchema{Property: plan.Property{Types: []plan.PropertyType{plan.CustomType{Name: "!!!", Type: plan.PrimitiveTypeString}}}}
		nestedSchema = plan.PropertySchema{
			Property: plan.Property{Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
			Schema:   &plan.ObjectSchema{Properties: property("$$", stringSchema)},
		}
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
			trackEvent("Some Event", property("$$", enumSchema)),
			`naming property "$$" of track event "Some Event": name "$$" has no letters or digits to build a Go identifier from`,
		},
		{
			"custom type without words",
			trackEvent("Some Event", property("kind", customSchema)),
			`naming custom type "!!!" of property "kind": name "!!!" has no letters or digits to build a Go identifier from`,
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
		stringProperty = plan.PropertySchema{Property: plan.Property{Name: "name", Types: []plan.PropertyType{plan.PrimitiveTypeString}}}
		enumProperty   = plan.PropertySchema{Property: plan.Property{Name: "kind", Types: []plan.PropertyType{plan.PrimitiveTypeString}, Config: &plan.PropertyConfig{Enum: []any{"a"}}}}
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
			[]plan.EventRule{rule("A", map[string]plan.PropertySchema{"kind": enumProperty}), rule("B", map[string]plan.PropertySchema{"name": stringProperty})},
			"//\terr := rt.TrackA(\n",
		},
		{
			"payload that declares a property before one without properties",
			[]plan.EventRule{rule("A", nil), rule("B", map[string]plan.PropertySchema{"kind": enumProperty})},
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
				"kind": {Property: plan.Property{Name: "kind", Types: []plan.PropertyType{plan.PrimitiveTypeString}, Config: &plan.PropertyConfig{Enum: []any{"a"}}}},
			},
		},
	}}}

	files, err := (&golang.Generator{}).Generate(p, core.GenerateOptions{}, nil)
	require.NoError(t, err)
	require.Len(t, files, 1)

	assert.Contains(t, files[0].Content, "\tm := withAdditional(v.AdditionalProperties, \"kind\", \"name\")\n")
	assert.Equal(t, []string{`skipping property "kind" (string) of track event "Some Event": Go generation does not support enums yet`}, *warnings)
}

func captureWarnings(t *testing.T) *[]string {
	t.Helper()
	var warnings []string
	original := core.Warn
	core.Warn = func(msg string) { warnings = append(warnings, msg) }
	t.Cleanup(func() { core.Warn = original })
	return &warnings
}

// trackPlan is a plan with one track event, "Some Event", declaring properties.
func trackPlan(properties ...plan.Property) *plan.TrackingPlan {
	schema := plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}}
	for _, p := range properties {
		schema.Properties[p.Name] = plan.PropertySchema{Property: p}
	}
	return &plan.TrackingPlan{Rules: []plan.EventRule{{
		Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Some Event"},
		Section: plan.IdentitySectionProperties,
		Schema:  schema,
	}}}
}

// A custom type cannot be a union member, as in Kotlin and Swift, whether the
// union holds the property or its array items. With one non-null type it is a
// nullable custom type, or an array of them, left out until custom types are
// generated.
func TestGenerateCustomTypeInUnion(t *testing.T) {
	var (
		email = plan.CustomType{Name: "email", Type: plan.PrimitiveTypeString}
		array = []plan.PropertyType{plan.PrimitiveTypeArray}
	)

	tests := []struct {
		name         string
		types        []plan.PropertyType
		items        []plan.PropertyType
		wantErr      string
		wantWarnings []string
	}{
		{
			name:    "union member",
			types:   []plan.PropertyType{plan.PrimitiveTypeString, email},
			wantErr: `mapping property "contact": custom type "email" cannot be a member of a multi-type union`,
		},
		{
			name:    "union member next to null",
			types:   []plan.PropertyType{email, plan.PrimitiveTypeInteger, plan.PrimitiveTypeNull},
			wantErr: `mapping property "contact": custom type "email" cannot be a member of a multi-type union`,
		},
		{
			name:    "union member with custom item types",
			types:   []plan.PropertyType{plan.PrimitiveTypeArray, email},
			items:   []plan.PropertyType{email},
			wantErr: `mapping property "contact": custom type "email" cannot be a member of a multi-type union`,
		},
		{
			name:    "item union member",
			types:   array,
			items:   []plan.PropertyType{plan.PrimitiveTypeString, email},
			wantErr: `mapping property "contact": mapping the item types of PropertyContact: custom type "email" cannot be a member of a multi-type union`,
		},
		{
			name:         "nullable custom type",
			types:        []plan.PropertyType{email, plan.PrimitiveTypeNull},
			wantWarnings: []string{`skipping property "contact" (custom:email|null) of track event "Some Event": Go generation does not support custom types yet`},
		},
		{
			name:         "custom item type",
			types:        array,
			items:        []plan.PropertyType{email},
			wantWarnings: []string{`skipping property "contact" (array of custom:email) of track event "Some Event": Go generation does not support arrays of custom types yet`},
		},
		{
			name:         "nullable custom item type",
			types:        array,
			items:        []plan.PropertyType{email, plan.PrimitiveTypeNull},
			wantWarnings: []string{`skipping property "contact" (array of custom:email|null) of track event "Some Event": Go generation does not support arrays of custom types yet`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := captureWarnings(t)

			_, err := (&golang.Generator{}).Generate(trackPlan(plan.Property{Name: "contact", Types: tt.types, ItemTypes: tt.items}), core.GenerateOptions{}, nil)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantWarnings, *warnings)
		})
	}
}

// Each value helper is emitted only when the generated types use it, under a
// section header emitted only with one of them. trackPlan's fields are
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := (&golang.Generator{}).Generate(trackPlan(tt.properties...), core.GenerateOptions{}, nil)
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
