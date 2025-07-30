package kotlin_test

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/platforms/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/Main.kt
var mainContent string

func TestGenerate(t *testing.T) {
	// Create a tracking plan with primitive custom types
	trackingPlan := createTestTrackingPlan()

	files, err := kotlin.Generate(trackingPlan)

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "Main.kt", files[0].Path)

	// Use go-cmp to provide a nice diff if the content doesn't match
	if diff := cmp.Diff(mainContent, files[0].Content); diff != "" {
		t.Errorf("Generated content does not match expected (-expected +actual):\n%s", diff)
	}
}

// createTestTrackingPlan creates a tracking plan with various primitive and object custom types for testing
func createTestTrackingPlan() *plan.TrackingPlan {
	// Create primitive custom types
	emailType := plan.CustomType{
		Name:        "email",
		Description: "Custom type for email validation",
		Type:        plan.PrimitiveTypeString,
	}

	emailProperty := plan.Property{
		Name:        "email",
		Description: "User's email address",
		Type:        emailType,
	}

	firstNameProperty := plan.Property{
		Name:        "first_name",
		Description: "User's first name",
		Type:        plan.PrimitiveTypeString,
	}

	lastNameProperty := plan.Property{
		Name:        "last_name",
		Description: "User's last name",
		Type:        plan.PrimitiveTypeString,
	}

	ageType := plan.CustomType{
		Name:        "age",
		Description: "User's age in years",
		Type:        plan.PrimitiveTypeNumber,
	}

	activeType := plan.CustomType{
		Name:        "active",
		Description: "Whether user is active",
		Type:        plan.PrimitiveTypeBoolean,
	}

	// Create object custom type
	userProfileType := plan.CustomType{
		Name:        "user_profile",
		Description: "User profile information",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"first_name": {
					Property: firstNameProperty,
					Required: true,
				},
				"last_name": {
					Property: lastNameProperty,
					Required: false,
				},
				"email": {
					Property: emailProperty,
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
	}

	// Create properties that reference custom types
	ageProperty := plan.Property{
		Name:        "age",
		Description: "User's age",
		Type:        ageType,
	}

	activeProperty := plan.Property{
		Name:        "active",
		Description: "User active status",
		Type:        activeType,
	}

	profileProperty := plan.Property{
		Name:        "profile",
		Description: "User profile data",
		Type:        userProfileType,
	}

	// Create an event that uses these custom types
	userSignedUpEvent := plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "User Signed Up",
		Description: "Triggered when a user signs up",
	}

	// Create event rule with properties using custom types
	eventRule := plan.EventRule{
		Event:   userSignedUpEvent,
		Section: plan.EventRuleSectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"age": {
					Property: ageProperty,
					Required: false,
				},
				"active": {
					Property: activeProperty,
					Required: true,
				},
				"profile": {
					Property: profileProperty,
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
	}

	return &plan.TrackingPlan{
		Name:  "Test Plan",
		Rules: []plan.EventRule{eventRule},
	}
}
