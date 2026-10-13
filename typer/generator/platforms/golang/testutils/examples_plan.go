package testutils

import "github.com/rudderlabs/rudder-iac/typer/plan"

// GetExamplesTrackingPlan returns the Go generation spec's input examples plus
// the shapes the shared reference plan lacks, so the examples golden pins how
// each of them is generated.
func GetExamplesTrackingPlan() *plan.TrackingPlan {
	var (
		object = plan.PrimitiveTypeObject
		// A primitive custom type, and an open object one whose field takes it.
		someStringType = &plan.CustomType{Name: "SomeStringType", Description: "some string custom type", Type: plan.PrimitiveTypeString}
		someObjectType = &plan.CustomType{
			Name:        "SomeObjectType",
			Description: "some object custom type",
			Type:        object,
			Schema: &plan.ObjectSchema{AdditionalProperties: true, Properties: map[string]plan.PropertySchema{
				"someCustomString": required(property("someCustomString", "", someStringType)),
				"someInteger":      optional(property("someInteger", "some integer property", plan.PrimitiveTypeInteger)),
			}},
		}
		// A custom type with an inline object, which Go hoists as well.
		userProfile = &plan.CustomType{
			Name: "UserProfile",
			Type: object,
			Schema: &plan.ObjectSchema{AdditionalProperties: true, Properties: map[string]plan.PropertySchema{
				"address": nested(optional(property("address", "", object)), true, map[string]plan.PropertySchema{
					"street": required(property("street", "", plan.PrimitiveTypeString)),
				}),
			}},
		}
		row = &plan.CustomType{Name: "Row", Type: plan.PrimitiveTypeArray, ItemType: plan.PrimitiveTypeString}
	)

	return &plan.TrackingPlan{
		Name: "Examples Plan",
		// No URL and version 0, as for a plan generated with --local.
		Metadata: plan.PlanMetadata{TrackingPlanID: "plan_examples"},
		Rules: []plan.EventRule{
			track("Some Track Event", "This is a track event for testing.", false, map[string]plan.PropertySchema{
				"someString":  required(property("someString", "some string property", plan.PrimitiveTypeString)),
				"someInteger": required(property("someInteger", "some integer property", plan.PrimitiveTypeInteger)),
				"someBoolean": optional(property("someBoolean", "some boolean property", plan.PrimitiveTypeBoolean)),
				// Nested rule objects are open unless the plan says otherwise.
				"someNestedObject": nested(optional(property("someNestedObject", "", object)), true, map[string]plan.PropertySchema{
					"someNestedLevel1": nested(optional(property("someNestedLevel1", "", object)), true, map[string]plan.PropertySchema{
						"someNestedLevel2": nested(optional(property("someNestedLevel2", "", object)), true, map[string]plan.PropertySchema{
							"someString":  optional(property("someString", "some string property", plan.PrimitiveTypeString)),
							"someInteger": optional(property("someInteger", "some integer property", plan.PrimitiveTypeInteger)),
						}),
					}),
				}),
			}),
			track("Some Empty Track Event", "", false, nil),
			track("Some Empty Track Event With Additional Properties", "", true, nil),
			track("Some Open Track Event", "Declares properties and allows unplanned ones.\nUndeclared keys travel in AdditionalProperties.", true, map[string]plan.PropertySchema{
				"price":                optional(property("price", "", plan.PrimitiveTypeNumber)),
				"productId":            required(property("productId", "", plan.PrimitiveTypeString)),
				"productName":          optional(property("productName", "", plan.PrimitiveTypeString)),
				"additionalProperties": optional(property("additionalProperties", "", plan.PrimitiveTypeString)),
				"amount":               optional(property("amount", "amount as a number", plan.PrimitiveTypeNumber)),
			}),
			// Both names format to EventWithNameCamelCase; the rule key of the
			// "$" one sorts first, so it keeps the base names.
			track("eventWithNameCamelCase", "", false, map[string]plan.PropertySchema{
				"someString": optional(property("someString", "some string property", plan.PrimitiveTypeString)),
			}),
			track("$eventWithNameCamelCase$!", "", false, map[string]plan.PropertySchema{
				"someString": optional(property("someString", "some string property", plan.PrimitiveTypeString)),
			}),
			// Property names that are Go keywords or predeclared identifiers.
			track("Every Primitive Type", "", false, map[string]plan.PropertySchema{
				"string":  required(property("string", "", plan.PrimitiveTypeString)),
				"integer": optional(property("integer", "", plan.PrimitiveTypeInteger)),
				"number":  required(property("number", "", plan.PrimitiveTypeNumber)),
				"boolean": optional(property("boolean", "", plan.PrimitiveTypeBoolean)),
				"array":   required(property("array", "", plan.PrimitiveTypeArray)),
				"object":  optional(property("object", "", plan.PrimitiveTypeObject)),
				"any":     required(property("any", "")),
				"type":    optional(property("type", "", plan.PrimitiveTypeString)),
			}),
			track("Name Collisions", "", false, map[string]plan.PropertySchema{
				"toProperties": optional(property("toProperties", "", plan.PrimitiveTypeString)),
				"userId":       required(property("userId", "", plan.PrimitiveTypeString)),
				"user_id":      optional(property("user_id", "", plan.PrimitiveTypeString)),
				"1st_place":    optional(property("1st_place", "", plan.PrimitiveTypeInteger)),
				"amount":       required(property("amount", "amount as a string", plan.PrimitiveTypeString)),
			}),
			track("User", "", false, map[string]plan.PropertySchema{
				"user": optional(property("user", "", plan.PrimitiveTypeString)),
			}),
			// Multi-type, nullable and array shapes the reference plan lacks.
			// The providers keep item types only when array is listed first.
			track("Some Multi Type Event", "", false, map[string]plan.PropertySchema{
				"someMultiType": optional(property("someMultiType", "",
					plan.PrimitiveTypeString, plan.PrimitiveTypeInteger, plan.PrimitiveTypeNumber, plan.PrimitiveTypeBoolean,
					plan.PrimitiveTypeObject, plan.PrimitiveTypeArray, plan.PrimitiveTypeNull)),
				"stringOrStrings":          required(items(property("stringOrStrings", "", plan.PrimitiveTypeArray, plan.PrimitiveTypeString), plan.PrimitiveTypeString)),
				"someArrayOrNull":          optional(items(property("someArrayOrNull", "", plan.PrimitiveTypeArray, plan.PrimitiveTypeNull), plan.PrimitiveTypeString)),
				"someObjectOrNull":         required(property("someObjectOrNull", "", plan.PrimitiveTypeObject, plan.PrimitiveTypeNull)),
				"someArrayOfMultipleTypes": optional(items(property("someArrayOfMultipleTypes", "", plan.PrimitiveTypeArray), plan.PrimitiveTypeString, plan.PrimitiveTypeInteger, plan.PrimitiveTypeNumber, plan.PrimitiveTypeBoolean)),
				"matrix":                   required(items(property("matrix", "", plan.PrimitiveTypeArray), plan.PrimitiveTypeArray)),
			}),
			// Enum shapes the reference plan lacks. statusActive takes the type
			// name PropertyStatusActive first, so status's "active" constant
			// gets the suffix.
			track("Some Enum Event", "", false, map[string]plan.PropertySchema{
				"stage":        optional(enum(property("stage", "", plan.PrimitiveTypeString, plan.PrimitiveTypeNull), "draft", "live", nil)),
				"cleared":      optional(enum(property("cleared", "", plan.PrimitiveTypeString, plan.PrimitiveTypeNull), nil)),
				"mode":         required(enum(property("mode", "", plan.PrimitiveTypeString), "Active", "active", "values", "")),
				"status":       required(enum(property("status", "", plan.PrimitiveTypeString), "active", "inactive")),
				"statusActive": optional(property("statusActive", "", plan.PrimitiveTypeBoolean)),
			}),
			// Custom type shapes the reference plan lacks. grid's items are a
			// custom array type, whose alias resolves through another alias,
			// and items' an object custom type.
			track("Some Custom Type Event", "", false, map[string]plan.PropertySchema{
				"someObject":  required(property("someObject", "", someObjectType)),
				"userProfile": optional(property("userProfile", "", userProfile)),
				"grid":        optional(items(property("grid", "", plan.PrimitiveTypeArray), row)),
				"items":       optional(items(property("items", "", plan.PrimitiveTypeArray), someObjectType)),
				// With the empty open schema the providers attach here, address
				// keeps its Property type.
				"address": nested(optional(property("address", "", object)), true, nil),
			}),
			// Nested object shapes the reference plan lacks. The providers keep
			// a nested schema only when object is listed first. foo.barBaz and
			// fooBar.baz both concatenate to ...FooBarBaz.
			track("Some Nested Object Event", "", false, map[string]plan.PropertySchema{
				"address": nested(optional(property("address", "", object)), false, map[string]plan.PropertySchema{
					"street": required(property("street", "", plan.PrimitiveTypeString)),
				}),
				"location": nested(required(property("location", "", object, plan.PrimitiveTypeNull)), true, map[string]plan.PropertySchema{
					"lat": required(property("lat", "", plan.PrimitiveTypeNumber)),
				}),
				"contact": nested(optional(property("contact", "", object, plan.PrimitiveTypeString)), true, map[string]plan.PropertySchema{
					"email": required(property("email", "", plan.PrimitiveTypeString)),
				}),
				// The providers' empty open schema hoists nothing, so these keep
				// their Property types.
				"objectOrNull":   nested(optional(property("objectOrNull", "", object, plan.PrimitiveTypeNull)), true, nil),
				"objectOrString": nested(optional(property("objectOrString", "", object, plan.PrimitiveTypeString)), true, nil),
				"foo": nested(optional(property("foo", "", object)), true, map[string]plan.PropertySchema{
					"barBaz": nested(optional(property("barBaz", "", object)), false, nil),
				}),
				"fooBar": nested(optional(property("fooBar", "", object)), true, map[string]plan.PropertySchema{
					"baz": nested(optional(property("baz", "", object)), false, nil),
				}),
			}),
		},
	}
}

// nested gives ps a nested object schema.
func nested(ps plan.PropertySchema, open bool, properties map[string]plan.PropertySchema) plan.PropertySchema {
	ps.Schema = &plan.ObjectSchema{Properties: properties, AdditionalProperties: open}
	return ps
}

func track(name, description string, allowUnplanned bool, properties map[string]plan.PropertySchema) plan.EventRule {
	return plan.EventRule{
		Event:   plan.Event{EventType: plan.EventTypeTrack, Name: name, Description: description},
		Section: plan.IdentitySectionProperties,
		Schema:  plan.ObjectSchema{Properties: properties, AdditionalProperties: allowUnplanned},
	}
}

func property(name, description string, types ...plan.PropertyType) plan.Property {
	return plan.Property{Name: name, Description: description, Types: types}
}

func items(p plan.Property, types ...plan.PropertyType) plan.Property {
	p.ItemTypes = types
	return p
}

func enum(p plan.Property, values ...any) plan.Property {
	p.Config = &plan.PropertyConfig{Enum: values}
	return p
}

func required(p plan.Property) plan.PropertySchema {
	return plan.PropertySchema{Property: p, Required: true}
}

func optional(p plan.Property) plan.PropertySchema { return plan.PropertySchema{Property: p} }
