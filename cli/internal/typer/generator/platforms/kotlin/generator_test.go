package kotlin_test

import (
	_ "embed"
	"os"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/generator/kotlin"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/Main.kt
var mainContents string

func TestGenerate(t *testing.T) {
	// Create a tracking plan with primitive custom types
	trackingPlan := createTestTrackingPlan()

	files, err := kotlin.Generate(trackingPlan)

	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.NotNil(t, files[0])
	assert.Equal(t, "Main.kt", files[0].Path)

	content := files[0].Content

	// Create a preview file with the generated content to see the actual output
	err = os.MkdirAll("output", 0755)
	if err == nil {
		_ = os.WriteFile("output/Generated_Main.kt", []byte(content), 0644)
	}

	// Verify the file contains expected content from testdata
	assert.Contains(t, content, mainContents)
}

// createTestTrackingPlan creates a tracking plan with various primitive custom types for testing
func createTestTrackingPlan() *plan.TrackingPlan {
	// Create primitive custom types
	emailType := plan.CustomType{
		Name:        "email",
		Description: "Custom type for email validation",
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

	// Create properties that reference custom types
	emailProperty := plan.Property{
		Name:        "email",
		Description: "User's email address",
		Type:        emailType,
	}

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
				"email": {
					Property: emailProperty,
					Required: true,
				},
				"age": {
					Property: ageProperty,
					Required: false,
				},
				"active": {
					Property: activeProperty,
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
