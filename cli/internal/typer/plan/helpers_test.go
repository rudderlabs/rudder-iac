package plan_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
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
