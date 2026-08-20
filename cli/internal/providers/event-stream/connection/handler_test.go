package connection

import (
	"context"
	"errors"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	sourceClient "github.com/rudderlabs/rudder-iac/api/client/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// connectionsSpec wraps a spec body in the specs.Spec shape handlers receive.
func connectionsSpec(body map[string]any) *specs.Spec {
	return &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    EventStreamConnectionResourceKind,
		Spec:    body,
	}
}

// ticketExampleSpec is the spec body from the ticket's YAML example.
func ticketExampleSpec() map[string]any {
	return map[string]any{
		"connections": []any{
			map[string]any{
				"id":          "android-to-s3",
				"source":      "#event-stream-source:my-android-source",
				"destination": "#destination:s3",
				"enabled":     true,
			},
		},
	}
}

func loadTicketExample(t *testing.T) *connectionResource {
	t.Helper()
	h := NewHandler(nil)
	require.NoError(t, h.LoadSpec("", connectionsSpec(ticketExampleSpec())))
	require.Len(t, h.resources, 1)
	return h.resources["android-to-s3"]
}

func TestParseSpec(t *testing.T) {
	t.Run("one URN per connection entry", func(t *testing.T) {
		h := NewHandler(nil)
		parsed, err := h.ParseSpec("", connectionsSpec(map[string]any{
			"connections": []any{
				map[string]any{"id": "one"},
				map[string]any{"id": "two"},
			},
		}))
		require.NoError(t, err)
		assert.Equal(t, &specs.ParsedSpec{URNs: []specs.URNEntry{
			{URN: "event-stream-connection:one", JSONPointerPath: "/spec/connections/0/id"},
			{URN: "event-stream-connection:two", JSONPointerPath: "/spec/connections/1/id"},
		}}, parsed)
	})

	errorTests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{"missing connections", map[string]any{}, "connections not found in event stream connections spec"},
		{"entry not a map", map[string]any{"connections": []any{"nope"}}, "connection at index 0 is not a map"},
		{"entry without id", map[string]any{"connections": []any{map[string]any{"source": "#event-stream-source:s"}}}, "id not found in connection at index 0"},
	}
	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(nil)
			parsed, err := h.ParseSpec("", connectionsSpec(tt.body))
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, parsed)
		})
	}
}

func TestLoadSpec(t *testing.T) {
	resource := loadTicketExample(t)
	assert.Equal(t, "android-to-s3", resource.LocalID)
	assert.True(t, resource.Enabled)
	assert.Equal(t,
		&resources.PropertyRef{URN: "event-stream-source:my-android-source", Property: "id"},
		resource.Source,
	)
	assert.Equal(t, "destination:s3", resource.Destination.URN)
	assert.Equal(t, "id", resource.Destination.Property)
	require.NotNil(t, resource.Destination.Resolve)
}

func TestLoadSpecEnabledDefaultsToTrue(t *testing.T) {
	h := NewHandler(nil)
	require.NoError(t, h.LoadSpec("", connectionsSpec(map[string]any{
		"connections": []any{
			map[string]any{
				"id":          "no-enabled",
				"source":      "#event-stream-source:src",
				"destination": "#destination:dst",
			},
			map[string]any{
				"id":          "disabled",
				"source":      "#event-stream-source:src",
				"destination": "#destination:dst",
				"enabled":     false,
			},
		},
	})))
	assert.True(t, h.resources["no-enabled"].Enabled)
	assert.False(t, h.resources["disabled"].Enabled)
}

func TestLoadSpecEmptyListIsValid(t *testing.T) {
	// An explicit empty list stays valid so removing the last connection does
	// not invalidate the spec.
	h := NewHandler(nil)
	require.NoError(t, h.LoadSpec("", connectionsSpec(map[string]any{"connections": []any{}})))
	assert.Empty(t, h.resources)
}

func TestLoadSpecErrors(t *testing.T) {
	entry := func(overrides map[string]any) map[string]any {
		e := map[string]any{
			"id":          "one",
			"source":      "#event-stream-source:src",
			"destination": "#destination:dst",
		}
		for k, v := range overrides {
			if v == nil {
				delete(e, k)
				continue
			}
			e[k] = v
		}
		return e
	}

	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name:    "config key fails at parse time",
			body:    map[string]any{"connections": []any{entry(map[string]any{"config": map[string]any{"eventFiltering": true}})}},
			wantErr: "invalid keys: config",
		},
		{
			name:    "unknown entry field",
			body:    map[string]any{"connections": []any{entry(map[string]any{"enbaled": true})}},
			wantErr: "invalid keys: enbaled",
		},
		{
			name:    "unknown top-level field",
			body:    map[string]any{"connections": []any{}, "links": []any{}},
			wantErr: "invalid keys: links",
		},
		{
			name:    "missing source",
			body:    map[string]any{"connections": []any{entry(map[string]any{"source": nil})}},
			wantErr: `connection "one": parsing source reference: invalid reference ""`,
		},
		{
			name:    "missing destination",
			body:    map[string]any{"connections": []any{entry(map[string]any{"destination": nil})}},
			wantErr: `connection "one": parsing destination reference: invalid reference ""`,
		},
		{
			name:    "source ref with wrong kind",
			body:    map[string]any{"connections": []any{entry(map[string]any{"source": "#retl-source-sql-model:my-model"})}},
			wantErr: `connection "one": parsing source reference: invalid reference "#retl-source-sql-model:my-model": expected format #event-stream-source:<id>`,
		},
		{
			name:    "destination ref with wrong kind",
			body:    map[string]any{"connections": []any{entry(map[string]any{"destination": "#event-stream-source:src"})}},
			wantErr: `connection "one": parsing destination reference: invalid reference "#event-stream-source:src": expected format #destination:<id>`,
		},
		{
			name:    "ref without hash prefix",
			body:    map[string]any{"connections": []any{entry(map[string]any{"source": "event-stream-source:src"})}},
			wantErr: "expected format #event-stream-source:<id>",
		},
		{
			name:    "ref with empty id",
			body:    map[string]any{"connections": []any{entry(map[string]any{"source": "#event-stream-source:"})}},
			wantErr: "expected format #event-stream-source:<id>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(nil)
			err := h.LoadSpec("", connectionsSpec(tt.body))
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, h.resources)
		})
	}
}

func TestDestinationRefResolve(t *testing.T) {
	ref, err := parseDestinationRef("#destination:s3")
	require.NoError(t, err)

	t.Run("resolves the remote id from the destination state", func(t *testing.T) {
		value, err := ref.Resolve(&destination.DestinationState{ID: "dst-remote-id"})
		require.NoError(t, err)
		assert.Equal(t, "dst-remote-id", value)
	})

	t.Run("empty remote id errors", func(t *testing.T) {
		_, err := ref.Resolve(&destination.DestinationState{})
		assert.ErrorContains(t, err, "destination state has empty ID")
	})

	t.Run("unexpected state type errors", func(t *testing.T) {
		_, err := ref.Resolve("not-a-destination-state")
		assert.ErrorContains(t, err, "invalid resource data type")
	})
}

// TestGetResourcesGraphDependencies proves the resource graph derives
// dependency edges from the endpoint PropertyRefs: the connection depends on
// both endpoints (created after them) and shows up as their dependent
// (deleted before them).
func TestGetResourcesGraphDependencies(t *testing.T) {
	h := NewHandler(nil)
	require.NoError(t, h.LoadSpec("", connectionsSpec(ticketExampleSpec())))
	connectionResources, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, connectionResources, 1)

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("my-android-source", source.ResourceType, resources.ResourceData{}, []string{}))
	graph.AddResource(resources.NewResource("s3", destination.DestinationResourceType, resources.ResourceData{}, []string{}))
	graph.AddResource(connectionResources[0])

	connURN := "event-stream-connection:android-to-s3"
	assert.Equal(t, connURN, connectionResources[0].URN())
	assert.ElementsMatch(t,
		[]string{"event-stream-source:my-android-source", "destination:s3"},
		graph.GetDependencies(connURN),
	)
	assert.Equal(t, []string{connURN}, graph.GetDependents("event-stream-source:my-android-source"))
	assert.Equal(t, []string{connURN}, graph.GetDependents("destination:s3"))

	cycle, err := graph.DetectCycles()
	require.NoError(t, err)
	assert.Nil(t, cycle)
}

// TestConnectionRefResolution proves both endpoint references resolve to the
// referenced resource's remote id from state through the same map-based
// dereference the syncer applies before lifecycle calls: the source via the
// legacy output map, the destination via its typed state's Resolve func.
func TestConnectionRefResolution(t *testing.T) {
	h := NewHandler(nil)
	require.NoError(t, h.LoadSpec("", connectionsSpec(ticketExampleSpec())))
	connectionResources, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, connectionResources, 1)

	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:     "my-android-source",
		Type:   source.ResourceType,
		Output: map[string]any{"id": "src-remote-id"},
	})
	st.AddResource(&state.ResourceState{
		ID:        "s3",
		Type:      destination.DestinationResourceType,
		OutputRaw: &destination.DestinationState{ID: "dst-remote-id"},
	})

	dereferenced, err := state.Dereference(connectionResources[0].Data(), st)
	require.NoError(t, err)
	assert.Equal(t, resources.ResourceData{
		SourceKey:      "src-remote-id",
		DestinationKey: "dst-remote-id",
		EnabledKey:     true,
	}, dereferenced)
}

func TestMigrateSpecIsIdentity(t *testing.T) {
	h := NewHandler(nil)
	spec := connectionsSpec(ticketExampleSpec())
	migrated, err := h.MigrateSpec(spec)
	require.NoError(t, err)
	assert.Same(t, spec, migrated)
}

func TestLoadImportMetadataIsNoOp(t *testing.T) {
	h := NewHandler(nil)
	assert.NoError(t, h.LoadImportMetadata(nil))
	assert.NoError(t, h.LoadImportMetadata(&specs.WorkspacesImportMetadata{}))
}

// eventStreamSources canns GetSources with the sources the rETL filter in
// LoadResourcesFromRemote checks connections against.
func eventStreamSources(mock *MockConnectionClient, ids ...string) {
	sources := make([]sourceClient.EventStreamSource, 0, len(ids))
	for _, id := range ids {
		sources = append(sources, sourceClient.EventStreamSource{ID: id})
	}
	mock.SetGetSourcesFunc(func(_ context.Context) ([]sourceClient.EventStreamSource, error) {
		return sources, nil
	})
}

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
		eventStreamSources(mock, "src-remote-1", "src-remote-2")
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

	t.Run("skips rETL connections", func(t *testing.T) {
		// rETL rows appear in the same generic list; only connections whose
		// source is an event stream source are kept.
		mock := &MockConnectionClient{
			ListFunc: func(_ client.ListConnectionsOptions) (*client.ConnectionsPage, error) {
				return &client.ConnectionsPage{
					Connections: []client.Connection{
						{ID: "conn-remote-1", ExternalID: "android-to-s3", SourceID: "src-remote-1", DestinationID: "dst-remote-1"},
						{ID: "conn-remote-retl", ExternalID: "retl-to-s3", SourceID: "retl-src-1", DestinationID: "dst-remote-1"},
					},
				}, nil
			},
		}
		eventStreamSources(mock, "src-remote-1")
		h := NewHandler(mock)

		collection, err := h.LoadResourcesFromRemote(t.Context())

		require.NoError(t, err)
		remotes := collection.GetAll(EventStreamConnectionResourceType)
		require.Len(t, remotes, 1)
		assert.NotNil(t, remotes["conn-remote-1"])
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
	t.Run("keys state on externalId with spec-shaped endpoint refs", func(t *testing.T) {
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

	t.Run("skips rows whose source is not CLI-managed", func(t *testing.T) {
		h := NewHandler(nil)
		collection := remoteCollection(t, client.Connection{
			ID: "conn-remote-1", ExternalID: "unmanaged-to-s3",
			SourceID: "src-remote-9", DestinationID: "dst-remote-1", IsEnabled: true,
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

	t.Run("keeps rows whose endpoints are both CLI-managed alongside skipped ones", func(t *testing.T) {
		h := NewHandler(nil)
		collection := remoteCollection(t,
			client.Connection{
				ID: "conn-remote-1", ExternalID: "android-to-s3",
				SourceID: "src-remote-1", DestinationID: "dst-remote-1", IsEnabled: true,
			},
			client.Connection{
				ID: "conn-remote-2", ExternalID: "unmanaged-to-s3",
				SourceID: "src-remote-9", DestinationID: "dst-remote-1", IsEnabled: true,
			},
		)

		s, err := h.MapRemoteToState(collection)

		require.NoError(t, err)
		require.Len(t, s.Resources, 1)
		assert.NotNil(t, s.Resources["event-stream-connection:android-to-s3"])
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

// TestEndpointURN pins the two "not CLI-managed" sentinels the endpoint lookup
// has to tolerate. Missing-from-collection is the common one: the event stream
// source and destination handlers both drop rows without an externalId while
// loading, so an unmanaged endpoint never reaches the collection at all.
func TestEndpointURN(t *testing.T) {
	collection := resources.NewRemoteResources()
	collection.Set(source.ResourceType, map[string]*resources.RemoteResource{
		"src-remote-1": {ID: "src-remote-1", ExternalID: "my-android-source"},
		"src-remote-2": {ID: "src-remote-2", ExternalID: ""},
	})

	t.Run("resolves a CLI-managed endpoint to its URN", func(t *testing.T) {
		urn, managed, err := endpointURN(collection, source.ResourceType, "src-remote-1")

		require.NoError(t, err)
		assert.True(t, managed)
		assert.Equal(t, "event-stream-source:my-android-source", urn)
	})

	t.Run("reports unmanaged when the endpoint is absent from the collection", func(t *testing.T) {
		_, managed, err := endpointURN(collection, source.ResourceType, "src-remote-9")

		require.NoError(t, err)
		assert.False(t, managed)
	})

	t.Run("reports unmanaged when the endpoint carries no externalId", func(t *testing.T) {
		_, managed, err := endpointURN(collection, source.ResourceType, "src-remote-2")

		require.NoError(t, err)
		assert.False(t, managed)
	})
}

func TestListAndImportNotImplemented(t *testing.T) {
	h := NewHandler(nil)

	_, err := h.List(t.Context(), nil)
	assert.ErrorIs(t, err, ErrNotImplemented)
	_, err = h.Import(t.Context(), "id", resources.ResourceData{}, "remote-id")
	assert.ErrorIs(t, err, ErrNotImplemented)
}

func TestImportLoadsAreEmptyUntilImplemented(t *testing.T) {
	h := NewHandler(nil)

	importable, err := h.LoadImportable(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, importable.GetAll(EventStreamConnectionResourceType))

	entities, entries, err := h.FormatForExport(resources.NewRemoteResources(), nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entities)
	assert.Nil(t, entries)
}
