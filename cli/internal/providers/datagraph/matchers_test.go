package datagraph_test

import (
	"testing"

	dgClient "github.com/rudderlabs/rudder-iac/api/client/datagraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph"
	dghandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/datagraph"
	modelhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	relhandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localDataGraph(id, accountID string) *resources.Resource {
	return resources.NewResource(id, dghandler.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&dgModel.DataGraphResource{ID: id, AccountID: accountID}))
}

func localModel(id, displayName, parentLocalID string) *resources.Resource {
	parentURN := resources.URN(parentLocalID, dghandler.HandlerMetadata.ResourceType)
	return resources.NewResource(id, modelhandler.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&dgModel.ModelResource{
			ID:           id,
			DisplayName:  displayName,
			DataGraphRef: &resources.PropertyRef{URN: parentURN},
		}))
}

func localRelationship(id, displayName, parentLocalID string) *resources.Resource {
	parentURN := resources.URN(parentLocalID, dghandler.HandlerMetadata.ResourceType)
	return resources.NewResource(id, relhandler.HandlerMetadata.ResourceType, resources.ResourceData{}, []string{},
		resources.WithRawData(&dgModel.RelationshipResource{
			ID:           id,
			DisplayName:  displayName,
			DataGraphRef: &resources.PropertyRef{URN: parentURN},
		}))
}

func remoteDataGraph(remoteID, accountID string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &dgModel.RemoteDataGraph{DataGraph: &dgClient.DataGraph{ID: remoteID, AccountID: accountID}},
	}
}

func remoteModel(remoteID, name, dataGraphID string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &dgModel.RemoteModel{Model: &dgClient.Model{ID: remoteID, Name: name, DataGraphID: dataGraphID}},
	}
}

func remoteRelationship(remoteID, name, dataGraphID string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:   remoteID,
		Data: &dgModel.RemoteRelationship{Relationship: &dgClient.Relationship{ID: remoteID, Name: name, DataGraphID: dataGraphID}},
	}
}

func matcherScope(locals []*resources.Resource, remotesByType map[string][]*resources.RemoteResource) importmatcher.Scope {
	g := resources.NewGraph()
	for _, r := range locals {
		g.AddResource(r)
	}

	importable := resources.NewRemoteResources()
	for resourceType, remotes := range remotesByType {
		m := make(map[string]*resources.RemoteResource, len(remotes))
		for _, r := range remotes {
			m[r.ID] = r
		}
		importable.Set(resourceType, m)
	}

	return importmatcher.Scope{LocalGraph: g, Importable: importable}
}

func matcherFor(t *testing.T, resourceType string) importmatcher.Matcher {
	t.Helper()

	for _, m := range datagraph.NewProvider(nil, nil).ResourceMatchers() {
		if m.ResourceType == resourceType {
			return m
		}
	}
	t.Fatalf("no matcher registered for %s", resourceType)
	return importmatcher.Matcher{}
}

func TestResourceMatchers_ParentRegisteredBeforeChildren(t *testing.T) {
	t.Parallel()

	matchers := datagraph.NewProvider(nil, nil).ResourceMatchers()

	require.Len(t, matchers, 3)
	assert.Equal(t, dghandler.HandlerMetadata.ResourceType, matchers[0].ResourceType)
	assert.Equal(t, modelhandler.HandlerMetadata.ResourceType, matchers[1].ResourceType)
	assert.Equal(t, relhandler.HandlerMetadata.ResourceType, matchers[2].ResourceType)
}

func TestDataGraphMatcher(t *testing.T) {
	t.Parallel()

	matcher := matcherFor(t, dghandler.HandlerMetadata.ResourceType)

	t.Run("matches on account id", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope([]*resources.Resource{localDataGraph("main-graph", "acc_1")}, nil)

		local := matcher.Match(scope, remoteDataGraph("dg_1", "acc_1"))

		require.NotNil(t, local)
		assert.Equal(t, "main-graph", local.ID())
	})

	t.Run("no match for different account", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope([]*resources.Resource{localDataGraph("main-graph", "acc_1")}, nil)

		assert.Nil(t, matcher.Match(scope, remoteDataGraph("dg_1", "acc_2")))
	})

	t.Run("empty account id never matches", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope([]*resources.Resource{localDataGraph("broken", "")}, nil)

		assert.Nil(t, matcher.Match(scope, remoteDataGraph("dg_1", "")))
	})
}

func TestModelMatcher(t *testing.T) {
	t.Parallel()

	dgMatcher := matcherFor(t, dghandler.HandlerMetadata.ResourceType)
	matcher := matcherFor(t, modelhandler.HandlerMetadata.ResourceType)

	// markParent runs the data-graph matcher the way Mark would, recording the
	// parent match the child matcher depends on.
	markParent := func(t *testing.T, scope importmatcher.Scope, parent *resources.RemoteResource) {
		t.Helper()
		local := dgMatcher.Match(scope, parent)
		require.NotNil(t, local)
		parent.MatchedWith = local
		parent.ExternalID = local.ID()
	}

	t.Run("matches by display name within matched parent", func(t *testing.T) {
		t.Parallel()

		parent := remoteDataGraph("dg_1", "acc_1")
		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localModel("users", "Users", "main-graph")},
			map[string][]*resources.RemoteResource{dghandler.HandlerMetadata.ResourceType: {parent}},
		)
		markParent(t, scope, parent)

		local := matcher.Match(scope, remoteModel("mdl_1", "Users", "dg_1"))

		require.NotNil(t, local)
		assert.Equal(t, "users", local.ID())
	})

	t.Run("no match when parent is unmatched", func(t *testing.T) {
		t.Parallel()

		parent := remoteDataGraph("dg_1", "acc_other")
		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localModel("users", "Users", "main-graph")},
			map[string][]*resources.RemoteResource{dghandler.HandlerMetadata.ResourceType: {parent}},
		)

		assert.Nil(t, matcher.Match(scope, remoteModel("mdl_1", "Users", "dg_1")))
	})

	t.Run("no match for model under a different local parent", func(t *testing.T) {
		t.Parallel()

		parent := remoteDataGraph("dg_1", "acc_1")
		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localModel("users", "Users", "other-graph")},
			map[string][]*resources.RemoteResource{dghandler.HandlerMetadata.ResourceType: {parent}},
		)
		markParent(t, scope, parent)

		assert.Nil(t, matcher.Match(scope, remoteModel("mdl_1", "Users", "dg_1")))
	})

	t.Run("no match when parent is not importable", func(t *testing.T) {
		t.Parallel()

		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localModel("users", "Users", "main-graph")},
			nil,
		)

		assert.Nil(t, matcher.Match(scope, remoteModel("mdl_1", "Users", "dg_1")))
	})

	t.Run("empty display name never matches", func(t *testing.T) {
		t.Parallel()

		parent := remoteDataGraph("dg_1", "acc_1")
		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localModel("broken", "", "main-graph")},
			map[string][]*resources.RemoteResource{dghandler.HandlerMetadata.ResourceType: {parent}},
		)
		markParent(t, scope, parent)

		assert.Nil(t, matcher.Match(scope, remoteModel("mdl_1", "", "dg_1")))
	})
}

func TestRelationshipMatcher(t *testing.T) {
	t.Parallel()

	dgMatcher := matcherFor(t, dghandler.HandlerMetadata.ResourceType)
	matcher := matcherFor(t, relhandler.HandlerMetadata.ResourceType)

	t.Run("matches by display name within matched parent", func(t *testing.T) {
		t.Parallel()

		parent := remoteDataGraph("dg_1", "acc_1")
		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localRelationship("user-orders", "User Orders", "main-graph")},
			map[string][]*resources.RemoteResource{dghandler.HandlerMetadata.ResourceType: {parent}},
		)
		local := dgMatcher.Match(scope, parent)
		require.NotNil(t, local)
		parent.MatchedWith = local
		parent.ExternalID = local.ID()

		matched := matcher.Match(scope, remoteRelationship("rel_1", "User Orders", "dg_1"))

		require.NotNil(t, matched)
		assert.Equal(t, "user-orders", matched.ID())
	})

	t.Run("no match when parent is unmatched", func(t *testing.T) {
		t.Parallel()

		parent := remoteDataGraph("dg_1", "acc_other")
		scope := matcherScope(
			[]*resources.Resource{localDataGraph("main-graph", "acc_1"), localRelationship("user-orders", "User Orders", "main-graph")},
			map[string][]*resources.RemoteResource{dghandler.HandlerMetadata.ResourceType: {parent}},
		)

		assert.Nil(t, matcher.Match(scope, remoteRelationship("rel_1", "User Orders", "dg_1")))
	})
}
