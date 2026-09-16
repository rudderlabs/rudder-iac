package connection

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localConnection(id, sourceURN, destinationURN string) *resources.Resource {
	return resources.NewResource(id, ResourceType, resources.ResourceData{
		SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
		DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
		EnabledKey:     true,
	}, []string{})
}

// matcherScope wires the universe the connection matcher consults: the local
// graph plus an importable collection holding a SQL model source already
// matched to the local source "users" — the rename import --merge performs.
func matcherScope(locals ...*resources.Resource) importmatcher.Scope {
	g := resources.NewGraph()
	// The destination is already CLI-managed: found via its import metadata.
	g.AddResource(resources.NewResource("webhook", destination.DestinationResourceType, resources.ResourceData{}, []string{},
		resources.WithResourceImportMetadata("dst-1", "ws-1")))
	localSource := resources.NewResource("users", sqlmodel.ResourceType, resources.ResourceData{}, []string{})
	g.AddResource(localSource)
	// A local source whose id happens to equal an unmatched importable's
	// assigned name: the import-merge resolution has to win over that
	// coincidence, or the connection would link to the wrong source.
	g.AddResource(resources.NewResource("orders", sqlmodel.ResourceType, resources.ResourceData{}, []string{}))
	for _, local := range locals {
		g.AddResource(local)
	}

	importable := resources.NewRemoteResources()
	importable.Set(sqlmodel.ResourceType, map[string]*resources.RemoteResource{
		"src-1": {ID: "src-1", ExternalID: "users", MatchedWith: localSource},
		"src-2": {ID: "src-2", ExternalID: "orders"}, // importable, unmatched
	})
	return importmatcher.Scope{LocalGraph: g, Importable: importable}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	m := Matcher()
	assert.Equal(t, ResourceType, m.ResourceType)

	t.Run("matches the local connection wired to the same endpoints", func(t *testing.T) {
		t.Parallel()

		scope := matcherScope(localConnection("users-to-webhook", "retl-source-sql-model:users", "destination:webhook"))

		local := m.Match(scope, importableConnection("conn-1", "users-to-webhook", "src-1", "dst-1"))

		require.NotNil(t, local)
		assert.Equal(t, "users-to-webhook", local.ID())
	})

	t.Run("no match when the source is importable but unmatched", func(t *testing.T) {
		t.Parallel()

		scope := matcherScope(localConnection("orders-to-webhook", "retl-source-sql-model:orders", "destination:webhook"))

		assert.Nil(t, m.Match(scope, importableConnection("conn-1", "orders-to-webhook", "src-2", "dst-1")))
	})

	t.Run("no match when the destination has no local counterpart", func(t *testing.T) {
		t.Parallel()

		scope := matcherScope(localConnection("users-to-webhook", "retl-source-sql-model:users", "destination:webhook"))

		assert.Nil(t, m.Match(scope, importableConnection("conn-1", "users-to-webhook", "src-1", "dst-unknown")))
	})

	t.Run("no match when no local connection has the pair", func(t *testing.T) {
		t.Parallel()

		scope := matcherScope(localConnection("users-to-s3", "retl-source-sql-model:users", "destination:s3"))

		assert.Nil(t, m.Match(scope, importableConnection("conn-1", "users-to-webhook", "src-1", "dst-1")))
	})

	t.Run("matches endpoints managed without import metadata", func(t *testing.T) {
		t.Parallel()

		// Endpoints created through a regular apply carry no import metadata;
		// their externalIds, captured at LoadImportable time, are the local
		// resource ids.
		g := resources.NewGraph()
		g.AddResource(resources.NewResource("users", sqlmodel.ResourceType, resources.ResourceData{}, []string{}))
		g.AddResource(resources.NewResource("webhook", destination.DestinationResourceType, resources.ResourceData{}, []string{}))
		g.AddResource(localConnection("users-to-webhook", "retl-source-sql-model:users", "destination:webhook"))
		scope := importmatcher.Scope{LocalGraph: g, Importable: resources.NewRemoteResources()}

		remote := importableConnection("conn-1", "users-to-webhook", "src-9", "dst-9")
		data := remote.Data.(*RemoteConnection)
		data.SourceExternalID = "users"
		data.DestinationExternalID = "webhook"

		local := m.Match(scope, remote)

		require.NotNil(t, local)
		assert.Equal(t, "users-to-webhook", local.ID())
	})
}
