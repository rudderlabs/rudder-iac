package typescript

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func groupRule(description string, section plan.IdentitySection, schema plan.ObjectSchema) *plan.EventRule {
	return &plan.EventRule{
		Event:   plan.Event{EventType: plan.EventTypeGroup, Description: description},
		Section: section,
		Schema:  schema,
	}
}

func pageRule(name, description string, schema plan.ObjectSchema) *plan.EventRule {
	return &plan.EventRule{
		Event:   plan.Event{EventType: plan.EventTypePage, Name: name, Description: description},
		Section: plan.IdentitySectionProperties,
		Schema:  schema,
	}
}

// ===== Identify =====

func TestBuildIdentifyMethod_StrictCast(t *testing.T) {
	rule := &plan.EventRule{
		Event:   plan.Event{EventType: plan.EventTypeIdentify, Description: "User identification"},
		Section: plan.IdentitySectionTraits,
		Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"email": {Property: plan.Property{Name: "email", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		}},
	}

	ctx := &TSContext{}
	method, err := buildIdentifyMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "identify",
		Comment:       "User identification",
		SDKMethodName: "identify",
		Overloads: []TSOverloadSignature{
			{Arguments: []TSMethodArgument{
				{Name: "userId", Type: "string"},
				{Name: "traits", Type: "IdentifyTraits", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
			{Arguments: []TSMethodArgument{
				{Name: "traits", Type: "IdentifyTraits", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
		},
		MethodArguments: []TSMethodArgument{
			{Name: "userIdOrTraits", Type: "string | IdentifyTraits", Optional: true},
			{Name: "traitsOrOptions", Type: "IdentifyTraits | ApiOptions", Optional: true},
			{Name: "optionsOrCallback", Type: "ApiOptions | ApiCallback", Optional: true},
			{Name: "callback", Type: "ApiCallback", Optional: true},
		},
		DispatcherBranches: []TSDispatcherBranch{
			{
				Condition: `typeof userIdOrTraits === "string"`,
				SDKArguments: []TSSDKArgument{
					{Value: "userIdOrTraits"},
					{Value: "traitsOrOptions as unknown as SDKIdentifyTraits"},
					{Value: "this.withRudderTyperContext(optionsOrCallback as ApiOptions | undefined)"},
					{Value: "callback"},
				},
			},
			{
				SDKArguments: []TSSDKArgument{
					{Value: "userIdOrTraits as unknown as SDKIdentifyTraits"},
					{Value: "this.withRudderTyperContext(traitsOrOptions as ApiOptions | undefined)"},
					{Value: "optionsOrCallback as ApiCallback | undefined"},
				},
			},
		},
	}, method)

	assert.True(t, ctx.UsesSDKIdentifyTraits)
	assert.True(t, ctx.UsesApiCallback)
}

func TestBuildIdentifyMethod_EmptySchema_NoTraitsType(t *testing.T) {
	rule := &plan.EventRule{
		Event:   plan.Event{EventType: plan.EventTypeIdentify},
		Section: plan.IdentitySectionTraits,
		Schema:  plan.ObjectSchema{Properties: map[string]plan.PropertySchema{}},
	}

	ctx := &TSContext{}
	method, err := buildIdentifyMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "identify",
		SDKMethodName: "identify",
		Overloads: []TSOverloadSignature{
			{Arguments: []TSMethodArgument{
				{Name: "userId", Type: "string"},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
		},
		MethodArguments: []TSMethodArgument{
			{Name: "userId", Type: "string"},
			{Name: "options", Type: "ApiOptions", Optional: true},
			{Name: "callback", Type: "ApiCallback", Optional: true},
		},
		DispatcherBranches: []TSDispatcherBranch{
			{SDKArguments: []TSSDKArgument{
				{Value: "userId"}, {Value: "undefined"},
				{Value: "this.withRudderTyperContext(options)"}, {Value: "callback"},
			}},
		},
	}, method)

	assert.False(t, ctx.UsesSDKIdentifyTraits)
	assert.True(t, ctx.UsesApiCallback)
}

func TestBuildIdentifyMethod_ContextTraitsRoutesToContext(t *testing.T) {
	rule := &plan.EventRule{
		Event:   plan.Event{EventType: plan.EventTypeIdentify, Description: "Identify with context traits"},
		Section: plan.IdentitySectionContextTraits,
		Schema: plan.ObjectSchema{Properties: map[string]plan.PropertySchema{
			"email": {Property: plan.Property{Name: "email", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		}},
	}

	ctx := &TSContext{}
	method, err := buildIdentifyMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, []TSDispatcherBranch{
		{
			Condition: `typeof userIdOrTraits === "string"`,
			SDKArguments: []TSSDKArgument{
				{Value: "userIdOrTraits"},
				{Value: "undefined"},
				{Value: "this.withRudderTyperContext(optionsOrCallback as ApiOptions | undefined, traitsOrOptions as unknown as Record<string, unknown>)"},
				{Value: "callback"},
			},
		},
		{
			SDKArguments: []TSSDKArgument{
				{Value: "undefined"},
				{Value: "this.withRudderTyperContext(traitsOrOptions as ApiOptions | undefined, userIdOrTraits as unknown as Record<string, unknown>)"},
				{Value: "optionsOrCallback as ApiCallback | undefined"},
			},
		},
	}, method.DispatcherBranches)

	assert.True(t, method.AddDataToContext)
	assert.False(t, ctx.UsesSDKIdentifyTraits)
}

// ===== Group =====

func TestBuildGroupMethod_EmitsOverloads(t *testing.T) {
	rule := groupRule("Group assoc", plan.IdentitySectionTraits, plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"plan": {Property: plan.Property{Name: "plan", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	})

	ctx := &TSContext{}
	method, err := buildGroupMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "group",
		Comment:       "Group assoc",
		SDKMethodName: "group",
		Overloads: []TSOverloadSignature{
			{Arguments: []TSMethodArgument{
				{Name: "groupId", Type: "string"},
				{Name: "traits", Type: "GroupTraits", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
			{Arguments: []TSMethodArgument{
				{Name: "traits", Type: "GroupTraits", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
		},
		MethodArguments: []TSMethodArgument{
			{Name: "groupIdOrTraits", Type: "string | GroupTraits", Optional: true},
			{Name: "traitsOrOptions", Type: "GroupTraits | ApiOptions", Optional: true},
			{Name: "optionsOrCallback", Type: "ApiOptions | ApiCallback", Optional: true},
			{Name: "callback", Type: "ApiCallback", Optional: true},
		},
		DispatcherBranches: []TSDispatcherBranch{
			{
				Condition: `typeof groupIdOrTraits === "string"`,
				SDKArguments: []TSSDKArgument{
					{Value: "groupIdOrTraits"},
					{Value: "traitsOrOptions as unknown as SDKIdentifyTraits"},
					{Value: "this.withRudderTyperContext(optionsOrCallback as ApiOptions | undefined)"},
					{Value: "callback"},
				},
			},
			{
				SDKArguments: []TSSDKArgument{
					{Value: "groupIdOrTraits as unknown as SDKIdentifyTraits"},
					{Value: "this.withRudderTyperContext(traitsOrOptions as ApiOptions | undefined)"},
					{Value: "optionsOrCallback as ApiCallback | undefined"},
				},
			},
		},
	}, method)

	assert.True(t, ctx.UsesSDKIdentifyTraits)
	assert.True(t, ctx.UsesApiCallback)
}

func TestBuildGroupMethod_EmptySchema_OmitsTraitsType(t *testing.T) {
	rule := groupRule("", plan.IdentitySectionTraits, plan.ObjectSchema{})

	ctx := &TSContext{}
	method, err := buildGroupMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "group",
		SDKMethodName: "group",
		Overloads: []TSOverloadSignature{
			{Arguments: []TSMethodArgument{
				{Name: "groupId", Type: "string"},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
		},
		MethodArguments: []TSMethodArgument{
			{Name: "groupId", Type: "string"},
			{Name: "options", Type: "ApiOptions", Optional: true},
			{Name: "callback", Type: "ApiCallback", Optional: true},
		},
		DispatcherBranches: []TSDispatcherBranch{
			{SDKArguments: []TSSDKArgument{
				{Value: "groupId"}, {Value: "undefined"},
				{Value: "this.withRudderTyperContext(options)"}, {Value: "callback"},
			}},
		},
	}, method)

	assert.False(t, ctx.UsesSDKIdentifyTraits)
	assert.True(t, ctx.UsesApiCallback)
}

func TestBuildGroupMethod_ContextTraitsRoutesToContext(t *testing.T) {
	rule := groupRule("", plan.IdentitySectionContextTraits, plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"tenant": {Property: plan.Property{Name: "tenant", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	})

	ctx := &TSContext{}
	method, err := buildGroupMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, []TSDispatcherBranch{
		{
			Condition: `typeof groupIdOrTraits === "string"`,
			SDKArguments: []TSSDKArgument{
				{Value: "groupIdOrTraits"},
				{Value: "undefined"},
				{Value: "this.withRudderTyperContext(optionsOrCallback as ApiOptions | undefined, traitsOrOptions as unknown as Record<string, unknown>)"},
				{Value: "callback"},
			},
		},
		{
			SDKArguments: []TSSDKArgument{
				{Value: "undefined"},
				{Value: "this.withRudderTyperContext(traitsOrOptions as ApiOptions | undefined, groupIdOrTraits as unknown as Record<string, unknown>)"},
				{Value: "optionsOrCallback as ApiCallback | undefined"},
			},
		},
	}, method.DispatcherBranches)

	assert.True(t, method.AddDataToContext)
	assert.False(t, ctx.UsesSDKIdentifyTraits)
}

// ===== Page =====

func TestBuildPageMethod_EmitsThreeOverloads(t *testing.T) {
	rule := pageRule("", "Page view event", plan.ObjectSchema{
		Properties: map[string]plan.PropertySchema{
			"url": {Property: plan.Property{Name: "url", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
		},
	})

	ctx := &TSContext{}
	method, err := buildPageMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, &TSAnalyticsMethod{
		Name:          "page",
		Comment:       "Page view event",
		SDKMethodName: "page",
		Overloads: []TSOverloadSignature{
			{Arguments: []TSMethodArgument{
				{Name: "category", Type: "string"},
				{Name: "name", Type: "string"},
				{Name: "properties", Type: "PageProperties", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
			{Arguments: []TSMethodArgument{
				{Name: "name", Type: "string"},
				{Name: "properties", Type: "PageProperties", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
			{Arguments: []TSMethodArgument{
				{Name: "properties", Type: "PageProperties", Optional: true},
				{Name: "options", Type: "ApiOptions", Optional: true},
				{Name: "callback", Type: "ApiCallback", Optional: true},
			}},
		},
		MethodArguments: []TSMethodArgument{
			{Name: "arg0", Type: "string | PageProperties", Optional: true},
			{Name: "arg1", Type: "string | PageProperties | ApiOptions", Optional: true},
			{Name: "arg2", Type: "PageProperties | ApiOptions | ApiCallback", Optional: true},
			{Name: "arg3", Type: "ApiOptions | ApiCallback", Optional: true},
			{Name: "arg4", Type: "ApiCallback", Optional: true},
		},
		DispatcherBranches: []TSDispatcherBranch{
			{
				Condition: `typeof arg0 === "string" && typeof arg1 === "string"`,
				SDKArguments: []TSSDKArgument{
					{Value: "arg0"}, {Value: "arg1"}, {Value: "arg2 as unknown as SDKApiObject"},
					{Value: "this.withRudderTyperContext(arg3 as ApiOptions | undefined)"}, {Value: "arg4"},
				},
			},
			{
				Condition: `typeof arg0 === "string"`,
				SDKArguments: []TSSDKArgument{
					{Value: "arg0"}, {Value: "arg1 as unknown as SDKApiObject"},
					{Value: "this.withRudderTyperContext(arg2 as ApiOptions | undefined)"}, {Value: "arg3 as ApiCallback | undefined"},
				},
			},
			{
				SDKArguments: []TSSDKArgument{
					{Value: "arg0 as unknown as SDKApiObject"},
					{Value: "this.withRudderTyperContext(arg1 as ApiOptions | undefined)"}, {Value: "arg2 as ApiCallback | undefined"},
				},
			},
		},
	}, method)

	assert.True(t, ctx.UsesSDKApiObject)
	assert.True(t, ctx.UsesApiCallback)
}

func TestBuildPageMethod_EmptyAllowUnplanned(t *testing.T) {
	rule := pageRule("Open Page", "", plan.ObjectSchema{
		Properties:           map[string]plan.PropertySchema{},
		AdditionalProperties: true,
	})

	ctx := &TSContext{}
	method, err := buildPageMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	require.Len(t, method.Overloads, 3)
	assert.Equal(t, "Record<string, unknown>", method.Overloads[0].Arguments[2].Type)
	assert.Equal(t, "Record<string, unknown>", method.Overloads[1].Arguments[1].Type)
	assert.Equal(t, "Record<string, unknown>", method.Overloads[2].Arguments[0].Type)
	assert.True(t, ctx.UsesSDKApiObject)
}

func TestBuildPageMethod_EmptyDisallowUnplanned(t *testing.T) {
	rule := pageRule("Closed Page", "", plan.ObjectSchema{
		Properties:           map[string]plan.PropertySchema{},
		AdditionalProperties: false,
	})

	ctx := &TSContext{}
	method, err := buildPageMethod(rule, ctx, newTestRegistry())
	require.NoError(t, err)

	assert.Equal(t, []TSOverloadSignature{
		{Arguments: []TSMethodArgument{
			{Name: "category", Type: "string"}, {Name: "name", Type: "string"},
			{Name: "options", Type: "ApiOptions", Optional: true}, {Name: "callback", Type: "ApiCallback", Optional: true},
		}},
		{Arguments: []TSMethodArgument{
			{Name: "name", Type: "string"},
			{Name: "options", Type: "ApiOptions", Optional: true}, {Name: "callback", Type: "ApiCallback", Optional: true},
		}},
		{Arguments: []TSMethodArgument{
			{Name: "options", Type: "ApiOptions", Optional: true}, {Name: "callback", Type: "ApiCallback", Optional: true},
		}},
	}, method.Overloads)

	assert.Equal(t, []TSDispatcherBranch{
		{
			Condition: `typeof arg0 === "string" && typeof arg1 === "string"`,
			SDKArguments: []TSSDKArgument{
				{Value: "arg0"}, {Value: "arg1"}, {Value: "{}"},
				{Value: "this.withRudderTyperContext(arg2 as ApiOptions | undefined)"}, {Value: "arg3"},
			},
		},
		{
			Condition: `typeof arg0 === "string"`,
			SDKArguments: []TSSDKArgument{
				{Value: "arg0"}, {Value: "{}"},
				{Value: "this.withRudderTyperContext(arg1 as ApiOptions | undefined)"}, {Value: "arg2 as ApiCallback | undefined"},
			},
		},
		{
			SDKArguments: []TSSDKArgument{
				{Value: "{}"},
				{Value: "this.withRudderTyperContext(arg0 as ApiOptions | undefined)"}, {Value: "arg1 as ApiCallback | undefined"},
			},
		},
	}, method.DispatcherBranches)

	assert.False(t, ctx.UsesSDKApiObject, "closed empty schema has no props cast")
}

// ===== Screen / Integration =====

func TestProcessEventRules_SkipsScreenEvents(t *testing.T) {
	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			{Event: plan.Event{EventType: plan.EventTypeScreen, Name: "Home"}, Section: plan.IdentitySectionProperties},
			*groupRule("Group", plan.IdentitySectionTraits, plan.ObjectSchema{}),
			*pageRule("About", "", plan.ObjectSchema{}),
			*trackRule("Allowed", "", plan.ObjectSchema{}),
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processEventRules(tp, ctx, newTestRegistry()))
	require.Len(t, ctx.AnalyticsMethods, 3, "screen rule skipped; group, page, and track survive")
	names := []string{}
	for _, m := range ctx.AnalyticsMethods {
		names = append(names, m.Name)
	}
	assert.ElementsMatch(t, []string{"group", "page", "trackAllowed"}, names)
}

func TestProcessEventRules_GeneratesSingletonGroupAndPageInterfaces(t *testing.T) {
	tp := &plan.TrackingPlan{
		Rules: []plan.EventRule{
			*groupRule("Group with traits", plan.IdentitySectionTraits, plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"plan": {Property: plan.Property{Name: "plan", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
				},
			}),
			*pageRule("Checkout", "Checkout page", plan.ObjectSchema{
				Properties: map[string]plan.PropertySchema{
					"cart_id": {Property: plan.Property{Name: "cart_id", Types: []plan.PropertyType{plan.PrimitiveTypeString}}, Required: true},
				},
			}),
		},
	}

	ctx := &TSContext{}
	require.NoError(t, processEventRules(tp, ctx, newTestRegistry()))

	require.Len(t, ctx.AnalyticsMethods, 2)
	require.Len(t, ctx.Interfaces, 2)

	interfaceNames := []string{ctx.Interfaces[0].Name, ctx.Interfaces[1].Name}
	assert.ElementsMatch(t, []string{"GroupTraits", "PageProperties"}, interfaceNames)
}
