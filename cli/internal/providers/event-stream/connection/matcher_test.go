package connection

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/importmatcher"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func localConnection(id, sourceURN, destinationURN string) *resources.Resource {
	return resources.NewResource(id, EventStreamConnectionResourceType, resources.ResourceData{
		SourceKey:      &resources.PropertyRef{URN: sourceURN, Property: "id"},
		DestinationKey: &resources.PropertyRef{URN: destinationURN, Property: "id"},
		EnabledKey:     true,
	}, []string{})
}

func remoteConnection(remoteID, sourceID, destinationID string) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID: remoteID,
		Data: &RemoteConnection{
			Connection: client.Connection{ID: remoteID, SourceID: sourceID, DestinationID: destinationID},
		},
	}
}

// matcherScope wires the universe the connection matcher consults: the local
// graph plus an importable collection holding a source remote already matched
// to the local source "android".
func matcherScope(locals ...*resources.Resource) importmatcher.Scope {
	g := resources.NewGraph()
	// The destination is already CLI-managed: found via its import metadata.
	g.AddResource(resources.NewResource("s3", destination.DestinationResourceType, resources.ResourceData{}, []string{},
		resources.WithResourceImportMetadata("dst-1", "workspace-123")))
	localSource := resources.NewResource("android", source.ResourceType, resources.ResourceData{}, []string{})
	g.AddResource(localSource)
	for _, r := range locals {
		g.AddResource(r)
	}

	importable := resources.NewRemoteResources()
	importable.Set(source.ResourceType, map[string]*resources.RemoteResource{
		"src-1": {ID: "src-1", ExternalID: "android", MatchedWith: localSource},
		"src-2": {ID: "src-2", ExternalID: "web-source"}, // importable, unmatched
	})
	return importmatcher.Scope{LocalGraph: g, Importable: importable}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	m := Matcher()
	assert.Equal(t, EventStreamConnectionResourceType, m.ResourceType)

	t.Run("matches the local connection wired to the same endpoints", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope(localConnection("android-to-s3", "event-stream-source:android", "destination:s3"))

		local := m.Match(scope, remoteConnection("conn-1", "src-1", "dst-1"))

		require.NotNil(t, local)
		assert.Equal(t, "android-to-s3", local.ID())
	})

	t.Run("no match when the source is importable but unmatched", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope(localConnection("web-to-s3", "event-stream-source:web-source", "destination:s3"))

		assert.Nil(t, m.Match(scope, remoteConnection("conn-1", "src-2", "dst-1")))
	})

	t.Run("no match when the destination has no local counterpart", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope(localConnection("android-to-s3", "event-stream-source:android", "destination:s3"))

		assert.Nil(t, m.Match(scope, remoteConnection("conn-1", "src-1", "dst-unknown")))
	})

	t.Run("no match when no local connection has the pair", func(t *testing.T) {
		t.Parallel()
		scope := matcherScope(localConnection("android-to-gcs", "event-stream-source:android", "destination:gcs"))

		assert.Nil(t, m.Match(scope, remoteConnection("conn-1", "src-1", "dst-1")))
	})
}
