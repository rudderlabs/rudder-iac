package connection

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNamer kebab-cases the human name, mirroring the real namer's strategy.
type mockNamer struct{}

func (m *mockNamer) Name(input namer.ScopeName) (string, error) {
	return strings.ToLower(strings.ReplaceAll(input.Name, " ", "-")), nil
}

func (m *mockNamer) Load(names []namer.ScopeName) error {
	return nil
}

// mockResolver resolves entityType/remoteID pairs from a canned map and errors
// on anything else, like the real ImportRefResolver does for resources that
// are neither importable nor CLI-managed.
type mockResolver struct {
	refs map[string]string
}

func (s *mockResolver) ResolveToReference(entityType string, remoteID string) (string, error) {
	if ref, ok := s.refs[entityType+"/"+remoteID]; ok {
		return ref, nil
	}
	return "", errors.New("resource not present in resources collection")
}

func TestLoadImportable(t *testing.T) {
	t.Run("names connections after their endpoints", func(t *testing.T) {
		conn := client.Connection{ID: "conn-remote-1", SourceID: "src-remote-1", DestinationID: "dst-remote-1", IsEnabled: true}
		mock := &MockConnectionClient{
			ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
				return &client.ConnectionsPage{Connections: []client.Connection{
					conn,
					{ID: "conn-remote-retl", SourceID: "retl-src-1", DestinationID: "dst-remote-1"},
				}}, nil
			},
			GetDestinationsFunc: func() ([]client.Destination, error) {
				return []client.Destination{{ID: "dst-remote-1", Name: "S3 Bucket", ExternalID: "s3"}}, nil
			},
		}
		mock.SetGetSourcesFunc(func(_ context.Context) ([]sourceClient.EventStreamSource, error) {
			return []sourceClient.EventStreamSource{
				{ID: "src-remote-1", Name: "Android App", WorkspaceID: "workspace-123", ExternalID: "android"},
			}, nil
		})
		h := NewHandler(mock, "event-stream")

		collection, err := h.LoadImportable(t.Context(), &mockNamer{})

		require.NoError(t, err)
		require.Len(t, mock.ListCalls, 1)
		require.NotNil(t, mock.ListCalls[0].HasExternalID)
		assert.False(t, *mock.ListCalls[0].HasExternalID, "must list only connections without an externalId")

		remotes := collection.GetAll(EventStreamConnectionResourceType)
		require.Len(t, remotes, 1)
		assert.Equal(t, &resources.RemoteResource{
			ID:         "conn-remote-1",
			ExternalID: "android-app-to-s3-bucket",
			Reference:  "#event-stream-connections:android-app-to-s3-bucket",
			Data: &RemoteConnection{
				Connection:            conn,
				WorkspaceID:           "workspace-123",
				SourceExternalID:      "android",
				DestinationExternalID: "s3",
			},
		}, remotes["conn-remote-1"])
	})

	t.Run("destination name falls back to its remote id", func(t *testing.T) {
		mock := &MockConnectionClient{
			ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
				return &client.ConnectionsPage{Connections: []client.Connection{
					{ID: "conn-remote-1", SourceID: "src-remote-1", DestinationID: "dst-remote-1"},
				}}, nil
			},
		}
		mock.SetGetSourcesFunc(func(_ context.Context) ([]sourceClient.EventStreamSource, error) {
			return []sourceClient.EventStreamSource{{ID: "src-remote-1", Name: "Android App"}}, nil
		})
		h := NewHandler(mock, "event-stream")

		collection, err := h.LoadImportable(t.Context(), &mockNamer{})

		require.NoError(t, err)
		remotes := collection.GetAll(EventStreamConnectionResourceType)
		require.Len(t, remotes, 1)
		assert.Equal(t, "android-app-to-dst-remote-1", remotes["conn-remote-1"].ExternalID)
	})

	t.Run("no unmanaged connections skips the destination lookup", func(t *testing.T) {
		mock := &MockConnectionClient{}
		h := NewHandler(mock, "event-stream")

		collection, err := h.LoadImportable(t.Context(), &mockNamer{})

		require.NoError(t, err)
		assert.Empty(t, collection.GetAll(EventStreamConnectionResourceType))
		assert.NotContains(t, mock.Calls, "GetDestinations")
	})

	t.Run("surfaces list errors", func(t *testing.T) {
		mock := &MockConnectionClient{
			ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
				return nil, errors.New("boom")
			},
		}
		h := NewHandler(mock, "event-stream")

		_, err := h.LoadImportable(t.Context(), &mockNamer{})

		require.Error(t, err)
		assert.ErrorContains(t, err, "listing event stream connections")
	})
}

// importableConnection builds one importable remote connection the way
// LoadImportable stores it.
func importableConnection(remoteID, externalID, sourceID, destinationID string, enabled bool) *resources.RemoteResource {
	return &resources.RemoteResource{
		ID:         remoteID,
		ExternalID: externalID,
		Reference:  fmt.Sprintf("#%s:%s", EventStreamConnectionResourceKind, externalID),
		Data: &RemoteConnection{
			Connection:  client.Connection{ID: remoteID, SourceID: sourceID, DestinationID: destinationID, IsEnabled: enabled},
			WorkspaceID: "workspace-123",
		},
	}
}

func connectionCollection(remotes ...*resources.RemoteResource) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	resourceMap := make(map[string]*resources.RemoteResource, len(remotes))
	for _, r := range remotes {
		resourceMap[r.ID] = r
	}
	collection.Set(EventStreamConnectionResourceType, resourceMap)
	return collection
}

func TestFormatForExport(t *testing.T) {
	refs := map[string]string{
		source.ResourceType + "/src-1":                 "#event-stream-source:android",
		source.ResourceType + "/src-2":                 "#event-stream-source:web",
		destination.DestinationResourceType + "/dst-1": "#destination:s3",
	}

	t.Run("writes one spec per run sorted by id", func(t *testing.T) {
		h := NewHandler(nil, "event-stream")
		collection := connectionCollection(
			importableConnection("conn-2", "web-to-s3", "src-2", "dst-1", false),
			importableConnection("conn-1", "android-to-s3", "src-1", "dst-1", true),
		)

		entities, entries, err := h.FormatForExport(collection, &mockNamer{}, &mockResolver{refs: refs})

		require.NoError(t, err)
		require.Len(t, entities, 1)
		assert.Equal(t, "event-stream/connections.yaml", entities[0].RelativePath)

		spec, ok := entities[0].Content.(*specs.Spec)
		require.True(t, ok)
		assert.Equal(t, specs.SpecVersionV1, spec.Version)
		assert.Equal(t, EventStreamConnectionResourceKind, spec.Kind)
		assert.Equal(t, map[string]any{
			"connections": []map[string]any{
				{
					"id":          "android-to-s3",
					"source":      "#event-stream-source:android",
					"destination": "#destination:s3",
					"enabled":     true,
				},
				{
					"id":          "web-to-s3",
					"source":      "#event-stream-source:web",
					"destination": "#destination:s3",
					"enabled":     false,
				},
			},
		}, spec.Spec)
		assert.Equal(t, map[string]any{
			"name": MetadataName,
			"import": map[string]any{
				"workspaces": []any{
					map[string]any{
						"workspace_id": "workspace-123",
						"resources": []any{
							map[string]any{"urn": "event-stream-connection:android-to-s3", "remote_id": "conn-1"},
							map[string]any{"urn": "event-stream-connection:web-to-s3", "remote_id": "conn-2"},
						},
					},
				},
			},
		}, spec.Metadata)

		assert.Equal(t, []importmanifest.ImportEntry{
			{WorkspaceID: "workspace-123", URN: "event-stream-connection:android-to-s3", RemoteID: "conn-1"},
			{WorkspaceID: "workspace-123", URN: "event-stream-connection:web-to-s3", RemoteID: "conn-2"},
		}, entries)
	})

	t.Run("matched connections write manifest entries only", func(t *testing.T) {
		h := NewHandler(nil, "event-stream")
		matched := importableConnection("conn-2", "existing-conn", "src-2", "dst-1", true)
		matched.MatchedWith = resources.NewResource("existing-conn", EventStreamConnectionResourceType, resources.ResourceData{}, []string{})
		collection := connectionCollection(
			importableConnection("conn-1", "android-to-s3", "src-1", "dst-1", true),
			matched,
		)

		entities, entries, err := h.FormatForExport(collection, &mockNamer{}, &mockResolver{refs: refs})

		require.NoError(t, err)
		require.Len(t, entities, 1)
		spec := entities[0].Content.(*specs.Spec)
		connections := spec.Spec["connections"].([]map[string]any)
		require.Len(t, connections, 1)
		assert.Equal(t, "android-to-s3", connections[0]["id"])

		assert.ElementsMatch(t, []importmanifest.ImportEntry{
			{WorkspaceID: "workspace-123", URN: "event-stream-connection:android-to-s3", RemoteID: "conn-1"},
			{WorkspaceID: "workspace-123", URN: "event-stream-connection:existing-conn", RemoteID: "conn-2"},
		}, entries)
	})

	t.Run("only matched connections write no spec file", func(t *testing.T) {
		h := NewHandler(nil, "event-stream")
		matched := importableConnection("conn-1", "existing-conn", "src-1", "dst-1", true)
		matched.MatchedWith = resources.NewResource("existing-conn", EventStreamConnectionResourceType, resources.ResourceData{}, []string{})

		entities, entries, err := h.FormatForExport(connectionCollection(matched), &mockNamer{}, &mockResolver{refs: refs})

		require.NoError(t, err)
		assert.Nil(t, entities)
		require.Len(t, entries, 1)
	})

	t.Run("skips connections whose endpoint cannot be resolved", func(t *testing.T) {
		// dst-9 resolves to neither an importable nor a CLI-managed destination
		// (e.g. its type is not onboarded); the connection cannot be expressed
		// as spec refs and is left out entirely — no spec entry, no manifest
		// entry.
		h := NewHandler(nil, "event-stream")
		collection := connectionCollection(
			importableConnection("conn-1", "android-to-s3", "src-1", "dst-1", true),
			importableConnection("conn-2", "web-to-unmanaged", "src-2", "dst-9", true),
		)

		entities, entries, err := h.FormatForExport(collection, &mockNamer{}, &mockResolver{refs: refs})

		require.NoError(t, err)
		require.Len(t, entities, 1)
		spec := entities[0].Content.(*specs.Spec)
		connections := spec.Spec["connections"].([]map[string]any)
		require.Len(t, connections, 1)
		assert.Equal(t, "android-to-s3", connections[0]["id"])
		assert.Equal(t, []importmanifest.ImportEntry{
			{WorkspaceID: "workspace-123", URN: "event-stream-connection:android-to-s3", RemoteID: "conn-1"},
		}, entries)
	})

	t.Run("builds refs from managed endpoint ids when the resolver cannot", func(t *testing.T) {
		// A managed destination's graph entry carries no file metadata, so the
		// resolver cannot serve it; the ref is built from the endpoint's
		// externalId instead.
		h := NewHandler(nil, "event-stream")
		managed := importableConnection("conn-1", "android-to-s3", "src-9", "dst-9", true)
		remote := managed.Data.(*RemoteConnection)
		remote.SourceExternalID = "android"
		remote.DestinationExternalID = "s3"

		entities, entries, err := h.FormatForExport(connectionCollection(managed), &mockNamer{}, &mockResolver{refs: refs})

		require.NoError(t, err)
		require.Len(t, entities, 1)
		spec := entities[0].Content.(*specs.Spec)
		connections := spec.Spec["connections"].([]map[string]any)
		require.Len(t, connections, 1)
		assert.Equal(t, "#event-stream-source:android", connections[0]["source"])
		assert.Equal(t, "#destination:s3", connections[0]["destination"])
		require.Len(t, entries, 1)
	})

	t.Run("empty collection writes nothing", func(t *testing.T) {
		h := NewHandler(nil, "event-stream")

		entities, entries, err := h.FormatForExport(resources.NewRemoteResources(), &mockNamer{}, &mockResolver{})

		require.NoError(t, err)
		assert.Nil(t, entities)
		assert.Nil(t, entries)
	})

	t.Run("errors on foreign data in the collection", func(t *testing.T) {
		h := NewHandler(nil, "event-stream")
		collection := resources.NewRemoteResources()
		collection.Set(EventStreamConnectionResourceType, map[string]*resources.RemoteResource{
			"x": {ID: "x", ExternalID: "x", Data: "not a connection"},
		})

		_, _, err := h.FormatForExport(collection, &mockNamer{}, &mockResolver{refs: refs})

		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to cast remote resource to event stream connection")
	})

	t.Run("errors on connections from multiple workspaces", func(t *testing.T) {
		h := NewHandler(nil, "event-stream")
		other := importableConnection("conn-2", "web-to-s3", "src-2", "dst-1", true)
		other.Data.(*RemoteConnection).WorkspaceID = "workspace-456"
		collection := connectionCollection(
			importableConnection("conn-1", "android-to-s3", "src-1", "dst-1", true),
			other,
		)

		_, _, err := h.FormatForExport(collection, &mockNamer{}, &mockResolver{refs: refs})

		require.Error(t, err)
		assert.ErrorContains(t, err, "cannot export resources from multiple workspaces into a single spec file")
	})
}

func TestImport(t *testing.T) {
	data := resources.ResourceData{
		SourceKey:      "src-1",
		DestinationKey: "dst-1",
		EnabledKey:     true,
	}

	t.Run("marks the remote managed when nothing changed", func(t *testing.T) {
		mock := &MockConnectionClient{
			GetFunc: func(id string) (*client.Connection, error) {
				return &client.Connection{ID: id, SourceID: "src-1", DestinationID: "dst-1", IsEnabled: true}, nil
			},
		}
		eventStreamSources(mock, "src-1")
		h := NewHandler(mock, "event-stream")

		result, err := h.Import(t.Context(), "android-to-s3", data, "conn-remote-1")

		require.NoError(t, err)
		assert.Equal(t, []string{"GetConnection", "SetConnectionExternalID"}, mock.Calls)
		assert.Equal(t, []SetExternalIDCall{{ID: "conn-remote-1", ExternalID: "android-to-s3"}}, mock.SetExternalIDCalls)
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-1",
		}, result)
	})

	t.Run("reconciles enabled then marks managed", func(t *testing.T) {
		mock := &MockConnectionClient{
			GetFunc: func(id string) (*client.Connection, error) {
				return &client.Connection{ID: id, SourceID: "src-1", DestinationID: "dst-1", IsEnabled: false}, nil
			},
		}
		eventStreamSources(mock, "src-1")
		h := NewHandler(mock, "event-stream")

		_, err := h.Import(t.Context(), "android-to-s3", data, "conn-remote-1")

		require.NoError(t, err)
		assert.Equal(t, []string{"GetConnection", "UpdateConnection", "SetConnectionExternalID"}, mock.Calls)
		require.Len(t, mock.UpdateCalls, 1)
		assert.True(t, mock.UpdateCalls[0].IsEnabled)
	})

	t.Run("endpoint change replaces the row and skips the stamp", func(t *testing.T) {
		// Update's replacement path creates a new row whose create body already
		// carries the externalId; the deleted row has nothing to stamp.
		mock := &MockConnectionClient{
			GetFunc: func(id string) (*client.Connection, error) {
				return &client.Connection{ID: id, SourceID: "src-old", DestinationID: "dst-1", IsEnabled: true}, nil
			},
		}
		eventStreamSources(mock, "src-old")
		h := NewHandler(mock, "event-stream")

		result, err := h.Import(t.Context(), "android-to-s3", data, "conn-remote-1")

		require.NoError(t, err)
		assert.Equal(t, []string{"GetConnection", "DeleteConnection", "CreateConnection"}, mock.Calls)
		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, "android-to-s3", mock.CreateCalls[0].ExternalID)
		assert.Equal(t, "remote-connection-id", (*result)[IDKey])
	})

	t.Run("refuses to adopt a non event stream connection", func(t *testing.T) {
		// A stale or hand-edited import manifest can point at a rETL row; the
		// guard must reject it before Update mutates it.
		mock := &MockConnectionClient{
			GetFunc: func(id string) (*client.Connection, error) {
				return &client.Connection{ID: id, SourceID: "retl-src-1", DestinationID: "dst-1", IsEnabled: true}, nil
			},
		}
		eventStreamSources(mock, "src-1")
		h := NewHandler(mock, "event-stream")

		_, err := h.Import(t.Context(), "android-to-s3", data, "conn-remote-1")

		require.Error(t, err)
		assert.ErrorContains(t, err, `connection "conn-remote-1": source "retl-src-1" is not an event stream source`)
		assert.Equal(t, []string{"GetConnection"}, mock.Calls)
	})

	t.Run("surfaces get errors", func(t *testing.T) {
		mock := &MockConnectionClient{
			GetFunc: func(_ string) (*client.Connection, error) {
				return nil, errors.New("boom")
			},
		}
		h := NewHandler(mock, "event-stream")

		_, err := h.Import(t.Context(), "android-to-s3", data, "conn-remote-1")

		require.Error(t, err)
		assert.ErrorContains(t, err, "getting event stream connection during import")
	})
}

func TestList(t *testing.T) {
	mock := &MockConnectionClient{
		ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
			return &client.ConnectionsPage{Connections: []client.Connection{
				{ID: "conn-remote-1", ExternalID: "android-to-s3", SourceID: "src-remote-1", DestinationID: "dst-remote-1", IsEnabled: true},
				{ID: "conn-remote-2", SourceID: "src-remote-1", DestinationID: "dst-remote-2", IsEnabled: false},
				{ID: "conn-remote-retl", SourceID: "retl-src-1", DestinationID: "dst-remote-1"},
			}}, nil
		},
	}
	eventStreamSources(mock, "src-remote-1")
	h := NewHandler(mock, "event-stream")

	result, err := h.List(t.Context(), nil)

	require.NoError(t, err)
	require.Len(t, mock.ListCalls, 1)
	assert.Nil(t, mock.ListCalls[0].HasExternalID, "list must report managed and unmanaged rows alike")
	assert.Equal(t, []resources.ResourceData{
		{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-1",
			EnabledKey:       true,
			ExternalIDKey:    "android-to-s3",
		},
		{
			IDKey:            "conn-remote-2",
			SourceIDKey:      "src-remote-1",
			DestinationIDKey: "dst-remote-2",
			EnabledKey:       false,
		},
	}, result)
}
