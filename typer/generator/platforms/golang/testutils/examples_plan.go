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

		// The spec's variant custom type: two match values in one case, and a
		// default schema.
		someVariantType = &plan.CustomType{
			Name: "SomeVariantType",
			Type: object,
			Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"someString":  required(property("someString", "some string property", plan.PrimitiveTypeString)),
				"someInteger": optional(property("someInteger", "some integer property", plan.PrimitiveTypeInteger)),
			}},
			Variants: []plan.Variant{{
				Type:          "discriminator",
				Discriminator: "someString",
				Cases: []plan.VariantCase{
					{DisplayName: "Case 1", Match: []any{"case_1"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
						"someInteger": required(property("someInteger", "some integer property", plan.PrimitiveTypeInteger)),
					}}},
					{DisplayName: "Case 2", Match: []any{"case_2_a", "case_2_b"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
						"someNumber": required(property("someNumber", "", plan.PrimitiveTypeNumber)),
					}}},
				},
				DefaultSchema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"someBoolean": optional(property("someBoolean", "some boolean property", plan.PrimitiveTypeBoolean)),
				}},
			}},
		}
		// A nullable enum discriminator: no case can match null, so Default
		// stays although the cases match every other value.
		release = &plan.CustomType{
			Name: "release",
			Type: object,
			Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"stage": required(enum(property("stage", "", plan.PrimitiveTypeString, plan.PrimitiveTypeNull), "draft", "live", nil)),
			}},
			Variants: []plan.Variant{{
				Type:          "discriminator",
				Discriminator: "stage",
				Cases: []plan.VariantCase{
					{Match: []any{"draft"}},
					{Match: []any{"live"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
						"url": required(property("url", "", plan.PrimitiveTypeString)),
					}}},
				},
			}},
		}
		// An optional discriminator on an open base, which adds an open Absent.
		shipping = &plan.CustomType{
			Name: "shipping",
			Type: object,
			Schema: &plan.ObjectSchema{AdditionalProperties: true, Properties: map[string]plan.PropertySchema{
				"address": required(property("address", "", object)),
				"method":  optional(property("method", "", plan.PrimitiveTypeString)),
			}},
			Variants: []plan.Variant{{
				Type:          "discriminator",
				Discriminator: "method",
				Cases: []plan.VariantCase{
					{DisplayName: "Express", Match: []any{"express"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
						"deadline": required(property("deadline", "", plan.PrimitiveTypeString)),
					}}},
				},
				DefaultSchema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"carrier": required(property("carrier", "", plan.PrimitiveTypeString)),
				}},
			}},
		}
		// The spec's variant rule.
		deviceEvent = track("Device Event", "", false, map[string]plan.PropertySchema{
			"deviceType": required(property("deviceType", "", plan.PrimitiveTypeString)),
		})
		// An open base with an open case and an open default schema. Both
		// match values name the case OnLine, so the second takes a suffix, and
		// a property named discriminator takes one from the method. Every
		// branch shares the struct the base hoists for meta, while each case
		// struct hoists its own for geo.
		openVariantEvent = track("Some Open Variant Event", "", true, map[string]plan.PropertySchema{
			"kind":          required(property("kind", "", plan.PrimitiveTypeString)),
			"discriminator": optional(property("discriminator", "", plan.PrimitiveTypeString)),
			"meta": nested(optional(property("meta", "", object)), false, map[string]plan.PropertySchema{
				"source": required(property("source", "", plan.PrimitiveTypeString)),
			}),
		})
	)
	deviceEvent.Variants = []plan.Variant{{
		Type:          "discriminator",
		Discriminator: "deviceType",
		Cases: []plan.VariantCase{
			{DisplayName: "Mobile", Match: []any{"mobile"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"osVersion":  required(property("osVersion", "", plan.PrimitiveTypeString)),
				"appVersion": required(property("appVersion", "", plan.PrimitiveTypeString)),
			}}},
			{DisplayName: "Desktop", Match: []any{"desktop"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
				"browser": required(property("browser", "", plan.PrimitiveTypeString)),
			}}},
		},
	}}
	openVariantEvent.Variants = []plan.Variant{{
		Type:          "discriminator",
		Discriminator: "kind",
		Cases: []plan.VariantCase{
			{DisplayName: "Online", Description: "Sold online.", Match: []any{"on-line", "on_line"}, Schema: plan.ObjectSchema{AdditionalProperties: true, Properties: map[string]plan.PropertySchema{
				"url": required(property("url", "", plan.PrimitiveTypeString)),
				"geo": nested(optional(property("geo", "", object)), false, map[string]plan.PropertySchema{
					"lat": required(property("lat", "", plan.PrimitiveTypeNumber)),
				}),
			}}},
		},
		DefaultSchema: &plan.ObjectSchema{AdditionalProperties: true, Properties: map[string]plan.PropertySchema{
			"note": optional(property("note", "", plan.PrimitiveTypeString)),
		}},
	}}

	// The spec's page and screen rules declare the same properties.
	product := map[string]plan.PropertySchema{
		"productId":   required(property("productId", "", plan.PrimitiveTypeString)),
		"productName": optional(property("productName", "", plan.PrimitiveTypeString)),
		"price":       optional(property("price", "", plan.PrimitiveTypeNumber)),
	}
	// The catalog rejects names on the other event types, but a plan can carry
	// them: these fixtures give each Event Types row and categoryMode a rule of
	// its own. Identify and group methods keep their names, so the named ones
	// take suffixes.
	var (
		identifyVariant = event(plan.EventTypeIdentify, "Variant", plan.IdentitySectionContextTraits, false, map[string]plan.PropertySchema{
			"accountType": required(property("accountType", "", plan.PrimitiveTypeString)),
			// ToTraits is reserved in every traits struct.
			"toTraits": optional(property("toTraits", "", plan.PrimitiveTypeString)),
		})
		groupVariant = event(plan.EventTypeGroup, "Variant", plan.IdentitySectionContextTraits, false, map[string]plan.PropertySchema{
			"tier": required(property("tier", "", plan.PrimitiveTypeString)),
		})
		// Every struct declares category, as the discriminator.
		pageCategoryVariant = event(plan.EventTypePage, "Category Variant", plan.IdentitySectionProperties, false, map[string]plan.PropertySchema{
			"category": required(property("category", "", plan.PrimitiveTypeString)),
		})
		// An open base whose closed case and Default each have a mode of their
		// own: a branch is open only when its base and its own schema are.
		pageKindVariant = event(plan.EventTypePage, "Kind Variant", plan.IdentitySectionProperties, true, map[string]plan.PropertySchema{
			"kind": required(property("kind", "", plan.PrimitiveTypeString)),
		})
		screenVariant = event(plan.EventTypeScreen, "Variant", plan.IdentitySectionProperties, false, map[string]plan.PropertySchema{
			"screenType": required(property("screenType", "", plan.PrimitiveTypeString)),
		})
	)
	identifyVariant.Variants = []plan.Variant{{Type: "discriminator", Discriminator: "accountType", Cases: []plan.VariantCase{
		{Match: []any{"business"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"companyName": required(property("companyName", "", plan.PrimitiveTypeString)),
		}}},
	}}}
	groupVariant.Variants = []plan.Variant{{Type: "discriminator", Discriminator: "tier", Cases: []plan.VariantCase{
		{Match: []any{"enterprise"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"employeeCount": required(property("employeeCount", "", plan.PrimitiveTypeInteger)),
		}}},
	}}}
	pageCategoryVariant.Variants = []plan.Variant{{Type: "discriminator", Discriminator: "category", Cases: []plan.VariantCase{
		{Match: []any{"news"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"headline": required(property("headline", "", plan.PrimitiveTypeString)),
		}}},
	}}}
	pageKindVariant.Variants = []plan.Variant{{Type: "discriminator", Discriminator: "kind", Cases: []plan.VariantCase{
		{Match: []any{"article"}, Schema: plan.ObjectSchema{AdditionalProperties: true, Properties: map[string]plan.PropertySchema{
			"author": optional(property("author", "", plan.PrimitiveTypeString)),
		}}},
		{Match: []any{"gallery"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"images": optional(property("images", "", plan.PrimitiveTypeInteger)),
		}}},
	}}}
	screenVariant.Variants = []plan.Variant{{Type: "discriminator", Discriminator: "screenType", Cases: []plan.VariantCase{
		{Match: []any{"product"}, Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"productId": required(property("productId", "", plan.PrimitiveTypeString)),
		}}},
	}}}

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
				// Without a nested schema here, address keeps its Property type.
				"address": optional(property("address", "", object)),
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
				"foo": nested(optional(property("foo", "", object)), true, map[string]plan.PropertySchema{
					"barBaz": nested(optional(property("barBaz", "", object)), false, nil),
				}),
				"fooBar": nested(optional(property("fooBar", "", object)), true, map[string]plan.PropertySchema{
					"baz": nested(optional(property("baz", "", object)), false, nil),
				}),
			}),
			deviceEvent,
			openVariantEvent,
			// Fields that hold variants: shipping is required, and releases
			// an array of variant items.
			track("Some Variant Event", "", false, map[string]plan.PropertySchema{
				"someVariantType": optional(property("someVariantType", "", someVariantType)),
				"shipping":        required(property("shipping", "", shipping)),
				"releases":        optional(items(property("releases", "", plan.PrimitiveTypeArray), release)),
			}),
			// The spec's identify, group, page and screen rules. The page rule is
			// open, so WithCategory may add a category; the screen rule is closed.
			described(event(plan.EventTypeIdentify, "", plan.IdentitySectionTraits, false, map[string]plan.PropertySchema{
				"userName": optional(property("userName", "", plan.PrimitiveTypeString)),
				"email":    required(property("email", "", plan.PrimitiveTypeString)),
				"plan":     optional(property("plan", "", plan.PrimitiveTypeString)),
			}), "Identifies a user with their traits."),
			described(event(plan.EventTypeGroup, "", plan.IdentitySectionTraits, false, map[string]plan.PropertySchema{
				"companyName":   required(property("companyName", "", plan.PrimitiveTypeString)),
				"industry":      optional(property("industry", "", plan.PrimitiveTypeString)),
				"employeeCount": optional(property("employeeCount", "", plan.PrimitiveTypeInteger)),
			}), "Associates user with their company."),
			described(event(plan.EventTypePage, "", plan.IdentitySectionProperties, true, product), "User viewed a product page."),
			described(event(plan.EventTypeScreen, "", plan.IdentitySectionProperties, false, product), "User viewed a product screen."),
			identifyVariant,
			event(plan.EventTypeIdentify, "Open", plan.IdentitySectionTraits, true, nil),
			groupVariant,
			event(plan.EventTypeGroup, "Open", plan.IdentitySectionTraits, true, nil),
			event(plan.EventTypePage, "Category", plan.IdentitySectionProperties, false, map[string]plan.PropertySchema{
				"category": optional(property("category", "", plan.PrimitiveTypeString)),
			}),
			pageCategoryVariant,
			pageKindVariant,
			event(plan.EventTypePage, "Open", plan.IdentitySectionProperties, true, nil),
			event(plan.EventTypePage, "Empty", plan.IdentitySectionProperties, false, nil),
			screenVariant,
			event(plan.EventTypeScreen, "Open", plan.IdentitySectionProperties, true, nil),
			event(plan.EventTypeScreen, "Empty", plan.IdentitySectionProperties, false, nil),
		},
	}
}

func event(eventType plan.EventType, name string, section plan.IdentitySection, allowUnplanned bool, properties map[string]plan.PropertySchema) plan.EventRule {
	return plan.EventRule{
		Event:   plan.Event{EventType: eventType, Name: name},
		Section: section,
		Schema:  plan.ObjectSchema{Properties: properties, AdditionalProperties: allowUnplanned},
	}
}

func described(r plan.EventRule, description string) plan.EventRule {
	r.Event.Description = description
	return r
}

// nested gives ps a nested object schema.
func nested(ps plan.PropertySchema, open bool, properties map[string]plan.PropertySchema) plan.PropertySchema {
	ps.Schema = &plan.ObjectSchema{Properties: properties, AdditionalProperties: open}
	return ps
}

func track(name, description string, allowUnplanned bool, properties map[string]plan.PropertySchema) plan.EventRule {
	return described(event(plan.EventTypeTrack, name, plan.IdentitySectionProperties, allowUnplanned, properties), description)
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
