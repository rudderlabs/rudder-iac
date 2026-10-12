package golang_test

import (
	"regexp"
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/generator/platforms/golang"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// payloadAndMethods matches the declarations of the payload types and of the
// event methods, without their bodies.
var payloadAndMethods = regexp.MustCompile(`(?m)^(type \w+(Properties|Traits)\d* .*?|func \(r \*RudderTyperAnalytics\) [A-Z].*?)( \{)?$`)

// One row per event type, section and schema shape of the spec's Event Types
// table: the payload type and method each generates, or the warning it is
// skipped with.
func TestGenerateEventTypes(t *testing.T) {
	var (
		// rule has an empty, closed schema.
		rule = func(eventType plan.EventType, name string, section plan.IdentitySection) plan.EventRule {
			return plan.EventRule{Event: plan.Event{EventType: eventType, Name: name}, Section: section}
		}
		required = func(name string) map[string]plan.PropertySchema {
			return map[string]plan.PropertySchema{name: {Property: plan.Property{Name: name, Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true}}
		}
		declared = func(r plan.EventRule) []plan.EventRule {
			r.Schema.Properties = required("title")
			return []plan.EventRule{r}
		}
		variant = func(r plan.EventRule) []plan.EventRule {
			r.Schema.Properties = required("kind")
			r.Variants = []plan.Variant{{Type: "discriminator", Discriminator: "kind", Cases: []plan.VariantCase{{Match: []any{"a"}}}}}
			return []plan.EventRule{r}
		}
		open = func(r plan.EventRule) []plan.EventRule {
			r.Schema.AdditionalProperties = true
			return []plan.EventRule{r}
		}
		closed = func(r plan.EventRule) []plan.EventRule { return []plan.EventRule{r} }
		method = func(signature string) string { return "func (r *RudderTyperAnalytics) " + signature + " error" }
	)

	tests := []struct {
		name         string
		rules        []plan.EventRule
		want         []string
		wantWarnings []string
	}{
		{
			name:  "identify, declared traits",
			rules: declared(rule(plan.EventTypeIdentify, "", plan.IdentitySectionTraits)),
			want:  []string{"type IdentifyTraits struct", method("Identify(identity Identity, traits *IdentifyTraits, opts ...Option)")},
		},
		{
			name:  "identify, variants",
			rules: variant(rule(plan.EventTypeIdentify, "", plan.IdentitySectionTraits)),
			want:  []string{"type IdentifyTraits interface", method("Identify(identity Identity, traits IdentifyTraits, opts ...Option)")},
		},
		{
			name:  "identify, empty and open",
			rules: open(rule(plan.EventTypeIdentify, "", plan.IdentitySectionTraits)),
			want:  []string{"type IdentifyTraits = map[string]any", method("Identify(identity Identity, traits IdentifyTraits, opts ...Option)")},
		},
		{
			name:  "identify, empty and closed",
			rules: closed(rule(plan.EventTypeIdentify, "", plan.IdentitySectionTraits)),
			want:  []string{method("Identify(identity Identity, opts ...Option)")},
		},
		{
			name:  "identify on context.traits, declared traits",
			rules: declared(rule(plan.EventTypeIdentify, "", plan.IdentitySectionContextTraits)),
			want:  []string{"type IdentifyTraits struct", method("Identify(identity Identity, traits *IdentifyTraits, opts ...Option)")},
		},
		{
			name:  "group, declared traits",
			rules: declared(rule(plan.EventTypeGroup, "", plan.IdentitySectionTraits)),
			want:  []string{"type GroupTraits struct", method("Group(identity Identity, groupID string, traits *GroupTraits, opts ...Option)")},
		},
		{
			name:  "group, variants",
			rules: variant(rule(plan.EventTypeGroup, "", plan.IdentitySectionTraits)),
			want:  []string{"type GroupTraits interface", method("Group(identity Identity, groupID string, traits GroupTraits, opts ...Option)")},
		},
		{
			name:  "group, empty and open",
			rules: open(rule(plan.EventTypeGroup, "", plan.IdentitySectionTraits)),
			want:  []string{"type GroupTraits = map[string]any", method("Group(identity Identity, groupID string, traits GroupTraits, opts ...Option)")},
		},
		{
			name:  "group, empty and closed",
			rules: closed(rule(plan.EventTypeGroup, "", plan.IdentitySectionTraits)),
			want:  []string{method("Group(identity Identity, groupID string, opts ...Option)")},
		},
		{
			name:  "group on context.traits, declared traits",
			rules: declared(rule(plan.EventTypeGroup, "", plan.IdentitySectionContextTraits)),
			want:  []string{"type GroupTraits struct", method("Group(identity Identity, groupID string, traits *GroupTraits, opts ...Option)")},
		},
		{
			name:  "page, declared properties",
			rules: declared(rule(plan.EventTypePage, "", plan.IdentitySectionProperties)),
			want:  []string{"type PageProperties struct", method("Page(identity Identity, name string, props *PageProperties, opts ...Option)")},
		},
		{
			name:  "page, variants",
			rules: variant(rule(plan.EventTypePage, "", plan.IdentitySectionProperties)),
			want:  []string{"type PageProperties interface", method("Page(identity Identity, name string, props PageProperties, opts ...Option)")},
		},
		{
			name:  "page, empty and open",
			rules: open(rule(plan.EventTypePage, "", plan.IdentitySectionProperties)),
			want:  []string{"type PageProperties = map[string]any", method("Page(identity Identity, name string, props PageProperties, opts ...Option)")},
		},
		{
			name:  "page, empty and closed",
			rules: closed(rule(plan.EventTypePage, "", plan.IdentitySectionProperties)),
			want:  []string{method("Page(identity Identity, name string, opts ...Option)")},
		},
		{
			name:  "screen, declared properties",
			rules: declared(rule(plan.EventTypeScreen, "", plan.IdentitySectionProperties)),
			want:  []string{"type ScreenProperties struct", method("Screen(identity Identity, name string, props *ScreenProperties, opts ...Option)")},
		},
		{
			name:  "screen, variants",
			rules: variant(rule(plan.EventTypeScreen, "", plan.IdentitySectionProperties)),
			want:  []string{"type ScreenProperties interface", method("Screen(identity Identity, name string, props ScreenProperties, opts ...Option)")},
		},
		{
			name:  "screen, empty and open",
			rules: open(rule(plan.EventTypeScreen, "", plan.IdentitySectionProperties)),
			want:  []string{"type ScreenProperties = map[string]any", method("Screen(identity Identity, name string, props ScreenProperties, opts ...Option)")},
		},
		{
			name:  "screen, empty and closed",
			rules: closed(rule(plan.EventTypeScreen, "", plan.IdentitySectionProperties)),
			want:  []string{method("Screen(identity Identity, name string, opts ...Option)")},
		},
		{
			name:  "named identify, whose method keeps its name",
			rules: declared(rule(plan.EventTypeIdentify, "User Identify", plan.IdentitySectionTraits)),
			want:  []string{"type IdentifyUserIdentifyTraits struct", method("Identify(identity Identity, traits *IdentifyUserIdentifyTraits, opts ...Option)")},
		},
		{
			name:  "named page",
			rules: declared(rule(plan.EventTypePage, "Product Page", plan.IdentitySectionProperties)),
			want:  []string{"type PageProductPageProperties struct", method("PageProductPage(identity Identity, name string, props *PageProductPageProperties, opts ...Option)")},
		},
		{
			name: "two identify rules, one per section: the second in rule-key order takes the suffix",
			rules: append(
				declared(rule(plan.EventTypeIdentify, "", plan.IdentitySectionTraits)),
				declared(rule(plan.EventTypeIdentify, "", plan.IdentitySectionContextTraits))...,
			),
			want: []string{
				"type IdentifyTraits struct",
				"type IdentifyTraits1 struct",
				method("Identify(identity Identity, traits *IdentifyTraits, opts ...Option)"),
				method("Identify1(identity Identity, traits *IdentifyTraits1, opts ...Option)"),
			},
		},
		{
			name:         "track on traits",
			rules:        []plan.EventRule{rule(plan.EventTypeTrack, "Some Event", plan.IdentitySectionTraits)},
			wantWarnings: []string{`invalid section "traits" for event type "track" and event name "Some Event"`},
		},
		{
			name:         "identify on properties",
			rules:        []plan.EventRule{rule(plan.EventTypeIdentify, "", plan.IdentitySectionProperties)},
			wantWarnings: []string{`invalid section "properties" for event type "identify"`},
		},
		{
			name:         "group on properties",
			rules:        []plan.EventRule{rule(plan.EventTypeGroup, "", plan.IdentitySectionProperties)},
			wantWarnings: []string{`invalid section "properties" for event type "group"`},
		},
		{
			name:         "page on traits",
			rules:        []plan.EventRule{rule(plan.EventTypePage, "", plan.IdentitySectionTraits)},
			wantWarnings: []string{`invalid section "traits" for event type "page"`},
		},
		{
			name:         "screen on context.traits",
			rules:        []plan.EventRule{rule(plan.EventTypeScreen, "", plan.IdentitySectionContextTraits)},
			wantWarnings: []string{`invalid section "context.traits" for event type "screen"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := captureWarnings(t)

			files, err := (&golang.Generator{}).Generate(&plan.TrackingPlan{Rules: tt.rules}, core.GenerateOptions{}, nil)
			require.NoError(t, err)
			require.Len(t, files, 1)

			var got []string
			for _, m := range payloadAndMethods.FindAllStringSubmatch(files[0].Content, -1) {
				got = append(got, m[1])
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantWarnings, *warnings)
		})
	}
}
