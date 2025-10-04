package plan_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/testutils"
	"github.com/stretchr/testify/assert"
)

func TestExtractAllCustomTypes(t *testing.T) {
	trackingPlan := testutils.GetReferenceTrackingPlan()

	customTypes := trackingPlan.ExtractAllCustomTypes()
	assert.Len(t, customTypes, testutils.ExpectedCustomTypeCount)

	expectedNames := []string{"email", "age", "active", "user_profile", "status", "email_list", "profile_list"}
	assert.Len(t, expectedNames, testutils.ExpectedCustomTypeCount) // consistency check
	for _, name := range expectedNames {
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
	expectedNames := []string{"email", "first_name", "last_name", "age", "active", "profile", "device_type", "tags", "contacts", "property_of_any", "untyped_field", "array_of_any", "untyped_array", "object_property", "status", "email_list", "profile_list", "ip_address", "nested_context", "context"}

	assert.Len(t, properties, testutils.ExpectedPropertyCount)
	assert.Len(t, expectedNames, testutils.ExpectedPropertyCount) // consistency check

	for _, name := range expectedNames {
		assert.Contains(t, properties, name, "Property %s should be present", name)
	}

	for name, property := range properties {
		assert.NotNil(t, property)
		referenceProperty := testutils.ReferenceProperties[name]
		assert.Equal(t, property, referenceProperty, "Property %s should match reference", name)
	}
}
