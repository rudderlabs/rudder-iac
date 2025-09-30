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
		Types:       []plan.PropertyType{*ReferenceCustomTypes["email"]},
	}

	ReferenceProperties["first_name"] = &plan.Property{
		Name:        "first_name",
		Description: "User's first name",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["last_name"] = &plan.Property{
		Name:        "last_name",
		Description: "User's last name",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["age"] = &plan.Property{
		Name:        "age",
		Description: "User's age",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["age"]},
	}

	ReferenceProperties["active"] = &plan.Property{
		Name:        "active",
		Description: "User active status",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["active"]},
	}

	ReferenceProperties["device_type"] = &plan.Property{
		Name:        "device_type",
		Description: "Type of device",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []any{"mobile", "tablet", "desktop", "smartTV", "IoT-Device"},
		},
	}

	ReferenceProperties["tags"] = &plan.Property{
		Name:        "tags",
		Description: "User tags as array of strings",
		Types:       []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemTypes:   []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["contacts"] = &plan.Property{
		Name:        "contacts",
		Description: "Array of user contacts",
		Types:       []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemTypes:   []plan.PropertyType{*ReferenceCustomTypes["email"]},
	}

	// Add properties for testing "any" type support
	ReferenceProperties["property_of_any"] = &plan.Property{
		Name:        "property_of_any",
		Description: "A field that can contain any type of value",
		Types:       []plan.PropertyType{plan.PrimitiveTypeAny},
	}

	ReferenceProperties["untyped_field"] = &plan.Property{
		Name:        "untyped_field",
		Description: "A field with no explicit type (treated as any)",
		Types:       []plan.PropertyType{},
	}

	ReferenceProperties["array_of_any"] = &plan.Property{
		Name:        "array_of_any",
		Description: "An array that can contain any type of items",
		Types:       []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemTypes:   []plan.PropertyType{plan.PrimitiveTypeAny},
	}

	ReferenceProperties["untyped_array"] = &plan.Property{
		Name:        "untyped_array",
		Description: "An array with no explicit item type (treated as any)",
		Types:       []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemTypes:   []plan.PropertyType{},
	}

	ReferenceProperties["object_property"] = &plan.Property{
		Name:        "object_property",
		Description: "An object field with no defined structure",
		Types:       []plan.PropertyType{plan.PrimitiveTypeObject},
	}

	ReferenceProperties["device_type"] = &plan.Property{
		Name:        "device_type",
		Description: "Type of device",
		Type:        []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []string{"mobile", "tablet", "desktop", "smartTV", "IoT-Device"},
		},
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
		Types:       []plan.PropertyType{*ReferenceCustomTypes["user_profile"]},
	}

	// Custom type with enum
	ReferenceCustomTypes["status"] = &plan.CustomType{
		Name:        "status",
		Description: "User status enum",
		Type:        plan.PrimitiveTypeString,
		Config: &plan.PropertyConfig{
			Enum: []any{"pending", "active", "suspended", "deleted"},
		},
	}

	ReferenceProperties["status"] = &plan.Property{
		Name:        "status",
		Description: "User account status",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["status"]},
	}

	// Custom array type with primitive items
	ReferenceCustomTypes["email_list"] = &plan.CustomType{
		Name:        "email_list",
		Description: "List of email addresses",
		Type:        plan.PrimitiveTypeArray,
		ItemType:    ReferenceCustomTypes["email"],
	}

	ReferenceProperties["email_list"] = &plan.Property{
		Name:        "email_list",
		Description: "User's email addresses",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["email_list"]},
	}

	// Custom array type with object items
	ReferenceCustomTypes["profile_list"] = &plan.CustomType{
		Name:        "profile_list",
		Description: "List of user profiles",
		Type:        plan.PrimitiveTypeArray,
		ItemType:    ReferenceCustomTypes["user_profile"],
	}

	ReferenceProperties["profile_list"] = &plan.Property{
		Name:        "profile_list",
		Description: "List of related user profiles",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["profile_list"]},
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
				"status": {
					Property: *ReferenceProperties["status"],
					Required: false,
				},
				"email_list": {
					Property: *ReferenceProperties["email_list"],
					Required: false,
				},
				"profile_list": {
					Property: *ReferenceProperties["profile_list"],
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
		Metadata: plan.PlanMetadata{
			TrackingPlanID:      "plan_12345",
			TrackingPlanVersion: 13,
		},
	}
}

// Constants for test assertions based on the reference plan
const (
	ExpectedCustomTypeCount = 7  // email, age, active, user_profile, status, email_list, profile_list
	ExpectedPropertyCount   = 17 // email, first_name, last_name, age, active, device_type, profile, tags, contacts, property_of_any, untyped_field, array_of_any, untyped_array, object_property, status, email_list, profile_list
	ExpectedEventCount      = 5  // User Signed Up, Identify, Page, Screen, Group
)
