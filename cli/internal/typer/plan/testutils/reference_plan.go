package testutils

import "github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"

var ReferenceCustomTypes map[string]*plan.CustomType = make(map[string]*plan.CustomType)
var ReferenceProperties map[string]*plan.Property = make(map[string]*plan.Property)
var ReferenceEvents map[string]*plan.Event = make(map[string]*plan.Event)

func init() {
	ReferenceEvents["User Signed Up"] = &plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "User Signed Up",
		Description: "Triggered when a user signs up",
	}

	ReferenceCustomTypes["email"] = &plan.CustomType{
		Name:        "email",
		Description: "Custom type for email validation",
		Type:        plan.PrimitiveTypeString,
	}
	ReferenceCustomTypes["age"] = &plan.CustomType{
		Name:        "age",
		Description: "User's age in years",
		Type:        plan.PrimitiveTypeNumber,
	}

	ReferenceCustomTypes["active"] = &plan.CustomType{
		Name:        "active",
		Description: "Whether user is active",
		Type:        plan.PrimitiveTypeBoolean,
	}

	ReferenceProperties["email"] = &plan.Property{
		Name:        "email",
		Description: "User's email address",
		Type:        *ReferenceCustomTypes["email"],
	}

	ReferenceProperties["first_name"] = &plan.Property{
		Name:        "first_name",
		Description: "User's first name",
		Type:        plan.PrimitiveTypeString,
	}

	ReferenceProperties["last_name"] = &plan.Property{
		Name:        "last_name",
		Description: "User's last name",
		Type:        plan.PrimitiveTypeString,
	}

	ReferenceProperties["age"] = &plan.Property{
		Name:        "age",
		Description: "User's age",
		Type:        *ReferenceCustomTypes["age"],
	}

	ReferenceProperties["active"] = &plan.Property{
		Name:        "active",
		Description: "User active status",
		Type:        *ReferenceCustomTypes["active"],
	}

	ReferenceCustomTypes["user_profile"] = &plan.CustomType{
		Name:        "user_profile",
		Description: "User profile information",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"first_name": {
					Property: *ReferenceProperties["first_name"],
					Required: true,
				},
				"last_name": {
					Property: *ReferenceProperties["last_name"],
					Required: false,
				},
				"email": {
					Property: *ReferenceProperties["email"],
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
	}

	ReferenceProperties["profile"] = &plan.Property{
		Name:        "profile",
		Description: "User profile data",
		Type:        *ReferenceCustomTypes["user_profile"],
	}
}

// GetReferenceTrackingPlan creates a tracking plan with various primitive and object custom types for testing
func GetReferenceTrackingPlan() *plan.TrackingPlan {
	// Create event rule with properties using custom types
	eventRule := plan.EventRule{
		Event:   *ReferenceEvents["User Signed Up"],
		Section: plan.EventRuleSectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"age": {
					Property: *ReferenceProperties["age"],
					Required: false,
				},
				"active": {
					Property: *ReferenceProperties["active"],
					Required: true,
				},
				"profile": {
					Property: *ReferenceProperties["profile"],
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

// Constants for test assertions based on the reference plan
const (
	ExpectedCustomTypeCount = 4 // email, age, active, user_profile
	ExpectedPropertyCount   = 6 // email, first_name, last_name, age, active, profile
	ExpectedEventCount      = 1 // User Signed Up
)
