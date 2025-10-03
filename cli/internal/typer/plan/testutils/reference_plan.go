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

	ReferenceEvents["Identify"] = &plan.Event{
		EventType:   plan.EventTypeIdentify,
		Description: "User identification event",
	}

	ReferenceEvents["Page"] = &plan.Event{
		EventType:   plan.EventTypePage,
		Description: "Page view event",
	}

	ReferenceEvents["Screen"] = &plan.Event{
		EventType:   plan.EventTypeScreen,
		Description: "Screen view event",
	}

	ReferenceEvents["Group"] = &plan.Event{
		EventType:   plan.EventTypeGroup,
		Description: "Group association event",
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
		Type:        []plan.PropertyType{*ReferenceCustomTypes["email"]},
	}

	ReferenceProperties["first_name"] = &plan.Property{
		Name:        "first_name",
		Description: "User's first name",
		Type:        []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["last_name"] = &plan.Property{
		Name:        "last_name",
		Description: "User's last name",
		Type:        []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["age"] = &plan.Property{
		Name:        "age",
		Description: "User's age",
		Type:        []plan.PropertyType{*ReferenceCustomTypes["age"]},
	}

	ReferenceProperties["active"] = &plan.Property{
		Name:        "active",
		Description: "User active status",
		Type:        []plan.PropertyType{*ReferenceCustomTypes["active"]},
	}

	ReferenceProperties["device_type"] = &plan.Property{
		Name:        "device_type",
		Description: "Type of device",
		Type:        []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []string{"mobile", "tablet", "desktop", "smartTV", "IoT-Device"},
		},
	}

	ReferenceProperties["tags"] = &plan.Property{
		Name:        "tags",
		Description: "User tags as array of strings",
		Type:        []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemType:    []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["contacts"] = &plan.Property{
		Name:        "contacts",
		Description: "Array of user contacts",
		Type:        []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemType:    []plan.PropertyType{*ReferenceCustomTypes["email"]},
	}

	// Add properties for testing "any" type support
	ReferenceProperties["property_of_any"] = &plan.Property{
		Name:        "property_of_any",
		Description: "A field that can contain any type of value",
		Type:        []plan.PropertyType{plan.PrimitiveTypeAny},
	}

	ReferenceProperties["untyped_field"] = &plan.Property{
		Name:        "untyped_field",
		Description: "A field with no explicit type (treated as any)",
		Type:        []plan.PropertyType{},
	}

	ReferenceProperties["array_of_any"] = &plan.Property{
		Name:        "array_of_any",
		Description: "An array that can contain any type of items",
		Type:        []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemType:    []plan.PropertyType{plan.PrimitiveTypeAny},
	}

	ReferenceProperties["untyped_array"] = &plan.Property{
		Name:        "untyped_array",
		Description: "An array with no explicit item type (treated as any)",
		Type:        []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemType:    []plan.PropertyType{},
	}

	ReferenceProperties["object_property"] = &plan.Property{
		Name:        "object_property",
		Description: "An object field with no defined structure",
		Type:        []plan.PropertyType{plan.PrimitiveTypeObject},
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
		Type:        []plan.PropertyType{*ReferenceCustomTypes["user_profile"]},
	}
}

// GetReferenceTrackingPlan creates a tracking plan with various primitive and object custom types for testing
func GetReferenceTrackingPlan() *plan.TrackingPlan {
	var rules []plan.EventRule

	// Track event - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["User Signed Up"],
		Section: plan.IdentitySectionProperties,
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
				"device_type": {
					Property: *ReferenceProperties["device_type"],
					Required: false,
				},
				"tags": {
					Property: *ReferenceProperties["tags"],
					Required: false,
				},
				"contacts": {
					Property: *ReferenceProperties["contacts"],
					Required: false,
				},
				"property_of_any": {
					Property: *ReferenceProperties["property_of_any"],
					Required: false,
				},
				"untyped_field": {
					Property: *ReferenceProperties["untyped_field"],
					Required: false,
				},
				"array_of_any": {
					Property: *ReferenceProperties["array_of_any"],
					Required: false,
				},
				"untyped_array": {
					Property: *ReferenceProperties["untyped_array"],
					Required: false,
				},
				"object_property": {
					Property: *ReferenceProperties["object_property"],
					Required: false,
				},
			},
			AdditionalProperties: false,
		},
	})

	// Identify event - traits
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Identify"],
		Section: plan.IdentitySectionTraits,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"email": {
					Property: *ReferenceProperties["email"],
					Required: true,
				},
				"active": {
					Property: *ReferenceProperties["active"],
					Required: false,
				},
			},
			AdditionalProperties: false,
		},
	})

	// Page event - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Page"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"profile": {
					Property: *ReferenceProperties["profile"],
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
	})

	// Screen event - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Screen"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"profile": {
					Property: *ReferenceProperties["profile"],
					Required: false,
				},
			},
			AdditionalProperties: false,
		},
	})

	// Group event - traits
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Group"],
		Section: plan.IdentitySectionTraits,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"active": {
					Property: *ReferenceProperties["active"],
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
	})

	return &plan.TrackingPlan{
		Name:  "Test Plan",
		Rules: rules,
		EventContext: plan.EventContext{
			Platform:            "test",
			RudderCLIVersion:    "1.0.0",
			TrackingPlanID:      "plan_12345",
			TrackingPlanVersion: 13,
		},
	}
}

// Constants for test assertions based on the reference plan
const (
	ExpectedCustomTypeCount = 4  // email, age, active, user_profile
	ExpectedPropertyCount   = 14 // email, first_name, last_name, age, active, device_type, profile, tags, contacts, property_of_any, untyped_field, array_of_any, untyped_array, object_property
	ExpectedEventCount      = 5  // User Signed Up, Identify, Page, Screen, Group
)
