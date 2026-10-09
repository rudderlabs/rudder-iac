package testutils

import "github.com/rudderlabs/rudder-iac/typer/plan"

// GetExamplesTrackingPlan returns the Go generation spec's input examples plus
// the shapes the shared reference plan lacks, so the examples golden pins how
// each of them is generated.
func GetExamplesTrackingPlan() *plan.TrackingPlan {
	return &plan.TrackingPlan{
		Name: "Examples Plan",
		// No URL and version 0, as for a plan generated with --local.
		Metadata: plan.PlanMetadata{TrackingPlanID: "plan_examples"},
		Rules: []plan.EventRule{
			track("Some Track Event", "This is a track event for testing.", false, map[string]plan.PropertySchema{
				"someString":  required(property("someString", "some string property", plan.PrimitiveTypeString)),
				"someInteger": required(property("someInteger", "some integer property", plan.PrimitiveTypeInteger)),
				"someBoolean": optional(property("someBoolean", "some boolean property", plan.PrimitiveTypeBoolean)),
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
		},
	}
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

func required(p plan.Property) plan.PropertySchema {
	return plan.PropertySchema{Property: p, Required: true}
}

func optional(p plan.Property) plan.PropertySchema { return plan.PropertySchema{Property: p} }
