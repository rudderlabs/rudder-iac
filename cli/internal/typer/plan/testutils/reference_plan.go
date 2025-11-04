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

	// Event with quotes in name to test escaping
	ReferenceEvents["Product \"Premium\" Clicked"] = &plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "Product \"Premium\" Clicked",
		Description: "Triggered when user clicks on a \"premium\" product /* important */",
	}

	// Event with dollar sign in name to test $ escaping
	ReferenceEvents["$Variable$String"] = &plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "$Variable$String",
		Description: "Event with dollar signs to test string interpolation escaping",
	}

	ReferenceEvents["Empty Event With Additional Props"] = &plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "Empty Event With Additional Props",
		Description: "Empty event schema with additionalProperties true",
	}

	ReferenceEvents["Empty Event No Additional Props"] = &plan.Event{
		EventType:   plan.EventTypeTrack,
		Name:        "Empty Event No Additional Props",
		Description: "Empty event schema with additionalProperties false",
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

	// Property with special characters in description to test comment escaping
	ReferenceProperties["special_field"] = &plan.Property{
		Name:        "special_field",
		Description: "Field with special chars: \"quotes\", backslash\\path, and /* comment */",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	// Property with enum containing special characters
	ReferenceProperties["status_code"] = &plan.Property{
		Name:        "status_code",
		Description: "HTTP status with special characters",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []any{"200: OK", "404: Not Found", "500: Internal \"Server\" Error"},
		},
	}

	// Property with dollar sign in description and enum to test $ escaping
	ReferenceProperties["dollar_field"] = &plan.Property{
		Name:        "dollar_field",
		Description: "Field with $ for testing string interpolation: $variable and ${expression}",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []any{"$USD", "$100", "Price: $99.99", "$variable_name"},
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

	// Add properties for testing empty types support
	ReferenceProperties["property_of_any"] = &plan.Property{
		Name:        "property_of_any",
		Description: "A field that can contain any type of value",
		Types:       []plan.PropertyType{},
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
		ItemTypes:   []plan.PropertyType{},
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
				"first_name": {Property: *ReferenceProperties["first_name"], Required: true},
				"last_name":  {Property: *ReferenceProperties["last_name"]},
				"email":      {Property: *ReferenceProperties["email"], Required: true},
			},
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

	// Add empty custom type with additionalProperties: false for testing
	ReferenceCustomTypes["empty_object_no_additional_props"] = &plan.CustomType{
		Name:        "empty_object_no_additional_props",
		Description: "Empty object that does not allow additional properties",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{
			Properties:           map[string]plan.PropertySchema{},
			AdditionalProperties: false,
		},
	}

	ReferenceProperties["empty_object_no_additional_props"] = &plan.Property{
		Name:        "empty_object_no_additional_props",
		Description: "Property with empty object not allowing additional properties",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["empty_object_no_additional_props"]},
	}

	ReferenceProperties["nested_empty_object_no_additional_props"] = &plan.Property{
		Name:        "nested_empty_object_no_additional_props",
		Description: "Nested property with empty object not allowing additional properties",
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
				"page_type": {Property: *ReferenceProperties["page_type"], Required: true},
			},
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
								"query": {Property: *ReferenceProperties["query"], Required: true},
							},
						},
					},
					{
						DisplayName: "Product",
						Match:       []any{"product"},
						Description: "Product page variant",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"product_id": {Property: *ReferenceProperties["product_id"], Required: true},
							},
						},
					},
					{
						DisplayName: "Home",
						Match:       []any{"home"},
						Description: "Home page variant with no additional properties",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{},
						},
					},
				},
				DefaultSchema: &plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"page_data": {Property: *ReferenceProperties["page_data"]},
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
				"active": {Property: *ReferenceProperties["active"], Required: true},
			},
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
								"email": {Property: *ReferenceProperties["email"], Required: true},
							},
						},
					},
					{
						DisplayName: "Inactive",
						Match:       []any{false},
						Description: "Inactive user access",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"status": {Property: *ReferenceProperties["status"], Required: true},
							},
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
				"feature_flag": {Property: *ReferenceProperties["feature_flag"], Required: true},
			},
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
								"age": {Property: *ReferenceProperties["age"]},
							},
						},
					},
					{
						DisplayName: "Disabled",
						Match:       []any{false},
						Description: "Feature disabled (boolean false)",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"first_name": {Property: *ReferenceProperties["first_name"]},
							},
						},
					},
					{
						DisplayName: "Beta",
						Match:       []any{"beta"},
						Description: "Feature in beta (string 'beta')",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"tags": {Property: *ReferenceProperties["tags"]},
							},
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

	// Add properties with Unicode characters for testing
	ReferenceProperties["用户名"] = &plan.Property{
		Name:        "用户名",
		Description: "Username in Chinese characters",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceProperties["unicode_enum_field"] = &plan.Property{
		Name:        "unicode_enum_field",
		Description: "Field demonstrating various Unicode characters in enum values",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
		Config: &plan.PropertyConfig{
			Enum: []any{
				"🎯",        // Emoji
				"✅",        // Emoji
				"активный", // Cyrillic
				"已完成",      // Chinese
				"ενεργός",  // Greek
				"café",     // Latin with diacritics
				"!!!",      // Symbols only (backtick escape)
			},
		},
	}

	ReferenceProperties["mixed_unicode"] = &plan.Property{
		Name:        "mixed_unicode",
		Description: "Property with mixed unicode: café, naïve, 日本語",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString},
	}

	ReferenceCustomTypes["типы_данных"] = &plan.CustomType{
		Name:        "типы_данных",
		Description: "Custom type with Cyrillic name",
		Type:        plan.PrimitiveTypeString,
		Config: &plan.PropertyConfig{
			Enum: []any{"активный", "неактивный", "pending"},
		},
	}

	ReferenceProperties["unicode_custom_type"] = &plan.Property{
		Name:        "unicode_custom_type",
		Description: "Property using custom type with Unicode",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["типы_данных"]},
	}

	// Integer enum for testing non-string enum serialization
	ReferenceProperties["priority"] = &plan.Property{
		Name:        "priority",
		Description: "Priority level",
		Types:       []plan.PropertyType{plan.PrimitiveTypeInteger},
		Config: &plan.PropertyConfig{
			Enum: []any{1, 2, 3},
		},
	}

	// Boolean enum for testing non-string enum serialization
	ReferenceProperties["enabled"] = &plan.Property{
		Name:        "enabled",
		Description: "Feature enabled flag",
		Types:       []plan.PropertyType{plan.PrimitiveTypeBoolean},
		Config: &plan.PropertyConfig{
			Enum: []any{true, false},
		},
	}

	// Float enum for testing non-string enum serialization
	ReferenceProperties["rating"] = &plan.Property{
		Name:        "rating",
		Description: "Rating value",
		Types:       []plan.PropertyType{plan.PrimitiveTypeNumber},
		Config: &plan.PropertyConfig{
			Enum: []any{1.5, 2.5, 3.5, 4.5, 5.0},
		},
	}

	// Mixed-type enum for testing non-string enum serialization
	ReferenceProperties["mixed_value"] = &plan.Property{
		Name:        "mixed_value",
		Description: "Mixed type enum",
		Types:       []plan.PropertyType{},
		Config: &plan.PropertyConfig{
			Enum: []any{"active", 1, true, 2.5},
		},
	}

	// Null type support testing
	ReferenceCustomTypes["null_type"] = &plan.CustomType{
		Name:        "null_type",
		Description: "Custom type representing a null value",
		Type:        plan.PrimitiveTypeNull,
	}

	ReferenceProperties["null_field"] = &plan.Property{
		Name:        "null_field",
		Description: "Property that is always null",
		Types:       []plan.PropertyType{plan.PrimitiveTypeNull},
	}

	ReferenceProperties["custom_null_field"] = &plan.Property{
		Name:        "custom_null_field",
		Description: "Property using custom null type",
		Types:       []plan.PropertyType{*ReferenceCustomTypes["null_type"]},
	}

	ReferenceProperties["string_or_null"] = &plan.Property{
		Name:        "string_or_null",
		Description: "Property that can be string or null",
		Types:       []plan.PropertyType{plan.PrimitiveTypeString, plan.PrimitiveTypeNull},
	}

	ReferenceProperties["number_or_null"] = &plan.Property{
		Name:        "number_or_null",
		Description: "Property that can be number or null",
		Types:       []plan.PropertyType{plan.PrimitiveTypeNumber, plan.PrimitiveTypeNull},
	}

	ReferenceProperties["multi_type_with_null"] = &plan.Property{
		Name:        "multi_type_with_null",
		Description: "Property that can be string, integer, or null",
		Types: []plan.PropertyType{
			plan.PrimitiveTypeString,
			plan.PrimitiveTypeInteger,
			plan.PrimitiveTypeNull,
		},
	}

	ReferenceProperties["array_with_null_items"] = &plan.Property{
		Name:        "array_with_null_items",
		Description: "Array with items that can be string or null",
		Types:       []plan.PropertyType{plan.PrimitiveTypeArray},
		ItemTypes: []plan.PropertyType{
			plan.PrimitiveTypeString,
			plan.PrimitiveTypeNull,
		},
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
				"age":                                {Property: *ReferenceProperties["age"]},
				"active":                             {Property: *ReferenceProperties["active"], Required: true},
				"profile":                            {Property: *ReferenceProperties["profile"], Required: true},
				"device_type":                        {Property: *ReferenceProperties["device_type"]},
				"tags":                               {Property: *ReferenceProperties["tags"]},
				"contacts":                           {Property: *ReferenceProperties["contacts"]},
				"property_of_any":                    {Property: *ReferenceProperties["property_of_any"]},
				"untyped_field":                      {Property: *ReferenceProperties["untyped_field"]},
				"array_of_any":                       {Property: *ReferenceProperties["array_of_any"]},
				"untyped_array":                      {Property: *ReferenceProperties["untyped_array"]},
				"object_property":                    {Property: *ReferenceProperties["object_property"]},
				"status":                             {Property: *ReferenceProperties["status"]},
				"email_list":                         {Property: *ReferenceProperties["email_list"]},
				"profile_list":                       {Property: *ReferenceProperties["profile_list"]},
				"empty_object_with_additional_props": {Property: *ReferenceProperties["empty_object_with_additional_props"]},
				"nested_empty_object": {
					Property: *ReferenceProperties["nested_empty_object"],
					Schema: &plan.ObjectSchema{
						Properties:           map[string]plan.PropertySchema{},
						AdditionalProperties: true,
					},
				},
				"multi_type_field": {Property: *ReferenceProperties["multi_type_field"]},
				"multi_type_array": {Property: *ReferenceProperties["multi_type_array"]},
				"user_access":      {Property: *ReferenceProperties["user_access"]},
				"feature_config":   {Property: *ReferenceProperties["feature_config"]},
				// Add Unicode properties for testing
				"用户名":                              {Property: *ReferenceProperties["用户名"]},
				"unicode_enum_field":               {Property: *ReferenceProperties["unicode_enum_field"]},
				"mixed_unicode":                    {Property: *ReferenceProperties["mixed_unicode"]},
				"unicode_custom_type":              {Property: *ReferenceProperties["unicode_custom_type"]},
				"priority":                         {Property: *ReferenceProperties["priority"]},
				"enabled":                          {Property: *ReferenceProperties["enabled"]},
				"rating":                           {Property: *ReferenceProperties["rating"]},
				"mixed_value":                      {Property: *ReferenceProperties["mixed_value"]},
				"empty_object_no_additional_props": {Property: *ReferenceProperties["empty_object_no_additional_props"]},
				"nested_empty_object_no_additional_props": {
					Property: *ReferenceProperties["nested_empty_object_no_additional_props"],
					Required: false,
					Schema:   &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
				},
				// Null type properties for testing
				"null_field":            {Property: *ReferenceProperties["null_field"]},
				"custom_null_field":     {Property: *ReferenceProperties["custom_null_field"]},
				"string_or_null":        {Property: *ReferenceProperties["string_or_null"]},
				"number_or_null":        {Property: *ReferenceProperties["number_or_null"]},
				"multi_type_with_null":  {Property: *ReferenceProperties["multi_type_with_null"]},
				"array_with_null_items": {Property: *ReferenceProperties["array_with_null_items"]},
				// Add nested object properties for testing
				"context": {
					Property: *ReferenceProperties["context"],
					Schema: &plan.ObjectSchema{
						Properties: map[string]plan.PropertySchema{
							"ip_address": {Property: *ReferenceProperties["ip_address"], Required: true},
							"nested_context": {
								Property: *ReferenceProperties["nested_context"],
								Required: true,
								Schema: &plan.ObjectSchema{
									Properties: map[string]plan.PropertySchema{
										"profile": {Property: *ReferenceProperties["profile"]},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Event With Variants"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"profile":      {Property: *ReferenceProperties["profile"], Required: true},
				"page_context": {Property: *ReferenceProperties["page_context"]},
				"device_type":  {Property: *ReferenceProperties["device_type"], Required: true},
			},
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
								"tags": {Property: *ReferenceProperties["tags"]},
							},
						},
					},
					{
						DisplayName: "Desktop",
						Match:       []any{"desktop"},
						Description: "Desktop page view",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"first_name": {Property: *ReferenceProperties["first_name"], Required: true},
								"last_name":  {Property: *ReferenceProperties["last_name"]},
							},
						},
					},
				},
				DefaultSchema: &plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"untyped_field": {Property: *ReferenceProperties["untyped_field"]},
					},
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
				"email":  {Property: *ReferenceProperties["email"], Required: true},
				"active": {Property: *ReferenceProperties["active"]},
			},
		},
	})

	// Page event - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Page"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"profile": {Property: *ReferenceProperties["profile"], Required: true},
			},
		},
	})

	// Screen event - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Screen"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"profile": {Property: *ReferenceProperties["profile"]},
			},
		},
	})

	// Group event - traits
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Group"],
		Section: plan.IdentitySectionTraits,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"active": {Property: *ReferenceProperties["active"], Required: true},
			},
		},
	})

	// Track event with special characters - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Product \"Premium\" Clicked"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"special_field": {
					Property: *ReferenceProperties["special_field"],
					Required: true,
				},
				"status_code": {
					Property: *ReferenceProperties["status_code"],
					Required: false,
				},
			},
		},
	})

	// Track event with dollar sign - properties
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["$Variable$String"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{
				"dollar_field": {
					Property: *ReferenceProperties["dollar_field"],
					Required: true,
				},
			},
		},
	})

	// Empty event with additionalProperties: true
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Empty Event With Additional Props"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties:           map[string]plan.PropertySchema{},
			AdditionalProperties: true,
		},
	})

	// Empty event with additionalProperties: false
	rules = append(rules, plan.EventRule{
		Event:   *ReferenceEvents["Empty Event No Additional Props"],
		Section: plan.IdentitySectionProperties,
		Schema: plan.ObjectSchema{
			Properties: map[string]plan.PropertySchema{},
		},
	})

	return &plan.TrackingPlan{
		Name:  "Test Plan",
		Rules: rules,
		Metadata: plan.PlanMetadata{
			TrackingPlanID:      "plan_12345",
			TrackingPlanVersion: 13,
			URL:                 "https://app.rudderstack.com/trackingPlans/plan_12345",
		},
	}
}
