package connection

import (
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadResourcesFromRemote(t *testing.T) {
	t.Run("collects externalId-carrying connections across pages", func(t *testing.T) {
		mock := &MockConnectionClient{
			ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
				return &client.ConnectionsPage{
					APIPage: client.APIPage{Paging: client.Paging{Next: "/v2/connections?page=2"}},
					Connections: []client.Connection{
						{ID: "conn-remote-1", ExternalID: "android-to-s3", SourceID: "src-remote-1", DestinationID: "dst-remote-1", IsEnabled: true},
					},
				}, nil
			},
			NextFunc: func(paging client.Paging) (*client.ConnectionsPage, error) {
				if paging.Next == "" {
					return nil, nil
				}
				return &client.ConnectionsPage{
					Connections: []client.Connection{
						{ID: "conn-remote-2", ExternalID: "web-to-s3", SourceID: "src-remote-2", DestinationID: "dst-remote-1", IsEnabled: false},
					},
				}, nil
			},
		}
		h := NewHandler(mock)

		collection, err := h.LoadResourcesFromRemote(t.Context())

		require.NoError(t, err)
		require.Len(t, mock.ListCalls, 1)
		require.NotNil(t, mock.ListCalls[0].HasExternalID)
		assert.True(t, *mock.ListCalls[0].HasExternalID, "must list only connections carrying an externalId")

		remotes := collection.GetAll(EventStreamConnectionResourceType)
		require.Len(t, remotes, 2)
		first := remotes["conn-remote-1"]
		require.NotNil(t, first)
		assert.Equal(t, "android-to-s3", first.ExternalID)
		assert.Equal(t, client.Connection{
			ID: "conn-remote-1", ExternalID: "android-to-s3",
			SourceID: "src-remote-1", DestinationID: "dst-remote-1", IsEnabled: true,
		}, first.Data)
		second := remotes["conn-remote-2"]
		require.NotNil(t, second)
		assert.Equal(t, "web-to-s3", second.ExternalID)
	})

	t.Run("surfaces list errors", func(t *testing.T) {
		mock := &MockConnectionClient{
			ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
				return nil, errors.New("boom")
			},
		}
		h := NewHandler(mock)

		_, err := h.LoadResourcesFromRemote(t.Context())

		require.Error(t, err)
		assert.ErrorContains(t, err, "listing event stream connections")
	})
}

// remoteCollection builds the merged cross-provider collection MapRemoteToState
// receives: event stream sources and destinations keyed by remote id (as their
// handlers load them), plus the given connection rows.
func remoteCollection(t *testing.T, conns ...client.Connection) *resources.RemoteResources {
	t.Helper()
	collection := resources.NewRemoteResources()
	collection.Set(source.ResourceType, map[string]*resources.RemoteResource{
		"src-remote-1": {ID: "src-remote-1", ExternalID: "my-android-source"},
	})
	collection.Set(destination.DestinationResourceType, map[string]*resources.RemoteResource{
		"dst-remote-1": {ID: "dst-remote-1", ExternalID: "s3"},
		"dst-remote-2": {ID: "dst-remote-2", ExternalID: ""}, // remote-only destination, not CLI-managed
	})
	connMap := make(map[string]*resources.RemoteResource)
	for _, conn := range conns {
		connMap[conn.ID] = &resources.RemoteResource{ID: conn.ID, ExternalID: conn.ExternalID, Data: conn}
	}
	collection.Set(EventStreamConnectionResourceType, connMap)
	return collection
}

func TestMapRemoteToState(t *testing.T) {
	t.Run("keys state on externalId with endpoint refs mirroring the spec side", func(t *testing.T) {
		h := NewHandler(nil)
		collection := remoteCollection(t, client.Connection{
			ID: "conn-remote-1", ExternalID: "android-to-s3",
			SourceID: "src-remote-1", DestinationID: "dst-remote-1", IsEnabled: true,
		})

		s, err := h.MapRemoteToState(collection)

		require.NoError(t, err)
		require.Len(t, s.Resources, 1)
		rs := s.Resources["event-stream-connection:android-to-s3"]
		require.NotNil(t, rs)
		assert.Equal(t, "android-to-s3", rs.ID)
		assert.Equal(t, EventStreamConnectionResourceType, rs.Type)
		assert.Equal(t, map[string]any{
			SourceKey:      &resources.PropertyRef{URN: "event-stream-source:my-android-source", Property: "id"},
			DestinationKey: &resources.PropertyRef{URN: "destination:s3", Property: "id"},
			EnabledKey:     true,
		}, rs.Input)
		assert.Equal(t, map[string]any{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-1",
		}, rs.Output)
	})

	t.Run("skips rows whose source is not an event stream source", func(t *testing.T) {
		// rETL connections share the generic list; their sources never appear
		// in the event stream source collection.
		h := NewHandler(nil)
		collection := remoteCollection(t, client.Connection{
			ID: "conn-remote-retl", ExternalID: "retl-to-s3",
			SourceID: "retl-src-1", DestinationID: "dst-remote-1", IsEnabled: true,
		})

		s, err := h.MapRemoteToState(collection)

		require.NoError(t, err)
		assert.Empty(t, s.Resources)
	})

	t.Run("skips rows whose destination is not CLI-managed", func(t *testing.T) {
		h := NewHandler(nil)
		collection := remoteCollection(t, client.Connection{
			ID: "conn-remote-1", ExternalID: "android-to-unmanaged",
			SourceID: "src-remote-1", DestinationID: "dst-remote-2", IsEnabled: true,
		})

		s, err := h.MapRemoteToState(collection)

		require.NoError(t, err)
		assert.Empty(t, s.Resources)
	})

	t.Run("errors on foreign data in the collection", func(t *testing.T) {
		h := NewHandler(nil)
		collection := resources.NewRemoteResources()
		collection.Set(EventStreamConnectionResourceType, map[string]*resources.RemoteResource{
			"x": {ID: "x", ExternalID: "x", Data: "not a connection"},
		})

		_, err := h.MapRemoteToState(collection)

		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to cast resource to event stream connection")
	})
}
