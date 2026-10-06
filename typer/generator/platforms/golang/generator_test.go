package golang_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	`skipping property "array_with_null_items" (array of null|string) of track event "User Signed Up": Go generation does not support array item types yet`,
	`skipping property "contacts" (array of custom:email) of track event "User Signed Up": Go generation does not support array item types yet`,
	`skipping property "context" (object) of track event "User Signed Up": Go generation does not support nested object schemas yet`,
	`skipping property "custom_null_field" (custom:null_type) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "device_type" (string) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "email_list" (custom:email_list) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "empty_object_no_additional_props" (custom:empty_object_no_additional_props) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "empty_object_with_additional_props" (custom:empty_object_with_additional_props) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "enabled" (boolean) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "feature_config" (custom:feature_config) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "mixed_value" (any) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "multi_type_array" (array of integer|string) of track event "User Signed Up": Go generation does not support array item types yet`,
	`skipping property "multi_type_field" (boolean|integer|string) of track event "User Signed Up": Go generation does not support multi-type properties yet`,
	`skipping property "multi_type_with_null" (integer|null|string) of track event "User Signed Up": Go generation does not support multi-type properties yet`,
	`skipping property "nested_empty_object" (object) of track event "User Signed Up": Go generation does not support nested object schemas yet`,
	`skipping property "nested_empty_object_no_additional_props" (object) of track event "User Signed Up": Go generation does not support nested object schemas yet`,
	`skipping property "null_field" (null) of track event "User Signed Up": Go generation does not support the null type yet`,
	`skipping property "number_or_null" (null|number) of track event "User Signed Up": Go generation does not support multi-type properties yet`,
	`skipping property "phone_numbers" (array of custom:phone_number) of track event "User Signed Up": Go generation does not support array item types yet`,
	`skipping property "priority" (integer) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "profile" (custom:user_profile) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "profile_list" (custom:profile_list) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "rating" (number) of track event "User Signed Up": Go generation does not support enums yet`,
	`skipping property "status" (custom:status) of track event "User Signed Up": Go generation does not support custom types yet`,
	`skipping property "string_or_null" (null|string) of track event "User Signed Up": Go generation does not support multi-type properties yet`,
	`skipping property "tags" (array of string) of track event "User Signed Up": Go generation does not support array item types yet`,
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

	for _, golden := range testutils.Goldens {
		t.Run(golden.Path, func(t *testing.T) {
			warnings := captureWarnings(t)

			files, err := (&golang.Generator{}).Generate(golden.Plan(), core.GenerateOptions{RudderCLIVersion: "1.0.0"}, golang.GoOptions{PackageName: golden.PackageName})
			require.NoError(t, err)
			require.Len(t, files, 1)
			assert.Equal(t, "ruddertyper.go", files[0].Path)
			assert.Regexp(t, regexp.MustCompile(`\A// Code generated .* DO NOT EDIT\.\n`), files[0].Content)

			want, err := os.ReadFile(golden.Path)
			require.NoError(t, err)
			if diff := cmp.Diff(string(want), files[0].Content); diff != "" {
				t.Errorf("generated content does not match %s (-want +got):\n%s\nRun 'make typer-go-update-testdata' to update the goldens.", golden.Path, diff)
			}
			assert.Equal(t, wantWarnings[golden.Path], *warnings)
		})
	}
}

func TestGenerateRejectsUnnamedTrackEvents(t *testing.T) {
	trackEvent := func(event string, properties map[string]plan.PropertySchema) *plan.TrackingPlan {
		return &plan.TrackingPlan{Rules: []plan.EventRule{{
			Event:   plan.Event{EventType: plan.EventTypeTrack, Name: event},
			Section: plan.IdentitySectionProperties,
			Schema:  plan.ObjectSchema{Properties: properties},
		}}}
	}
	stringProperty := func(name string) map[string]plan.PropertySchema {
		return map[string]plan.PropertySchema{name: {Property: plan.Property{Name: name, Types: []plan.PropertyType{plan.PrimitiveTypeString}}}}
	}

	tests := []struct {
		name    string
		plan    *plan.TrackingPlan
		wantErr string
	}{
		{"empty track event name", trackEvent("", nil), "a track event has an empty name"},
		{"symbols only", trackEvent("!!!", nil), `naming track event "!!!": name "!!!" has no letters or digits to build a Go identifier from`},
		{"emoji only", trackEvent("🎯", nil), `naming track event "🎯": name "🎯" has no letters or digits to build a Go identifier from`},
		{"property without words", trackEvent("Some Event", stringProperty("$$")), `naming property "$$": name "$$" has no letters or digits to build a Go identifier from`},
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
			"payload with an emitted field before one whose fields were all skipped",
			[]plan.EventRule{rule("A", map[string]plan.PropertySchema{"kind": enumProperty}), rule("B", map[string]plan.PropertySchema{"name": stringProperty})},
			"//\terr := rt.TrackB(\n",
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
