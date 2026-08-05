package typescript

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	needsMap := collectKeyMaps(ctx)

	assert.True(t, needsMap["UserSignedUp"])
	assert.False(t, needsMap["AllIdentity"])

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

	needsMap := collectKeyMaps(ctx)

	assert.True(t, needsMap["Parent"])
	assert.True(t, needsMap["Child"])

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

// TestReferencedInterfaceName_StripsDecorations checks array/optional handling
// and that unions are not resolved (documented nested-object limitation).
func TestReferencedInterfaceName_StripsDecorations(t *testing.T) {
	needsMap := map[string]bool{"Profile": true}

	assert.Equal(t, "Profile", referencedInterfaceName("Profile", needsMap))
	assert.Equal(t, "Profile", referencedInterfaceName("Profile[]", needsMap))
	assert.Equal(t, "Profile", referencedInterfaceName("Array<Profile>", needsMap))
	// Unions/generics are not remapped.
	assert.Equal(t, "", referencedInterfaceName("Profile | null", needsMap))
	assert.Equal(t, "", referencedInterfaceName("Unknown", needsMap))
}

// TestWireTrackKeyMaps_RewritesPropsArg verifies the track props SDK argument
// is routed through applyKeyMap only when the interface needs a map.
func TestWireTrackKeyMaps_RewritesPropsArg(t *testing.T) {
	ctx := &TSContext{
		AnalyticsMethods: []TSAnalyticsMethod{
			{
				Name:            "trackUserSignedUp",
				SDKMethodName:   "track",
				MethodArguments: []TSMethodArgument{{Name: "props", Type: "UserSignedUp"}},
				SDKArguments: []TSSDKArgument{
					{Value: `"User Signed Up"`},
					{Value: "props as unknown as SDKApiObject"},
				},
			},
			{
				Name:            "trackPlain",
				SDKMethodName:   "track",
				MethodArguments: []TSMethodArgument{{Name: "props", Type: "Plain"}},
				SDKArguments: []TSSDKArgument{
					{Value: `"Plain"`},
					{Value: "props as unknown as SDKApiObject"},
				},
			},
		},
	}

	wireTrackKeyMaps(ctx, map[string]bool{"UserSignedUp": true})

	assert.True(t, ctx.UsesApplyKeyMap)
	assert.Equal(t, "UserSignedUpKeyMap", ctx.AnalyticsMethods[0].PropsKeyMapName)
	assert.Equal(t, "applyKeyMap(props, UserSignedUpKeyMap) as unknown as SDKApiObject", ctx.AnalyticsMethods[0].SDKArguments[1].Value)

	// Interface without a map: props forwarded verbatim.
	assert.Empty(t, ctx.AnalyticsMethods[1].PropsKeyMapName)
	assert.Equal(t, "props as unknown as SDKApiObject", ctx.AnalyticsMethods[1].SDKArguments[1].Value)
}
