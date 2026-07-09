package importmatcher_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localResource(id, resourceType string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(id, resourceType, data, []string{})
}

func graphWith(rs ...*resources.Resource) *resources.Graph {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return g
}

func importableWith(resourceType string, rs ...*resources.RemoteResource) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	m := make(map[string]*resources.RemoteResource, len(rs))
	for _, r := range rs {
		m[r.ID] = r
	}
	collection.Set(resourceType, m)
	return collection
}

// matchByName links remotes to locals when the remote Data (a plain string in
// these tests) equals the local resource's "name" data field.
func matchByName(resourceType string) importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: resourceType,
		Match: func(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
			name, _ := r.Data.(string)
			local, _ := importmatcher.ByData(scope.LocalGraph, resourceType, func(data resources.ResourceData) bool {
				localName, _ := data["name"].(string)
				return localName == name
			})
			return local
		},
	}
}

func TestMark_MatchAdoptsLocalIdentity(t *testing.T) {
	t.Parallel()

	local := localResource("checkout", "category", resources.ResourceData{"name": "Checkout"})
	remote := &resources.RemoteResource{
		ID:         "cat_remote_1",
		ExternalID: "checkout-1",
		Reference:  "#category:checkout-1",
		Data:       "Checkout",
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(local),
		Importable: importableWith("category", remote),
	}

	importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

	require.NotNil(t, remote.MatchedWith)
	assert.Equal(t, "checkout", remote.MatchedWith.ID())
	assert.Equal(t, "checkout", remote.ExternalID)
	assert.Equal(t, "#category:checkout", remote.Reference)
}

func TestMark_NoMatchKeepsNamerIdentity(t *testing.T) {
	t.Parallel()

	local := localResource("checkout", "category", resources.ResourceData{"name": "Checkout"})
	remote := &resources.RemoteResource{
		ID:         "cat_remote_1",
		ExternalID: "payments",
		Reference:  "#category:payments",
		Data:       "Payments",
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(local),
		Importable: importableWith("category", remote),
	}

	importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

	assert.Nil(t, remote.MatchedWith)
	assert.Equal(t, "payments", remote.ExternalID)
	assert.Equal(t, "#category:payments", remote.Reference)
}

func TestMark_NoMatchersIsNoOp(t *testing.T) {
	t.Parallel()

	remote := &resources.RemoteResource{
		ID:         "src_remote_1",
		ExternalID: "my-source",
		Reference:  "#event-stream-source:my-source",
		Data:       "My Source",
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(),
		Importable: importableWith("event-stream-source", remote),
	}

	importmatcher.Mark(scope, nil)

	assert.Nil(t, remote.MatchedWith)
	assert.Equal(t, "my-source", remote.ExternalID)
}

func TestMark_MatcherForOtherTypeLeavesResourcesUntouched(t *testing.T) {
	t.Parallel()

	remote := &resources.RemoteResource{
		ID:         "src_remote_1",
		ExternalID: "my-source",
		Reference:  "#event-stream-source:my-source",
		Data:       "My Source",
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(),
		Importable: importableWith("event-stream-source", remote),
	}

	importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

	assert.Nil(t, remote.MatchedWith)
	assert.Equal(t, "my-source", remote.ExternalID)
}

func TestMark_ClaimedFirstWinsBySortedRemoteID(t *testing.T) {
	t.Parallel()

	local := localResource("checkout", "category", resources.ResourceData{"name": "Checkout"})

	// Both remotes match the same local; the one with the lower remote ID wins.
	first := &resources.RemoteResource{
		ID:         "cat_a",
		ExternalID: "checkout-1",
		Reference:  "#category:checkout-1",
		Data:       "Checkout",
	}
	second := &resources.RemoteResource{
		ID:         "cat_b",
		ExternalID: "checkout-2",
		Reference:  "#category:checkout-2",
		Data:       "Checkout",
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(local),
		Importable: importableWith("category", first, second),
	}

	importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

	require.NotNil(t, first.MatchedWith)
	assert.Equal(t, "checkout", first.ExternalID)

	// Loser keeps the namer identity it already has.
	assert.Nil(t, second.MatchedWith)
	assert.Equal(t, "checkout-2", second.ExternalID)
	assert.Equal(t, "#category:checkout-2", second.Reference)
}

func TestMark_RewritesReferenceShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		reference string
		want      string
	}{
		{"colon shape", "#category:checkout-1", "#category:checkout"},
		{"slash shape", "#/transformation/transformations/checkout-1", "#/transformation/transformations/checkout"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			local := localResource("checkout", "category", resources.ResourceData{"name": "Checkout"})
			remote := &resources.RemoteResource{
				ID:         "cat_remote_1",
				ExternalID: "checkout-1",
				Reference:  tc.reference,
				Data:       "Checkout",
			}

			scope := importmatcher.Scope{
				LocalGraph: graphWith(local),
				Importable: importableWith("category", remote),
			}

			importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

			assert.Equal(t, tc.want, remote.Reference)
		})
	}
}

func TestByData_ReturnsDeterministicFirstMatch(t *testing.T) {
	t.Parallel()

	// Two locals satisfy the predicate; the one with the lower ID is returned.
	a := localResource("a-checkout", "category", resources.ResourceData{"name": "Checkout"})
	b := localResource("b-checkout", "category", resources.ResourceData{"name": "Checkout"})
	g := graphWith(b, a)

	local, ok := importmatcher.ByData(g, "category", func(data resources.ResourceData) bool {
		name, _ := data["name"].(string)
		return name == "Checkout"
	})

	require.True(t, ok)
	assert.Equal(t, "a-checkout", local.ID())
}

func TestByData_NoMatch(t *testing.T) {
	t.Parallel()

	g := graphWith(localResource("checkout", "category", resources.ResourceData{"name": "Checkout"}))

	local, ok := importmatcher.ByData(g, "category", func(data resources.ResourceData) bool {
		return false
	})

	assert.False(t, ok)
	assert.Nil(t, local)
}

type rawPayload struct {
	ImportName string
}

func TestByRawData_MatchesTypedPayload(t *testing.T) {
	t.Parallel()

	r := resources.NewResource(
		"lodash",
		"transformation-library",
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&rawPayload{ImportName: "lodash"}),
	)
	g := graphWith(r)

	local, ok := importmatcher.ByRawData(g, "transformation-library", func(raw any) bool {
		p, ok := raw.(*rawPayload)
		return ok && p.ImportName == "lodash"
	})

	require.True(t, ok)
	assert.Equal(t, "lodash", local.ID())
}

func TestByRawData_NoMatch(t *testing.T) {
	t.Parallel()

	r := resources.NewResource(
		"lodash",
		"transformation-library",
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&rawPayload{ImportName: "lodash"}),
	)
	g := graphWith(r)

	local, ok := importmatcher.ByRawData(g, "transformation-library", func(raw any) bool {
		p, ok := raw.(*rawPayload)
		return ok && p.ImportName == "underscore"
	})

	assert.False(t, ok)
	assert.Nil(t, local)
}
