package golang

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The categoryMode each page or screen method passes for its schema, and each
// struct of a variant payload returns for its own: the base merged with its
// branch.
func TestGenerateCategoryMode(t *testing.T) {
	var (
		property = func(name string, required bool) map[string]plan.PropertySchema {
			return map[string]plan.PropertySchema{name: {Property: plan.Property{Name: name, Types: onlyString}, Required: required}}
		}
		rule = func(eventType plan.EventType, open bool, properties map[string]plan.PropertySchema) plan.EventRule {
			return plan.EventRule{
				Event:   plan.Event{EventType: eventType},
				Section: plan.IdentitySectionProperties,
				Schema:  plan.ObjectSchema{Properties: properties, AdditionalProperties: open},
			}
		}
		variants = func(r plan.EventRule, key string, cases ...plan.VariantCase) plan.EventRule {
			r.Variants = []plan.Variant{{Type: "discriminator", Discriminator: key, Cases: cases}}
			return r
		}
		branch = func(match string, open bool, properties map[string]plan.PropertySchema) plan.VariantCase {
			return plan.VariantCase{Match: []any{match}, Schema: plan.ObjectSchema{Properties: properties, AdditionalProperties: open}}
		}
	)

	tests := []struct {
		name string
		rule plan.EventRule
		want map[string]string
	}{
		{
			name: "declared field",
			rule: rule(plan.EventTypePage, false, property("category", false)),
			want: map[string]string{"Page": "categoryDeclared"},
		},
		{
			name: "declared field of an open schema",
			rule: rule(plan.EventTypePage, true, property("category", true)),
			want: map[string]string{"Page": "categoryDeclared"},
		},
		{
			name: "open",
			rule: rule(plan.EventTypePage, true, property("title", true)),
			want: map[string]string{"Page": "categoryOpen"},
		},
		{
			name: "closed",
			rule: rule(plan.EventTypeScreen, false, property("title", true)),
			want: map[string]string{"Screen": "categoryClosed"},
		},
		{
			name: "empty and open",
			rule: rule(plan.EventTypeScreen, true, nil),
			want: map[string]string{"Screen": "categoryOpen"},
		},
		{
			name: "empty and closed",
			rule: rule(plan.EventTypePage, false, nil),
			want: map[string]string{"Page": "categoryClosed"},
		},
		{
			name: "discriminator",
			rule: variants(rule(plan.EventTypePage, true, property("category", true)), "category", branch("news", true, nil)),
			want: map[string]string{
				"Page":                   "categoryDeclared",
				"PagePropertiesCaseNews": "categoryDeclared",
				"PagePropertiesDefault":  "categoryDeclared",
			},
		},
		{
			// A branch is open only when its base and its own schema are, and
			// Default without a schema of its own is closed.
			name: "variant branches of an open base",
			rule: variants(rule(plan.EventTypePage, true, property("kind", false)), "kind",
				branch("open", true, nil),
				branch("closed", false, nil),
				branch("declared", true, property("category", false)),
			),
			want: map[string]string{
				"Page":                       "categoryOpen",
				"PagePropertiesAbsent":       "categoryOpen",
				"PagePropertiesCaseOpen":     "categoryOpen",
				"PagePropertiesCaseClosed":   "categoryClosed",
				"PagePropertiesCaseDeclared": "categoryDeclared",
				"PagePropertiesDefault":      "categoryClosed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := newContext(&plan.TrackingPlan{Rules: []plan.EventRule{tt.rule}}, "", "ruddertyper")
			require.NoError(t, err)

			got := map[string]string{}
			for _, m := range ctx.Methods {
				got[m.Name] = m.CategoryMode
			}
			for _, d := range ctx.Payloads {
				if d.Struct != nil && d.Struct.CategoryMode != "" {
					got[d.Struct.Name] = d.Struct.CategoryMode
				}
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
