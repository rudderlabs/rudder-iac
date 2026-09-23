package typescript

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCollectKeyMaps_OnlyRenamedInterfaces verifies that a map is emitted only
// for interfaces with at least one camelCase→serial rename, and that
// identity-keyed interfaces are skipped.
func TestCollectKeyMaps_OnlyRenamedInterfaces(t *testing.T) {
	ctx := &TSContext{
		Interfaces: []TSInterface{
			{
				Name: "UserSignedUp",
				Properties: []TSInterfaceProperty{
					{Name: "productId", SerialName: "product_id", Type: "string"},
					{Name: "email", SerialName: "email", Type: "string"},
				},
			},
			{
				// No renames → no map.
				Name: "AllIdentity",
				Properties: []TSInterfaceProperty{
					{Name: "email", SerialName: "email", Type: "string"},
				},
			},
		},
	}

	r, err := collectKeyMaps(ctx)
	require.NoError(t, err)

	assert.True(t, r.needsMap["UserSignedUp"])
	assert.False(t, r.needsMap["AllIdentity"])

	assert.Equal(t, []TSKeyMap{
		{
			Name: "UserSignedUpKeyMap",
			Entries: []TSKeyMapEntry{
				{FieldName: "productId", SerialName: "product_id"},
			},
		},
	}, ctx.KeyMaps)
}

// TestCollectKeyMaps_NestedRecursion verifies a parent whose only "rename" is a
// nested object still gets a map that references the child's map, and that the
// child map is ordered before the parent (TS const TDZ safety).
func TestCollectKeyMaps_NestedRecursion(t *testing.T) {
	ctx := &TSContext{
		Interfaces: []TSInterface{
			{
				Name: "Parent",
				Properties: []TSInterfaceProperty{
					// Identity key, but its type references a child that renames.
					{Name: "child", SerialName: "child", Type: "Child"},
				},
			},
		},
		NestedInterfaces: []TSInterface{
			{
				Name: "Child",
				Properties: []TSInterfaceProperty{
					{Name: "firstName", SerialName: "first_name", Type: "string"},
				},
			},
		},
	}

	r, err := collectKeyMaps(ctx)
	require.NoError(t, err)

	assert.True(t, r.needsMap["Parent"])
	assert.True(t, r.needsMap["Child"])

	// Child map must be emitted before Parent map so the nested reference is
	// initialized first.
	assert.Equal(t, []TSKeyMap{
		{
			Name: "ChildKeyMap",
			Entries: []TSKeyMapEntry{
				{FieldName: "firstName", SerialName: "first_name"},
			},
		},
		{
			Name: "ParentKeyMap",
			Entries: []TSKeyMapEntry{
				{FieldName: "child", SerialName: "child", NestedMapName: "ChildKeyMap"},
			},
		},
	}, ctx.KeyMaps)
}

// TestCollectKeyMaps_ArrayOfObjectsThroughAlias covers the shape that shipped
// camelCase keys before DAW-3732: a property typed as an alias for an array of
// objects (`CustomTypeProfileList = Profile[]`). Resolution has to see through
// both the alias and the array, otherwise the array items keep their camelCase
// keys on the wire.
func TestCollectKeyMaps_ArrayOfObjectsThroughAlias(t *testing.T) {
	ctx := &TSContext{
		CustomTypeAliases: []TSTypeAlias{
			{Alias: "ProfileList", Type: "Profile[]"},
		},
		CustomInterfaces: []TSInterface{
			{
				Name: "Profile",
				Properties: []TSInterfaceProperty{
					{Name: "firstName", SerialName: "first_name", Type: "string"},
				},
			},
		},
		Interfaces: []TSInterface{
			{
				Name: "Event",
				Properties: []TSInterfaceProperty{
					{Name: "profileList", SerialName: "profile_list", Type: "ProfileList"},
				},
			},
		},
	}

	_, err := collectKeyMaps(ctx)
	require.NoError(t, err)

	require.Len(t, ctx.KeyMaps, 2)
	assert.Equal(t, TSKeyMap{
		Name: "EventKeyMap",
		Entries: []TSKeyMapEntry{
			{FieldName: "profileList", SerialName: "profile_list", NestedMapName: "ProfileKeyMap"},
		},
	}, ctx.KeyMaps[1])
}

// TestCollectKeyMaps_VariantsMergeIntoAliasMap verifies a discriminated union
// produces one merged map named after the alias, and no per-case maps (which
// would be dead code the validator tsconfig rejects).
func TestCollectKeyMaps_VariantsMergeIntoAliasMap(t *testing.T) {
	ctx := &TSContext{
		VariantTypes: []TSVariantGroup{
			{
				UnionAlias: TSTypeAlias{Alias: "PageContext", Type: "PageContextCaseHome | PageContextCaseProduct"},
				CaseInterfaces: []TSInterface{
					{
						Name: "PageContextCaseHome",
						Properties: []TSInterfaceProperty{
							{Name: "pageType", SerialName: "page_type", Type: `"home"`},
						},
					},
					{
						Name: "PageContextCaseProduct",
						Properties: []TSInterfaceProperty{
							{Name: "pageType", SerialName: "page_type", Type: `"product"`},
							{Name: "productId", SerialName: "product_id", Type: "string"},
						},
					},
				},
			},
		},
	}

	_, err := collectKeyMaps(ctx)
	require.NoError(t, err)

	assert.Equal(t, []TSKeyMap{
		{
			Name: "PageContextKeyMap",
			Entries: []TSKeyMapEntry{
				{FieldName: "pageType", SerialName: "page_type"},
				{FieldName: "productId", SerialName: "product_id"},
			},
		},
	}, ctx.KeyMaps, "branch maps merge into one alias map; no per-case maps are emitted")
}

// TestCollectKeyMaps_InlineUnionOfMappedInterfacesErrors guards the one shape
// merging cannot cover: a union of remappable interfaces with no alias to hang
// the merged map on. Failing to generate beats silently shipping camelCase keys.
func TestCollectKeyMaps_InlineUnionOfMappedInterfacesErrors(t *testing.T) {
	ctx := &TSContext{
		CustomInterfaces: []TSInterface{
			{
				Name:       "A",
				Properties: []TSInterfaceProperty{{Name: "firstName", SerialName: "first_name", Type: "string"}},
			},
			{
				Name:       "B",
				Properties: []TSInterfaceProperty{{Name: "lastName", SerialName: "last_name", Type: "string"}},
			},
		},
		Interfaces: []TSInterface{
			{
				Name:       "Event",
				Properties: []TSInterfaceProperty{{Name: "either", SerialName: "either", Type: "A | B"}},
			},
		},
	}

	_, err := collectKeyMaps(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inline union of remappable interfaces")
}

// TestWireKeyMaps_RewritesPropsArg verifies the props SDK argument is routed
// through applyKeyMap only when the type needs a map, for both the simple
// (track) and dispatcher (page/identify/group) method shapes.
func TestWireKeyMaps_RewritesPropsArg(t *testing.T) {
	ctx := &TSContext{
		Interfaces: []TSInterface{
			{
				Name:       "UserSignedUp",
				Properties: []TSInterfaceProperty{{Name: "productId", SerialName: "product_id", Type: "string"}},
			},
			{
				Name:       "Plain",
				Properties: []TSInterfaceProperty{{Name: "email", SerialName: "email", Type: "string"}},
			},
		},
		AnalyticsMethods: []TSAnalyticsMethod{
			{
				Name:          "trackUserSignedUp",
				SDKMethodName: "track",
				PropsTypeName: "UserSignedUp",
				SDKArguments: []TSSDKArgument{
					{Value: `"User Signed Up"`},
					propsArg("%s as unknown as SDKApiObject", "props"),
				},
			},
			{
				Name:          "trackPlain",
				SDKMethodName: "track",
				PropsTypeName: "Plain",
				SDKArguments: []TSSDKArgument{
					{Value: `"Plain"`},
					propsArg("%s as unknown as SDKApiObject", "props"),
				},
			},
			{
				Name:          "page",
				SDKMethodName: "page",
				PropsTypeName: "UserSignedUp",
				DispatcherBranches: []TSDispatcherBranch{
					{
						Condition: `typeof arg0 === "string"`,
						SDKArguments: []TSSDKArgument{
							{Value: "arg0"},
							propsArg("%s as unknown as SDKApiObject | undefined", "arg1"),
						},
					},
				},
			},
		},
	}

	r, err := collectKeyMaps(ctx)
	require.NoError(t, err)
	require.NoError(t, wireKeyMaps(ctx, r))

	assert.True(t, ctx.UsesApplyKeyMap)
	assert.Equal(t, "UserSignedUpKeyMap", ctx.AnalyticsMethods[0].PropsKeyMapName)
	assert.Equal(t,
		"applyKeyMap(props, UserSignedUpKeyMap) as unknown as SDKApiObject",
		ctx.AnalyticsMethods[0].SDKArguments[1].Value)

	// Type without a map: props forwarded verbatim.
	assert.Empty(t, ctx.AnalyticsMethods[1].PropsKeyMapName)
	assert.Equal(t, "props as unknown as SDKApiObject", ctx.AnalyticsMethods[1].SDKArguments[1].Value)

	// Dispatcher branches are rewritten the same way, and the non-props argument
	// in the same branch is untouched.
	pageArgs := ctx.AnalyticsMethods[2].DispatcherBranches[0].SDKArguments
	assert.Equal(t, "arg0", pageArgs[0].Value)
	assert.Equal(t,
		"applyKeyMap(arg1, UserSignedUpKeyMap) as unknown as SDKApiObject | undefined",
		pageArgs[1].Value)
}

// TestResolveTypeExpressions covers the decoration stripping and alias
// following that mapNameFor depends on.
func TestResolveTypeExpressions(t *testing.T) {
	r := &keyMapResolver{
		ifaces:   map[string]*TSInterface{"Profile": {Name: "Profile"}},
		aliases:  map[string]string{"ProfileList": "Profile[]", "Email": "string"},
		variants: map[string][]string{"PageContext": {"PageContextCaseHome"}},
		caseOf:   map[string]bool{"PageContextCaseHome": true},
		needsMap: map[string]bool{"Profile": true, "PageContextCaseHome": true},
	}

	for _, tc := range []struct {
		tsType string
		want   string
	}{
		{"Profile", "ProfileKeyMap"},
		{"Profile[]", "ProfileKeyMap"},
		{"Array<Profile>", "ProfileKeyMap"},
		{"Profile | null", "ProfileKeyMap"},
		{"ProfileList", "ProfileKeyMap"},
		{"PageContext", "PageContextKeyMap"},
		{"Email", ""},
		{"string | number", ""},
		{"Record<string, unknown>", ""},
		{"Unknown", ""},
	} {
		got, err := r.mapNameFor(tc.tsType)
		require.NoError(t, err, tc.tsType)
		assert.Equal(t, tc.want, got, tc.tsType)
	}
}
