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

	ReferenceEvents["Event With Variants"] = &plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "Event With Variants",
		Description: "Example event to demonstrate variants",
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

	// Add properties for testing nested objects
	ReferenceProperties["ip_address"] = &plan.Property{
		Name:        "ip_address",
		Description: "IP address of the user",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["nested_context"] = &plan.Property{
		Name:        "nested_context",
		Description: "demonstrates multiple levels of nesting",
		Types:       []plan.PropertyType{plan.PrimitiveTypeObject},
	}

	ReferenceProperties["context"] = &plan.Property{
		Name:        "context",
		Description: "example of object property",
		Types:       []plan.PropertyType{plan.PrimitiveTypeObject},
	}

	// Add multi-type properties for testing
	ReferenceProperties["multi_type_field"] = &plan.Property{
		Name:        "multi_type_field",
		Description: "A field that can be string, integer, or boolean",
		Types: []plan.PropertyType{
			plan.PrimitiveTypeString,
			plan.PrimitiveTypeInteger,
			plan.PrimitiveTypeBoolean,
		},
	}

	ReferenceProperties["multi_type_array"] = &plan.Property{
		Name:        "multi_type_array",
		Description: "An array with items that can be string or integer",
		Types:       []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemTypes: []plan.PropertyType{
			plan.PrimitiveTypeString,
			plan.PrimitiveTypeInteger,
		},
	}

	// Add empty custom type with additionalProperties: true for testing
	ReferenceCustomTypes["empty_object_with_additional_props"] = &plan.CustomType{
		Name:        "empty_object_with_additional_props",
		Description: "Empty object that allows additional properties",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties:           map[string]plan.PropertySchema{},
			AdditionalProperties: true,
		},
	}

	ReferenceProperties["empty_object_with_additional_props"] = &plan.Property{
		Name:        "empty_object_with_additional_props",
		Description: "Property with empty object allowing additional properties",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["empty_object_with_additional_props"]},
	}

	ReferenceProperties["nested_empty_object"] = &plan.Property{
		Name:        "nested_empty_object",
		Description: "Nested property with empty object allowing additional properties",
		Types:       []plan.PropertyType{plan.PrimitiveTypeObject},
	}

	// Properties for variant testing
	ReferenceProperties["page_type"] = &plan.Property{
		Name:        "page_type",
		Description: "Type of page",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["query"] = &plan.Property{
		Name:        "query",
		Description: "Search query",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["product_id"] = &plan.Property{
		Name:        "product_id",
		Description: "Product identifier",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["page_data"] = &plan.Property{
		Name:        "page_data",
		Description: "Additional page data",
		Types:       []plan.PropertyType{plan.PrimitiveTypeObject},
	}

	// Custom type with variants (properties defined inline, but also added to ReferenceProperties for extraction)
	ReferenceCustomTypes["page_context"] = &plan.CustomType{
		Name:        "page_context",
		Description: "Page context with variants based on page type",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"page_type": {
					Property: plan.Property{
						Name:        "page_type",
						Description: "Type of page",
						Types:       []plan.PropertyType{plan.PrimitiveTypeString},
					},
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
		Variants: []plan.Variant{
			{
				Type:          "discriminator",
				Discriminator: "page_type",
				Cases: []plan.VariantCase{
					{
						DisplayName: "Search",
						Match:       []any{"search"},
						Description: "Search page variant",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"query": {
									Property: plan.Property{
										Name:        "query",
										Description: "Search query",
										Types:       []plan.PropertyType{plan.PrimitiveTypeString},
									},
									Required: true,
								},
							},
						},
					},
					{
						DisplayName: "Product",
						Match:       []any{"product"},
						Description: "Product page variant",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"product_id": {
									Property: plan.Property{
										Name:        "product_id",
										Description: "Product identifier",
										Types:       []plan.PropertyType{plan.PrimitiveTypeString},
									},
									Required: true,
								},
							},
						},
					},
				},
				DefaultSchema: &plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"page_data": {
							Property: plan.Property{
								Name:        "page_data",
								Description: "Additional page data",
								Types:       []plan.PropertyType{plan.PrimitiveTypeObject},
							},
							Required: false,
						},
					},
				},
			},
		},
	}

	ReferenceProperties["page_context"] = &plan.Property{
		Name:        "page_context",
		Description: "Page context information",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["page_context"]},
	}

	// Custom type with boolean discriminator (no default case)
	ReferenceCustomTypes["user_access"] = &plan.CustomType{
		Name:        "user_access",
		Description: "User access with variants based on active status",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"active": {
					Property: *ReferenceProperties["active"],
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
		Variants: []plan.Variant{
			{
				Type:          "discriminator",
				Discriminator: "active",
				Cases: []plan.VariantCase{
					{
						DisplayName: "Active",
						Match:       []any{true},
						Description: "Active user access",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"email": {
									Property: *ReferenceProperties["email"],
									Required: true,
								},
							},
							AdditionalProperties: false,
						},
					},
					{
						DisplayName: "Inactive",
						Match:       []any{false},
						Description: "Inactive user access",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"status": {
									Property: *ReferenceProperties["status"],
									Required: true,
								},
							},
							AdditionalProperties: false,
						},
					},
				},
			},
		},
	}

	ReferenceProperties["user_access"] = &plan.Property{
		Name:        "user_access",
		Description: "User access information",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["user_access"]},
	}

	// Properties for multi-type discriminator testing
	ReferenceProperties["feature_flag"] = &plan.Property{
		Name:        "feature_flag",
		Description: "Feature flag that can be boolean or string",
		Types:       []plan.PropertyType{plan.PrimitiveTypeBoolean, plan.PrimitiveTypeString},
	}

	// Custom type with multi-type discriminator (boolean | string)
	ReferenceCustomTypes["feature_config"] = &plan.CustomType{
		Name:        "feature_config",
		Description: "Feature configuration with variants based on multi-type flag",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"feature_flag": {
					Property: plan.Property{
						Name:        "feature_flag",
						Description: "Feature flag that can be boolean or string",
						Types:       []plan.PropertyType{plan.PrimitiveTypeBoolean, plan.PrimitiveTypeString},
					},
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
		Variants: []plan.Variant{
			{
				Type:          "discriminator",
				Discriminator: "feature_flag",
				Cases: []plan.VariantCase{
					{
						DisplayName: "Enabled",
						Match:       []any{true},
						Description: "Feature enabled (boolean true)",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"email": {
									Property: *ReferenceProperties["email"],
									Required: true,
								},
							},
							AdditionalProperties: false,
						},
					},
					{
						DisplayName: "Disabled",
						Match:       []any{false},
						Description: "Feature disabled (boolean false)",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"status": {
									Property: *ReferenceProperties["status"],
									Required: true,
								},
							},
							AdditionalProperties: false,
						},
					},
					{
						DisplayName: "Beta",
						Match:       []any{"beta"},
						Description: "Feature in beta (string 'beta')",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"tags": {
									Property: *ReferenceProperties["tags"],
									Required: true,
								},
							},
							AdditionalProperties: false,
						},
					},
				},
			},
		},
	}

	ReferenceProperties["feature_config"] = &plan.Property{
		Name:        "feature_config",
		Description: "Feature configuration information",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["feature_config"]},
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
				"empty_object_with_additional_props": {
					Property: *ReferenceProperties["empty_object_with_additional_props"],
					Required: false,
				},
				"nested_empty_object": {
					Property: *ReferenceProperties["nested_empty_object"],
					Required: false,
					Schema: &plan.ObjectSchema{
						Properties:           map[string]plan.PropertySchema{},
						AdditionalProperties: true,
					},
				},
				"multi_type_field": {
					Property: *ReferenceProperties["multi_type_field"],
					Required: false,
				},
				"multi_type_array": {
					Property: *ReferenceProperties["multi_type_array"],
					Required: false,
				},
				"user_access": {
					Property: *ReferenceProperties["user_access"],
					Required: false,
				},
				"feature_config": {
					Property: *ReferenceProperties["feature_config"],
					Required: false,
				},
				// Add nested object properties for testing
				"context": {
					Property: *ReferenceProperties["context"],
					Required: false,
					Schema: &plan.ObjectSchema{
						Properties: map[string]plan.PropertySchema{
							"ip_address": {
								Property: *ReferenceProperties["ip_address"],
								Required: true,
							},
							"nested_context": {
								Property: *ReferenceProperties["nested_context"],
								Required: true,
								Schema: &plan.ObjectSchema{
									Properties: map[string]plan.PropertySchema{
										"profile": {
											Property: *ReferenceProperties["profile"],
											Required: false,
										},
									},
									AdditionalProperties: false,
								},
							},
						},
						AdditionalProperties: false,
					},
				},
			},
			AdditionalProperties: false,
		},
	})

	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Event With Variants"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"profile": {
					Property: *ReferenceProperties["profile"],
					Required: true,
				},
				"page_context": {
					Property: *ReferenceProperties["page_context"],
					Required: false,
				},
				"device_type": {
					Property: *ReferenceProperties["device_type"],
					Required: true,
				},
			},
			AdditionalProperties: false,
		},
		Variants: []plan.Variant{
			{
				Type:          "discriminator",
				Discriminator: "device_type",
				Cases: []plan.VariantCase{
					{
						DisplayName: "Mobile",
						Match:       []any{"mobile"},
						Description: "Mobile device page view",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"tags": {
									Property: *ReferenceProperties["tags"],
									Required: true,
								},
							},
							AdditionalProperties: false,
						},
					},
					{
						DisplayName: "Desktop",
						Match:       []any{"desktop"},
						Description: "Desktop page view",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"first_name": {
									Property: *ReferenceProperties["first_name"],
									Required: true,
								},
								"last_name": {
									Property: *ReferenceProperties["last_name"],
									Required: false,
								},
							},
							AdditionalProperties: false,
						},
					},
				},
				DefaultSchema: &plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"untyped_field": {
							Property: *ReferenceProperties["untyped_field"],
							Required: false,
						},
					},
					AdditionalProperties: false,
				},
			},
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
	ExpectedCustomTypeCount = 11 // email, age, active, user_profile, status, email_list, profile_list, empty_object_with_additional_props, page_context, user_access, feature_config
	ExpectedPropertyCount   = 32 // email, first_name, last_name, age, active, device_type, profile, tags, contacts, property_of_any, untyped_field, array_of_any, untyped_array, object_property, status, email_list, profile_list, ip_address, nested_context, context, empty_object_with_additional_props, nested_empty_object, page_type, query, product_id, page_data, page_context, multi_type_field, multi_type_array, user_access, feature_flag, feature_config
	ExpectedEventCount      = 5  // User Signed Up, Identify, Page, Screen, Group
)
