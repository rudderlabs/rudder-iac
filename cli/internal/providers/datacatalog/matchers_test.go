package datacatalog

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localData(id, resourceType string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(id, resourceType, data, []string{})
}

func scopeWith(rs ...*resources.Resource) importmatcher.Scope {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return importmatcher.Scope{LocalGraph: g}
}

func matcherFor(t *testing.T, resourceType string) importmatcher.Matcher {
	t.Helper()

	for _, m := range New(nil).ResourceMatchers() {
		if m.ResourceType == resourceType {
			return m
		}
	}
	t.Fatalf("no matcher registered for %s", resourceType)
	return importmatcher.Matcher{}
}

func TestResourceMatchers_RegistersAllFiveTypes(t *testing.T) {
	t.Parallel()

	matchers := New(nil).ResourceMatchers()

	registered := make([]string, 0, len(matchers))
	for _, m := range matchers {
		registered = append(registered, m.ResourceType)
	}
	assert.ElementsMatch(t, []string{
		types.CategoryResourceType,
		types.CustomTypeResourceType,
		types.PropertyResourceType,
		types.EventResourceType,
		types.TrackingPlanResourceType,
	}, registered)
}

func TestCategoryMatcher(t *testing.T) {
	t.Parallel()

	matcher := matcherFor(t, types.CategoryResourceType)
	remote := func(name string) *resources.RemoteResource {
		return &resources.RemoteResource{ID: "cat_1", Data: &catalog.Category{ID: "cat_1", Name: name}}
	}

	t.Run("matches on name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("checkout", types.CategoryResourceType, resources.ResourceData{"name": "Checkout"}))

		local := matcher.Match(scope, remote("Checkout"))

		require.NotNil(t, local)
		assert.Equal(t, "checkout", local.ID())
	})

	t.Run("no match for different name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("checkout", types.CategoryResourceType, resources.ResourceData{"name": "Checkout"}))

		assert.Nil(t, matcher.Match(scope, remote("Payments")))
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("broken", types.CategoryResourceType, resources.ResourceData{"name": ""}))

		assert.Nil(t, matcher.Match(scope, remote("")))
	})
}

func TestCustomTypeMatcher(t *testing.T) {
	t.Parallel()

	matcher := matcherFor(t, types.CustomTypeResourceType)
	remote := func(name string) *resources.RemoteResource {
		return &resources.RemoteResource{ID: "ct_1", Data: &catalog.CustomType{ID: "ct_1", Name: name}}
	}

	t.Run("matches on name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("email-type", types.CustomTypeResourceType, resources.ResourceData{"name": "EmailType"}))

		local := matcher.Match(scope, remote("EmailType"))

		require.NotNil(t, local)
		assert.Equal(t, "email-type", local.ID())
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("broken", types.CustomTypeResourceType, resources.ResourceData{"name": ""}))

		assert.Nil(t, matcher.Match(scope, remote("")))
	})
}

func TestTrackingPlanMatcher(t *testing.T) {
	t.Parallel()

	matcher := matcherFor(t, types.TrackingPlanResourceType)
	remote := func(name string) *resources.RemoteResource {
		return &resources.RemoteResource{ID: "tp_1", Data: &catalog.TrackingPlanWithIdentifiers{
			TrackingPlan: catalog.TrackingPlan{ID: "tp_1", Name: name},
		}}
	}

	t.Run("matches on name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("mobile-plan", types.TrackingPlanResourceType, resources.ResourceData{"name": "Mobile Plan"}))

		local := matcher.Match(scope, remote("Mobile Plan"))

		require.NotNil(t, local)
		assert.Equal(t, "mobile-plan", local.ID())
	})

	t.Run("no match for different name", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localData("mobile-plan", types.TrackingPlanResourceType, resources.ResourceData{"name": "Mobile Plan"}))

		assert.Nil(t, matcher.Match(scope, remote("Web Plan")))
	})
}

func TestEventMatcher(t *testing.T) {
	t.Parallel()

	matcher := matcherFor(t, types.EventResourceType)
	remote := func(name, eventType string) *resources.RemoteResource {
		return &resources.RemoteResource{ID: "ev_1", Data: &catalog.Event{ID: "ev_1", Name: name, EventType: eventType}}
	}
	localEvent := func(id, name, eventType string) *resources.Resource {
		return localData(id, types.EventResourceType, resources.ResourceData{"name": name, "eventType": eventType})
	}

	t.Run("matches on name and event type", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localEvent("page-viewed", "Page Viewed", "track"))

		local := matcher.Match(scope, remote("Page Viewed", "track"))

		require.NotNil(t, local)
		assert.Equal(t, "page-viewed", local.ID())
	})

	t.Run("no match when event type differs", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localEvent("page-viewed", "Page Viewed", "track"))

		assert.Nil(t, matcher.Match(scope, remote("Page Viewed", "page")))
	})

	t.Run("non-track events with empty names match on event type", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localEvent("identify-event", "", "identify"))

		local := matcher.Match(scope, remote("", "identify"))

		require.NotNil(t, local)
		assert.Equal(t, "identify-event", local.ID())
	})

	t.Run("empty event type never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localEvent("broken", "", ""))

		assert.Nil(t, matcher.Match(scope, remote("", "")))
	})
}

func TestPropertyMatcher(t *testing.T) {
	t.Parallel()

	matcher := matcherFor(t, types.PropertyResourceType)
	remoteProperty := func(p catalog.Property) *resources.RemoteResource {
		p.ID = "prop_1"
		return &resources.RemoteResource{ID: "prop_1", Data: &p}
	}
	localProperty := func(id, name string, propType any, config map[string]interface{}) *resources.Resource {
		return localData(id, types.PropertyResourceType, resources.ResourceData{
			"name":   name,
			"type":   propType,
			"config": config,
		})
	}

	t.Run("matches on name and type without item types", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localProperty("email", "email", "string", map[string]interface{}{}))

		local := matcher.Match(scope, remoteProperty(catalog.Property{Name: "email", Type: "string"}))

		require.NotNil(t, local)
		assert.Equal(t, "email", local.ID())
	})

	t.Run("matches sorted item types across key spellings", func(t *testing.T) {
		t.Parallel()
		// Local: snake_case key, pre-sorted at load. Remote: camelCase key, unsorted.
		scope := scopeWith(localProperty("tags", "tags", "array", map[string]interface{}{
			"item_types": []interface{}{"number", "string"},
		}))

		local := matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "tags", Type: "array",
			Config: map[string]interface{}{"itemTypes": []interface{}{"string", "number"}},
		}))

		require.NotNil(t, local)
		assert.Equal(t, "tags", local.ID())
	})

	t.Run("no match when item types differ", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localProperty("tags", "tags", "array", map[string]interface{}{
			"item_types": []interface{}{"string"},
		}))

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "tags", Type: "array",
			Config: map[string]interface{}{"itemTypes": []interface{}{"number"}},
		})))
	})

	t.Run("matches sorted comma-joined multi types", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localProperty("flexible", "flexible", "integer,string", map[string]interface{}{}))

		local := matcher.Match(scope, remoteProperty(catalog.Property{Name: "flexible", Type: "integer,string"}))

		require.NotNil(t, local)
	})

	t.Run("local custom-type against remote string type does not match", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localProperty("typed", "typed", resources.PropertyRef{URN: "custom-type:email-type"}, map[string]interface{}{}))

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{Name: "typed", Type: "string"})))
	})

	t.Run("remote custom-type against local string type does not match", func(t *testing.T) {
		t.Parallel()
		// Remote references a custom type (ct_remote_1) but the local property is
		// a plain string — even were ct_remote_1 resolvable, the types differ.
		scope := scopeWith(localProperty("typed", "typed", "string", map[string]interface{}{}))

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "typed", Type: "string", DefinitionId: "ct_remote_1",
		})))
	})

	t.Run("local custom-type item against remote string item does not match", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localProperty("tags", "tags", "array", map[string]interface{}{
			"item_types": []interface{}{resources.PropertyRef{URN: "custom-type:email-type"}},
		}))

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "tags", Type: "array",
			Config: map[string]interface{}{"itemTypes": []interface{}{"string"}},
		})))
	})

	t.Run("empty name never matches", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(localProperty("broken", "", "string", map[string]interface{}{}))

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{Name: "", Type: "string"})))
	})

	// managedCustomType is a local custom-type resource carrying the remote ID
	// it was imported from, so resolveTypeRef can map a remote DefinitionId to
	// its local URN.
	managedCustomType := func(id, remoteID string) *resources.Resource {
		return resources.NewResource(id, types.CustomTypeResourceType,
			resources.ResourceData{"name": id}, []string{},
			resources.WithResourceImportMetadata(remoteID, "ws-1"))
	}

	t.Run("custom-type property links via an already-managed custom type", func(t *testing.T) {
		t.Parallel()
		// Local property's type is a PropertyRef to custom-type:email-type; the
		// custom type is already managed, imported from remote id ct_remote_1.
		scope := scopeWith(
			localProperty("email", "User Email", resources.PropertyRef{URN: "custom-type:email-type"}, map[string]interface{}{}),
			managedCustomType("email-type", "ct_remote_1"),
		)

		local := matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "User Email", Type: "email-type", DefinitionId: "ct_remote_1",
		}))

		require.NotNil(t, local)
		assert.Equal(t, "email", local.ID())
	})

	t.Run("different custom types do not link", func(t *testing.T) {
		t.Parallel()
		// Local references email-type; remote references a different custom type
		// (email-string, ct_remote_2). Same name, different type ⇒ no match.
		scope := scopeWith(
			localProperty("email", "User Email", resources.PropertyRef{URN: "custom-type:email-type"}, map[string]interface{}{}),
			managedCustomType("email-type", "ct_remote_1"),
			managedCustomType("email-string", "ct_remote_2"),
		)

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "User Email", Type: "email-string", DefinitionId: "ct_remote_2",
		})))
	})

	t.Run("custom-type property links via a custom type matched in this import", func(t *testing.T) {
		t.Parallel()
		// The custom type is not yet managed but is being imported now and has
		// matched a local custom type; resolveTypeRef uses its MatchedWith URN.
		localCustomType := resources.NewResource("email-type", types.CustomTypeResourceType,
			resources.ResourceData{"name": "email-type"}, []string{})
		importable := resources.NewRemoteResources()
		importable.Set(types.CustomTypeResourceType, map[string]*resources.RemoteResource{
			"ct_remote_1": {ID: "ct_remote_1", MatchedWith: localCustomType},
		})
		scope := scopeWith(localProperty("email", "User Email", resources.PropertyRef{URN: "custom-type:email-type"}, map[string]interface{}{}))
		scope.Importable = importable

		local := matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "User Email", Type: "email-type", DefinitionId: "ct_remote_1",
		}))

		require.NotNil(t, local)
		assert.Equal(t, "email", local.ID())
	})

	t.Run("unresolvable custom type does not link", func(t *testing.T) {
		t.Parallel()
		// Remote references a custom type that is neither managed nor imported.
		scope := scopeWith(localProperty("email", "User Email", resources.PropertyRef{URN: "custom-type:email-type"}, map[string]interface{}{}))

		assert.Nil(t, matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "User Email", Type: "email-type", DefinitionId: "ct_unknown",
		})))
	})

	t.Run("array of custom type links via an already-managed custom type", func(t *testing.T) {
		t.Parallel()
		scope := scopeWith(
			localProperty("tags", "tags", "array", map[string]interface{}{
				"item_types": []interface{}{resources.PropertyRef{URN: "custom-type:tag-type"}},
			}),
			managedCustomType("tag-type", "ct_remote_9"),
		)

		local := matcher.Match(scope, remoteProperty(catalog.Property{
			Name: "tags", Type: "array", ItemDefinitionId: "ct_remote_9",
		}))

		require.NotNil(t, local)
		assert.Equal(t, "tags", local.ID())
	})
}
