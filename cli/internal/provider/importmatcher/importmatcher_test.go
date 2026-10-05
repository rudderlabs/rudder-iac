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

	claimed := importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

	require.NotNil(t, first.MatchedWith)
	assert.Equal(t, "checkout", first.ExternalID)

	// Loser keeps the namer identity it already has.
	assert.Nil(t, second.MatchedWith)
	assert.Equal(t, "checkout-2", second.ExternalID)
	assert.Equal(t, "#category:checkout-2", second.Reference)

	// The collision is reported so the caller can fail fast: matcher predicates
	// are meant to mirror upstream uniqueness, so two remotes claiming one local
	// signals flawed matching or dirty upstream data.
	require.Len(t, claimed, 1)
	assert.Equal(t, importmatcher.MultipleClaimed{
		ResourceType:      "category",
		LocalURN:          "category:checkout",
		ClaimedByRemoteID: "cat_a",
		RemoteID:          "cat_b",
	}, claimed[0])
}

func TestMultipleClaimed_String(t *testing.T) {
	t.Parallel()

	claimed := importmatcher.MultipleClaimed{
		ResourceType:      "category",
		LocalURN:          "category:checkout",
		ClaimedByRemoteID: "cat_a",
		RemoteID:          "cat_b",
	}

	assert.Equal(t,
		`local resource "category:checkout" is matched by multiple remotes "cat_a" and "cat_b"`,
		claimed.String())
}

func TestMark_NoCollisionReturnsNoClaims(t *testing.T) {
	t.Parallel()

	local := localResource("checkout", "category", resources.ResourceData{"name": "Checkout"})
	remote := &resources.RemoteResource{
		ID:         "cat_a",
		ExternalID: "checkout-1",
		Reference:  "#category:checkout-1",
		Data:       "Checkout",
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(local),
		Importable: importableWith("category", remote),
	}

	claimed := importmatcher.Mark(scope, []importmatcher.Matcher{matchByName("category")})

	assert.Empty(t, claimed)
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

// importableOf builds an importable collection spanning several resource types.
func importableOf(byType map[string][]*resources.RemoteResource) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	for resourceType, rs := range byType {
		m := make(map[string]*resources.RemoteResource, len(rs))
		for _, r := range rs {
			m[r.ID] = r
		}
		collection.Set(resourceType, m)
	}
	return collection
}

// endpointRemote is a remote identified by what it points at, the way a
// connection names its destination by remote ID.
type endpointRemote struct {
	name       string
	endpointID string
}

// matchByEndpoint links a remote to the local resource of the same name whose
// "endpoint" data holds the URN the remote's endpoint resolves to — the shape
// of the connection matchers.
func matchByEndpoint(resourceType, endpointType string) importmatcher.Matcher {
	return importmatcher.Matcher{
		ResourceType: resourceType,
		Match: func(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
			remote := r.Data.(endpointRemote)
			endpointURN, ok := importmatcher.ResolveLocalURN(scope, endpointType, remote.endpointID)
			if !ok {
				return nil
			}
			local, _ := importmatcher.ByData(scope.LocalGraph, resourceType, func(data resources.ResourceData) bool {
				return data["name"] == remote.name && data["endpoint"] == endpointURN
			})
			return local
		},
	}
}

func TestMark_DependentListedBeforeItsDependencyStillLinks(t *testing.T) {
	t.Parallel()

	var (
		warehouse      = localResource("warehouse", "destination", resources.ResourceData{"name": "Warehouse"})
		webToWarehouse = localResource("web-to-warehouse", "connection", resources.ResourceData{"name": "Web", "endpoint": "destination:warehouse"})
		destination    = &resources.RemoteResource{ID: "dst_1", ExternalID: "warehouse-1", Reference: "#destination:warehouse-1", Data: "Warehouse"}
		connection     = &resources.RemoteResource{ID: "conn_1", ExternalID: "web-to-warehouse-1", Reference: "#connection:web-to-warehouse-1", Data: endpointRemote{name: "Web", endpointID: "dst_1"}}
	)

	scope := importmatcher.Scope{
		LocalGraph: graphWith(warehouse, webToWarehouse),
		Importable: importableOf(map[string][]*resources.RemoteResource{
			"destination": {destination},
			"connection":  {connection},
		}),
	}

	// Dependent first: aggregating matchers across providers can yield either
	// order.
	importmatcher.Mark(scope, []importmatcher.Matcher{
		matchByEndpoint("connection", "destination"),
		matchByName("destination"),
	})

	assert.Same(t, webToWarehouse, connection.MatchedWith)
	assert.Same(t, warehouse, destination.MatchedWith)
}

func TestMark_MatchesADependencyOnceForAllDependents(t *testing.T) {
	t.Parallel()

	var (
		webToWarehouse = localResource("web-to-warehouse", "connection", resources.ResourceData{"name": "Web", "endpoint": "destination:warehouse"})
		iosToWarehouse = localResource("ios-to-warehouse", "connection", resources.ResourceData{"name": "iOS", "endpoint": "destination:warehouse"})
		destination    = &resources.RemoteResource{ID: "dst_1", ExternalID: "warehouse-1", Reference: "#destination:warehouse-1", Data: "Warehouse"}
		web            = &resources.RemoteResource{ID: "conn_1", ExternalID: "web-1", Reference: "#connection:web-1", Data: endpointRemote{name: "Web", endpointID: "dst_1"}}
		ios            = &resources.RemoteResource{ID: "conn_2", ExternalID: "ios-1", Reference: "#connection:ios-1", Data: endpointRemote{name: "iOS", endpointID: "dst_1"}}

		destinationMatches int
	)

	byName := matchByName("destination")
	countingDestination := importmatcher.Matcher{
		ResourceType: "destination",
		Match: func(scope importmatcher.Scope, r *resources.RemoteResource) *resources.Resource {
			destinationMatches++
			return byName.Match(scope, r)
		},
	}

	scope := importmatcher.Scope{
		LocalGraph: graphWith(
			localResource("warehouse", "destination", resources.ResourceData{"name": "Warehouse"}),
			webToWarehouse,
			iosToWarehouse,
		),
		Importable: importableOf(map[string][]*resources.RemoteResource{
			"destination": {destination},
			"connection":  {web, ios},
		}),
	}

	importmatcher.Mark(scope, []importmatcher.Matcher{
		matchByEndpoint("connection", "destination"),
		countingDestination,
	})

	assert.Equal(t, 1, destinationMatches)
	assert.Same(t, webToWarehouse, web.MatchedWith)
	assert.Same(t, iosToWarehouse, ios.MatchedWith)
}

func TestMark_DependencyCycleLeavesRemotesUnmatched(t *testing.T) {
	t.Parallel()

	// Each remote matches only through the other's match, so neither can
	// resolve first.
	var (
		left  = &resources.RemoteResource{ID: "l_1", ExternalID: "left-1", Reference: "#left:left-1", Data: endpointRemote{name: "L", endpointID: "r_1"}}
		right = &resources.RemoteResource{ID: "r_1", ExternalID: "right-1", Reference: "#right:right-1", Data: endpointRemote{name: "R", endpointID: "l_1"}}
	)

	scope := importmatcher.Scope{
		LocalGraph: graphWith(
			localResource("left", "left", resources.ResourceData{"name": "L", "endpoint": "right:right"}),
			localResource("right", "right", resources.ResourceData{"name": "R", "endpoint": "left:left"}),
		),
		Importable: importableOf(map[string][]*resources.RemoteResource{
			"left":  {left},
			"right": {right},
		}),
	}

	importmatcher.Mark(scope, []importmatcher.Matcher{
		matchByEndpoint("left", "right"),
		matchByEndpoint("right", "left"),
	})

	assert.Nil(t, left.MatchedWith)
	assert.Nil(t, right.MatchedWith)
}

func TestMark_DependencyWithoutMatcherResolvesToItsRecordedMark(t *testing.T) {
	t.Parallel()

	var (
		warehouse   = localResource("warehouse", "destination", resources.ResourceData{"name": "Warehouse"})
		web         = localResource("web", "connection", resources.ResourceData{"name": "Web", "endpoint": "destination:warehouse"})
		destination = &resources.RemoteResource{ID: "dst_1", ExternalID: "warehouse", Reference: "#destination:warehouse", Data: "Warehouse", MatchedWith: warehouse}
		connection  = &resources.RemoteResource{ID: "conn_1", ExternalID: "web-1", Reference: "#connection:web-1", Data: endpointRemote{name: "Web", endpointID: "dst_1"}}
	)

	scope := importmatcher.Scope{
		LocalGraph: graphWith(warehouse, web),
		Importable: importableOf(map[string][]*resources.RemoteResource{
			"destination": {destination},
			"connection":  {connection},
		}),
	}

	// No destination matcher this pass: the destination's earlier mark stands.
	importmatcher.Mark(scope, []importmatcher.Matcher{
		matchByEndpoint("connection", "destination"),
	})

	assert.Same(t, web, connection.MatchedWith)
}

func TestMatchedLocal_WithoutImportableCollection(t *testing.T) {
	t.Parallel()

	local, importable := importmatcher.MatchedLocal(importmatcher.Scope{LocalGraph: graphWith()}, "destination", "dst_1")

	assert.Nil(t, local)
	assert.False(t, importable)
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
