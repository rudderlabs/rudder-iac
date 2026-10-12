package plan_test

import (
	"runtime/debug"
	"testing"

	"github.com/rudderlabs/rudder-iac/typer/plan"
	"github.com/rudderlabs/rudder-iac/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

func TestExtractAllCustomTypes(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	customTypes := trackingPlan.ExtractAllCustomTypes()

	assert.Len(t, customTypes, len(testutils.ReferenceCustomTypes))
	for name := range testutils.ReferenceCustomTypes {
		assert.Contains(t, customTypes, name, "Custom type %s should be present", name)
	}

	for name, customType := range customTypes {
		assert.NotNil(t, customType)
		referenceType := testutils.ReferenceCustomTypes[name]
		assert.Equal(t, customType, referenceType, "Custom type %s should match reference", name)
	}
}

// A custom type that refers to itself is extracted once, without recursing
// forever.
func TestExtractAllCustomTypesRecursive(t *testing.T) {
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
		return &plan.TrackingPlan{Rules: []plan.EventRule{{Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"value": {Property: plan.Property{Name: "value", Types: []plan.PropertyType{ct}}},
		}}}}}
	}

	t.Run("direct", func(t *testing.T) {
		a := object("a")
		refer(a, a)

		assert.Equal(t, map[string]*plan.CustomType{"a": a}, trackingPlan(a).ExtractAllCustomTypes())
	})
	t.Run("indirect", func(t *testing.T) {
		a, b := object("a"), object("b")
		refer(a, b)
		refer(b, a)

		assert.Equal(t, map[string]*plan.CustomType{"a": a, "b": b}, trackingPlan(a).ExtractAllCustomTypes())
	})
}

func TestExtractAllProperties(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	properties := trackingPlan.ExtractAllProperties()

	assert.Len(t, properties, len(testutils.ReferenceProperties))
	for name := range testutils.ReferenceProperties {
		assert.Contains(t, properties, name, "Property %s should be present", name)
	}

	for name, property := range properties {
		assert.NotNil(t, property)
		referenceProperty := testutils.ReferenceProperties[name]
		assert.Equal(t, property, referenceProperty, "Property %s should match reference", name)
	}
}
