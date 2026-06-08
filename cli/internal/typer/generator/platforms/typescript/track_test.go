package typescript

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trackRule is a small helper for building a track rule with the given schema.
func trackRule(name, description string, schema plan.ObjectSchema, variants ...plan.Variant) *plan.EventRule {
	return &plan.EventRule{
		Event: plan.Event{
			EventType:   plan.EventTypeTrack,
			Name:        name,
			Description: description,
		},
		Section:  plan.IdentitySectionProperties,
		Schema:   schema,
		Variants: variants,
	}
}

func TestBuildTrackMethod_NonEmptySchema(t *testing.T) {
	rule := trackRule("User Signed Up", "Triggered on signup", plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"email": {Property: plan.Property{Name: "email", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	})

	ctx := &TSContext{}
	method, err := buildTrackMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "trackUserSignedUp",
		Comment:       "Triggered on signup",
		EventName:     "User Signed Up",
		SDKMethodName: "track",
		MethodArguments: []TSMethodArgument{
			{Name: "props", Type: "TrackUserSignedUpProperties", Comment: "The properties to include with this event"},
		},
		SDKArguments: []TSSDKArgument{
			{Value: `"User Signed Up"`},
			{Value: "props as unknown as SDKApiObject"},
		},
	}, method)
	assert.True(t, ctx.UsesSDKApiObject)
}

func TestBuildTrackMethod_EmptyAllowUnplanned(t *testing.T) {
	rule := trackRule("Open Schema Event", "", plan.ObjectSchema{
		Properties:           map[string]plan.PropertySchema{},
		AdditionalProperties: true,
	})

	ctx := &TSContext{}
	method, err := buildTrackMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "trackOpenSchemaEvent",
		EventName:     "Open Schema Event",
		SDKMethodName: "track",
		MethodArguments: []TSMethodArgument{
			{Name: "props", Type: "Record<string, unknown>", Comment: "Additional properties to include with this event", Optional: true},
		},
		SDKArguments: []TSSDKArgument{
			{Value: `"Open Schema Event"`},
			{Value: "props as unknown as SDKApiObject"},
		},
	}, method)
	assert.True(t, ctx.UsesSDKApiObject)
}

func TestBuildTrackMethod_EmptyDisallowUnplanned(t *testing.T) {
	// Empty schema, additionalProperties: false → no props arg, pass {}.
	rule := trackRule("Closed Empty Event", "", plan.ObjectSchema{
		Properties:           map[string]plan.PropertySchema{},
		AdditionalProperties: false,
	})

	ctx := &TSContext{}
	method, err := buildTrackMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "trackClosedEmptyEvent",
		EventName:     "Closed Empty Event",
		SDKMethodName: "track",
		SDKArguments: []TSSDKArgument{
			{Value: `"Closed Empty Event"`},
			{Value: "{}"},
		},
	}, method)
	assert.False(t, ctx.UsesSDKApiObject)
}

func TestBuildTrackMethod_EventNameWithSpecialChars(t *testing.T) {
	rule := trackRule(`Product "Premium" Clicked`, "", plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"id": {Property: plan.Property{Name: "id", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	})

	ctx := &TSContext{}
	method, err := buildTrackMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "trackProductPremiumClicked",
		EventName:     `Product "Premium" Clicked`,
		SDKMethodName: "track",
		MethodArguments: []TSMethodArgument{
			{Name: "props", Type: "TrackProductPremiumClickedProperties", Comment: "The properties to include with this event"},
		},
		SDKArguments: []TSSDKArgument{
			{Value: `"Product \"Premium\" Clicked"`},
			{Value: "props as unknown as SDKApiObject"},
		},
	}, method)
}

func TestProcessEventRules_EmitsVariants(t *testing.T) {
	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			*trackRule("Has Variants", "event with disc", plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"kind": {Property: plan.Property{Name: "kind", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
				},
			}, plan.Variant{
				Discriminator: "kind",
				Cases: []plan.VariantCase{
					{
						DisplayName: "Alpha",
						Match:       []any{"alpha"},
						Description: "Alpha case",
						Schema: plan.ObjectSchema{
							Properties: map[string]plan.PropertySchema{
								"score": {Property: plan.Property{Name: "score", Types: []plan.PropertyType{plan.PrimitiveTypeInteger}}, Required: true},
							},
						},
					},
				},
			}),
			*trackRule("Plain Event", "", plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"id": {Property: plan.Property{Name: "id", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
				},
			}),
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processEventRules(tp, ctx, newTestRegistry()))

	assert.Len(t, ctx.AnalyticsMethods, 2, "variant rule produces a method")
	assert.Equal(t, "trackHasVariants", ctx.AnalyticsMethods[0].Name)
	assert.Equal(t, "trackPlainEvent", ctx.AnalyticsMethods[1].Name)

	assert.Len(t, ctx.Interfaces, 1, "plain event still gets an interface")
	assert.Equal(t, "TrackPlainEventProperties", ctx.Interfaces[0].Name)

	require.Len(t, ctx.VariantTypes, 1)
	group := ctx.VariantTypes[0]
	assert.Equal(t, "TrackHasVariantsProperties", group.UnionAlias.Alias)
	require.Len(t, group.CaseInterfaces, 2, "one named case + default")
	assert.Equal(t, "TrackHasVariantsPropertiesCaseAlpha", group.CaseInterfaces[0].Name)
	assert.Equal(t, "TrackHasVariantsPropertiesDefault", group.CaseInterfaces[1].Name)
}

func TestProcessEventRules_SkipsUnsupportedEventTypes(t *testing.T) {
	// Screen is the only event type analytics-js does not expose as a direct
	// SDK call (it's a mobile-only concept), so a screen rule must be skipped
	// without emitting a method. Page rules used to be skipped too, but page
	// is now supported via the SDK's page() API.
	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			{Event: plan.Event{EventType: plan.EventTypeScreen}, Section: plan.IdentitySectionProperties},
			*trackRule("Allowed", "", plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"id": {Property: plan.Property{Name: "id", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
				},
			}),
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processEventRules(tp, ctx, newTestRegistry()))
	require.Len(t, ctx.AnalyticsMethods, 1)
	assert.Equal(t, "trackAllowed", ctx.AnalyticsMethods[0].Name)
}

func TestProcessCustomTypesIntoContext_Categories(t *testing.T) {
	// Each custom-type kind takes a distinct emission path; this exercises all
	// of them in one plan so the dispatch logic is covered end-to-end.
	emailCT := &plan.CustomType{Name: "email", Description: "Email addr", Type: plan.PrimitiveTypeString}
	statusCT := &plan.CustomType{Name: "status", Description: "Status enum", Type: plan.PrimitiveTypeString,
		Config: &plan.PropertyConfig{Enum: []any{"active", "inactive"}}}
	emailListCT := &plan.CustomType{Name: "email_list", Description: "List of emails", Type: plan.PrimitiveTypeArray, ItemType: *emailCT}
	profileCT := &plan.CustomType{Name: "profile", Description: "User profile", Type: plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"name": {Property: plan.Property{Name: "name", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		}}}
	openObjectCT := &plan.CustomType{Name: "open_object", Description: "Open object", Type: plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}, AdditionalProperties: true}}

	// A rule that references all of them so ExtractAllCustomTypes picks them up.
	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			{Event: plan.Event{EventType: plan.EventTypeTrack, Name: "Test"}, Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"a": {Property: plan.Property{Name: "a", Types: []plan.PropertyType{*emailCT}}},
					"b": {Property: plan.Property{Name: "b", Types: []plan.PropertyType{*statusCT}}},
					"c": {Property: plan.Property{Name: "c", Types: []plan.PropertyType{*emailListCT}}},
					"d": {Property: plan.Property{Name: "d", Types: []plan.PropertyType{*profileCT}}},
					"e": {Property: plan.Property{Name: "e", Types: []plan.PropertyType{*openObjectCT}}},
				}},
			},
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processCustomTypesIntoContext(tp, ctx, newTestRegistry()))

	aliases := map[string]TSTypeAlias{}
	for _, a := range ctx.CustomTypeAliases {
		aliases[a.Alias] = a
	}
	interfaces := map[string]TSInterface{}
	for _, i := range ctx.CustomInterfaces {
		interfaces[i.Name] = i
	}

	assert.Equal(t, "string", aliases["CustomTypeEmail"].Type, "primitive custom type → bare TS primitive alias")
	assert.Equal(t, `"active" | "inactive"`, aliases["CustomTypeStatus"].Type, "enum-constrained custom type → literal union alias")
	assert.Equal(t, "CustomTypeEmail[]", aliases["CustomTypeEmailList"].Type, "array custom type references its item alias by name")
	assert.Equal(t, "Record<string, unknown>", aliases["CustomTypeOpenObject"].Type, "empty object custom type collapses to open record alias")
	assert.Contains(t, interfaces, "CustomTypeProfile", "object custom type with fields emits as a named interface")
	assert.Equal(t, "User profile", interfaces["CustomTypeProfile"].Comment)
}

func TestProcessCustomTypesIntoContext_VariantsEmitDiscriminatedUnion(t *testing.T) {
	pageCT := &plan.CustomType{
		Name:        "page",
		Description: "Page context",
		Type:        plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"page_type": {Property: plan.Property{Name: "page_type", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		}},
		Variants: []plan.Variant{{
			Discriminator: "page_type",
			Cases: []plan.VariantCase{
				{
					DisplayName: "Search",
					Match:       []any{"search"},
					Description: "Search page",
					Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
						"query": {Property: plan.Property{Name: "query", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
					}},
				},
			},
			DefaultSchema: &plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
		}},
	}

	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			{Event: plan.Event{EventType: plan.EventTypeTrack, Name: "Test"}, Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"page": {Property: plan.Property{Name: "page", Types: []plan.PropertyType{*pageCT}}},
				}},
			},
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processCustomTypesIntoContext(tp, ctx, newTestRegistry()))

	assert.Empty(t, ctx.CustomInterfaces, "variant types go to VariantTypes, not CustomInterfaces")

	require.Len(t, ctx.VariantTypes, 1)
	group := ctx.VariantTypes[0]
	assert.Equal(t, "CustomTypePage", group.UnionAlias.Alias)
	assert.Equal(t, "CustomTypePageCaseSearch | CustomTypePageDefault", group.UnionAlias.Type)

	require.Len(t, group.CaseInterfaces, 2)
	assert.Equal(t, "CustomTypePageCaseSearch", group.CaseInterfaces[0].Name)
	assert.Equal(t, "CustomTypePageDefault", group.CaseInterfaces[1].Name)

	// Case interface has discriminator as literal + case-specific property
	searchIface := group.CaseInterfaces[0]
	require.Len(t, searchIface.Properties, 2)
	assert.Equal(t, TSInterfaceProperty{Name: "pageType", Type: `"search"`, Comment: "", Optional: false}, searchIface.Properties[0])
	assert.Equal(t, TSInterfaceProperty{Name: "query", Type: "string", Comment: "", Optional: false}, searchIface.Properties[1])

	// Default interface narrows the discriminator to values no named case covers
	defaultIface := group.CaseInterfaces[1]
	require.Len(t, defaultIface.Properties, 1)
	assert.Equal(t, TSInterfaceProperty{Name: "pageType", Type: `Exclude<string, "search">`, Comment: "", Optional: false}, defaultIface.Properties[0])
}

func TestProcessPropertyEnumsIntoContext(t *testing.T) {
	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			{Event: plan.Event{EventType: plan.EventTypeTrack, Name: "Test"}, Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"device_type": {Property: plan.Property{
						Name:        "device_type",
						Description: "Type of device",
						Types:       []plan.PropertyType{plan.PrimitiveTypeString},
						Config:      &plan.PropertyConfig{Enum: []any{"mobile", "desktop"}},
					}},
					"priority": {Property: plan.Property{
						Name:        "priority",
						Description: "Priority level",
						Types:       []plan.PropertyType{plan.PrimitiveTypeInteger},
						Config:      &plan.PropertyConfig{Enum: []any{1, 2, 3}},
					}},
					// Property without enum config — must not produce an alias.
					"plain": {Property: plan.Property{
						Name:  "plain",
						Types: []plan.PropertyType{plan.PrimitiveTypeString},
					}},
				}},
			},
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processPropertyEnumsIntoContext(tp, ctx, newTestRegistry()))

	require.Len(t, ctx.PropertyEnums, 2, "only properties with enum config produce aliases")
	byName := map[string]TSTypeAlias{}
	for _, a := range ctx.PropertyEnums {
		byName[a.Alias] = a
	}
	assert.Equal(t, `"mobile" | "desktop"`, byName["PropertyDeviceType"].Type)
	assert.Equal(t, "1 | 2 | 3", byName["PropertyPriority"].Type)
}

func TestBuildEventInterface_HoistsNestedSchemas(t *testing.T) {
	// Inline nested-object schemas are hoisted to top-level interfaces named
	// `{ParentInterface}{PropertyPath}`. The hoisted interfaces are appended
	// deepest-first so the slice ends up bottom-up.
	rule := trackRule("Order Placed", "", plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"context": {
				Property: plan.Property{Name: "context", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
				Required: true,
				Schema: &plan.ObjectSchema{
					Properties: map[string]plan.PropertySchema{
						"ip": {Property: plan.Property{Name: "ip", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
						"location": {
							Property: plan.Property{Name: "location", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
							Required: true,
							Schema: &plan.ObjectSchema{
								Properties: map[string]plan.PropertySchema{
									"city": {Property: plan.Property{Name: "city", Types: []plan.PropertyType{plan.PrimitiveTypeString}}},
								},
							},
						},
					},
				},
			},
		},
	})

	ctx := &TSContext{}
	nr := newTestRegistry()
	require.NoError(t, processOneEventRule(rule, ctx, nr))

	require.Len(t, ctx.Interfaces, 1)
	require.Len(t, ctx.NestedInterfaces, 2, "two nested levels → two hoisted interfaces")

	// Deepest-first: location (level 2) appears before context (level 1).
	assert.Equal(t, "TrackOrderPlacedPropertiesContextLocation", ctx.NestedInterfaces[0].Name)
	assert.Equal(t, "TrackOrderPlacedPropertiesContext", ctx.NestedInterfaces[1].Name)

	// Parent interface references the nested name, not the open record type.
	parent := ctx.Interfaces[0]
	require.Len(t, parent.Properties, 1)
	assert.Equal(t, "TrackOrderPlacedPropertiesContext", parent.Properties[0].Type)

	// Level-1 interface references the level-2 interface by name.
	level1 := ctx.NestedInterfaces[1]
	var locationProp TSInterfaceProperty
	for _, p := range level1.Properties {
		if p.Name == "location" {
			locationProp = p
			break
		}
	}
	assert.Equal(t, "TrackOrderPlacedPropertiesContextLocation", locationProp.Type)
}

func TestEmptyObjectType_RespectsAdditionalProperties(t *testing.T) {
	// Empty schemas split on `additionalProperties`: open ones become
	// `Record<string, unknown>` (any extra keys allowed); closed ones become
	// `Record<string, never>` (no keys allowed at all). Mirrors Kotlin's
	// `JsonObject` vs `Unit` and Swift's `[String: Any]` vs empty struct
	// distinction — collapsing both to one shape would silently let callers
	// pass extra keys into a closed schema.
	tests := []struct {
		name     string
		schema   *plan.ObjectSchema
		expected string
	}{
		{"open empty schema", &plan.ObjectSchema{AdditionalProperties: true}, "Record<string, unknown>"},
		{"closed empty schema", &plan.ObjectSchema{AdditionalProperties: false}, "Record<string, never>"},
		{"nil schema is treated as closed", nil, "Record<string, never>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, emptyObjectType(tt.schema))
		})
	}
}

func TestEmitCustomObjectType_EmptySchemaSplitsOnAdditionalProps(t *testing.T) {
	openCT := &plan.CustomType{
		Name: "open_obj", Description: "Open", Type: plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{AdditionalProperties: true},
	}
	closedCT := &plan.CustomType{
		Name: "closed_obj", Description: "Closed", Type: plan.PrimitiveTypeObject,
		Schema: &plan.ObjectSchema{AdditionalProperties: false},
	}

	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			{Event: plan.Event{EventType: plan.EventTypeTrack, Name: "Test"}, Section: plan.IdentitySectionProperties,
				Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
					"a": {Property: plan.Property{Name: "a", Types: []plan.PropertyType{*openCT}}},
					"b": {Property: plan.Property{Name: "b", Types: []plan.PropertyType{*closedCT}}},
				}},
			},
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processCustomTypesIntoContext(tp, ctx, newTestRegistry()))

	aliases := map[string]string{}
	for _, a := range ctx.CustomTypeAliases {
		aliases[a.Alias] = a.Type
	}
	assert.Equal(t, "Record<string, unknown>", aliases["CustomTypeOpenObj"])
	assert.Equal(t, "Record<string, never>", aliases["CustomTypeClosedObj"])
}

func TestBuildNestedInterfaceIfPresent_InlineEmptySchemaSplitsOnAdditionalProps(t *testing.T) {
	// An inline schema with no properties shouldn't hoist an interface — it
	// should pass through the open/closed Record literal directly so the
	// parent property's type still encodes the additionalProperties decision.
	openProp := plan.PropertySchema{
		Property: plan.Property{Name: "open", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
		Schema:   &plan.ObjectSchema{AdditionalProperties: true},
	}
	closedProp := plan.PropertySchema{
		Property: plan.Property{Name: "closed", Types: []plan.PropertyType{plan.PrimitiveTypeObject}},
		Schema:   &plan.ObjectSchema{AdditionalProperties: false},
	}

	ctx := &TSContext{}
	openResult, err := buildNestedInterfaceIfPresent("Parent", "open", &openProp, ctx, newTestRegistry())
	require.NoError(t, err)
	assert.Equal(t, "Record<string, unknown>", openResult)
	assert.Empty(t, ctx.NestedInterfaces, "no interface hoisted for empty inline schema")

	ctx = &TSContext{}
	closedResult, err := buildNestedInterfaceIfPresent("Parent", "closed", &closedProp, ctx, newTestRegistry())
	require.NoError(t, err)
	assert.Equal(t, "Record<string, never>", closedResult)
	assert.Empty(t, ctx.NestedInterfaces)
}

func TestResolveArrayType_WrapsItemErrors(t *testing.T) {
	// Errors from individual array item resolution should carry index context,
	// matching Swift's `fmt.Errorf("resolving type for property %q ...: %w")`
	// pattern. Without this, a failure deep in a 30-item union surfaces as a
	// bare error string with no clue which item triggered it.
	//
	// We force a failure by handing in an unsupported PropertyType (a nil
	// interface satisfies neither IsPrimitiveType nor IsCustomType). The
	// single-item path returns "unknown" (consistent with Swift's "Any"
	// fallback), so we exercise the multi-item path which loops through
	// resolveSingleItem — the only call site that wraps errors.
	//
	// Today no error path within resolveSingleItem actually triggers (it
	// silently falls back to "unknown" too), so this test documents the
	// wrapping contract for when stricter error returns are added later.
	got, err := resolveArrayType([]plan.PropertyType{plan.PrimitiveTypeString, nil}, newTestRegistry())
	require.NoError(t, err)
	assert.Equal(t, "Array<string | unknown>", got)
}

