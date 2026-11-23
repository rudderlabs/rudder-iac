package generator

import (
	"encoding/json"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/typer/generator/core"
	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The remote plan provider keeps enum and match numbers as json.Number, so Go
// generation sees their exact values. Every generator renders one as it
// renders the float64 a local plan holds.
func TestJSONNumbersRenderAsFloat64(t *testing.T) {
	trackingPlan := func(number func(string) any) *plan.TrackingPlan {
		enum := func(name string, values []any, types ...plan.PropertyType) plan.PropertySchema {
			return plan.PropertySchema{
				Property: plan.Property{Name: name, Types: types, Config: &plan.PropertyConfig{Enum: values}},
				Required: true,
			}
		}
		level := &plan.CustomType{
			Name:   "level",
			Type:   plan.PrimitiveTypeInteger,
			Config: &plan.PropertyConfig{Enum: []any{number("1"), number("2.0")}},
		}
		return &plan.TrackingPlan{Name: "Numbers", Rules: []plan.EventRule{{
			Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Numbers"},
			Section: plan.IdentitySectionProperties,
			Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"code":  enum("code", []any{number("200"), number("4.04e2")}, plan.PrimitiveTypeInteger),
				"ratio": enum("ratio", []any{number("0.50"), number("2.0")}, plan.PrimitiveTypeNumber),
				"mixed": enum("mixed", []any{"a", number("1.0"), number("2.50"), true}),
				"level": {Property: plan.Property{Name: "level", Types: []plan.PropertyType{level}}},
			}},
			Variants: []plan.Variant{{
				Type:          "discriminator",
				Discriminator: "code",
				Cases: []plan.VariantCase{{
					DisplayName: "Not Found",
					Match:       []any{number("4.04e2")},
					Schema:      plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
				}},
			}},
		}}}
	}
	var (
		float64Number = func(s string) any { f, _ := strconv.ParseFloat(s, 64); return f }
		jsonNumber    = func(s string) any { return json.Number(s) }
	)

	for name, g := range platforms {
		t.Run(name, func(t *testing.T) {
			want, err := g.Generate(trackingPlan(float64Number), core.GenerateOptions{}, nil)
			require.NoError(t, err)
			got, err := g.Generate(trackingPlan(jsonNumber), core.GenerateOptions{}, nil)
			require.NoError(t, err)

			assert.Equal(t, want, got)
		})
	}
}

// Extracting a recursive custom type's references used to recurse forever on
// every platform. Each generator only has to return here, not to emit code that
// compiles: Swift's output, and TypeScript's once a key is renamed, does not.
// Only Go rejects the cycle, and its own tests pin the error.
func TestRecursiveCustomTypesTerminate(t *testing.T) {
	// Should the recursion come back, crash fast instead of growing the stack
	// to 1 GB.
	maxStack := debug.SetMaxStack(64 << 20)
	t.Cleanup(func() { debug.SetMaxStack(maxStack) })

	object := func(name string) *plan.CustomType {
		return &plan.CustomType{Name: name, Type: plan.PrimitiveTypeObject, Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}}}
	}
	refer := func(from, to *plan.CustomType) {
		from.Schema.Properties[to.Name] = plan.PropertySchema{Property: plan.Property{Name: to.Name, Types: []plan.PropertyType{to}}}
	}
	trackingPlan := func(ct *plan.CustomType) *plan.TrackingPlan {
		return &plan.TrackingPlan{Name: "Recursive", Rules: []plan.EventRule{{
			Event:   plan.Event{EventType: plan.EventTypeTrack, Name: "Recursive"},
			Section: plan.IdentitySectionProperties,
			Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"value": {Property: plan.Property{Name: "value", Types: []plan.PropertyType{ct}}},
			}},
		}}}
	}
	cycles := map[string]func() *plan.TrackingPlan{
		"direct": func() *plan.TrackingPlan {
			a := object("a")
			refer(a, a)
			return trackingPlan(a)
		},
		"indirect": func() *plan.TrackingPlan {
			a, b := object("a"), object("b")
			refer(a, b)
			refer(b, a)
			return trackingPlan(a)
		},
	}

	for name, g := range platforms {
		for cycle, cyclicPlan := range cycles {
			t.Run(name+"/"+cycle, func(t *testing.T) {
				// The stack limit cannot catch a loop that never grows the stack.
				done := make(chan struct{})
				go func() {
					_, _ = g.Generate(cyclicPlan(), core.GenerateOptions{}, nil)
					close(done)
				}()

				select {
				case <-done:
				case <-time.After(5 * time.Second):
					require.FailNow(t, "generation did not return")
				}
			})
		}
	}
}
