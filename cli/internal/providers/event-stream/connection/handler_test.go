package connection

import (
	"testing"

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
		Kind:    ResourceKind,
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
	h := NewHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(ticketExampleSpec())))
	require.Len(t, h.resources, 1)
	return h.resources["android-to-s3"]
}

func TestParseSpec(t *testing.T) {
	t.Run("one URN per connection entry", func(t *testing.T) {
		h := NewHandler()
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
			h := NewHandler()
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
	h := NewHandler()
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
	h := NewHandler()
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
			name:    "missing connections key",
			body:    map[string]any{},
			wantErr: "connections not found in event stream connections spec",
		},
		{
			name:    "missing id",
			body:    map[string]any{"connections": []any{entry(map[string]any{"id": nil})}},
			wantErr: "id is required for connection at index 0",
		},
		{
			name:    "missing source",
			body:    map[string]any{"connections": []any{entry(map[string]any{"source": nil})}},
			wantErr: `source is required for connection "one"`,
		},
		{
			name:    "missing destination",
			body:    map[string]any{"connections": []any{entry(map[string]any{"destination": nil})}},
			wantErr: `destination is required for connection "one"`,
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
			h := NewHandler()
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
	h := NewHandler()
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
// referenced resource's remote id from state: the source via the legacy output
// map, the destination via its typed state.
func TestConnectionRefResolution(t *testing.T) {
	resource := loadTicketExample(t)

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

	require.NoError(t, state.DereferenceByReflection(resource, st))

	assert.True(t, resource.Source.IsResolved)
	assert.Equal(t, "src-remote-id", resource.Source.Value)
	assert.True(t, resource.Destination.IsResolved)
	assert.Equal(t, "dst-remote-id", resource.Destination.Value)
}

func TestMigrateSpecIsIdentity(t *testing.T) {
	h := NewHandler()
	spec := connectionsSpec(ticketExampleSpec())
	migrated, err := h.MigrateSpec(spec)
	require.NoError(t, err)
	assert.Same(t, spec, migrated)
}

func TestLoadSpecRejectsInlineImportMetadata(t *testing.T) {
	h := NewHandler()
	spec := connectionsSpec(ticketExampleSpec())
	spec.Metadata = map[string]any{
		"name": "app-connections",
		"import": map[string]any{
			"workspaces": []any{map[string]any{
				"workspace_id": "ws-1",
				"resources": []any{map[string]any{
					"urn":       "event-stream-connection:android-to-s3",
					"remote_id": "remote-1",
				}},
			}},
		},
	}
	err := h.LoadSpec("", spec)
	assert.ErrorContains(t, err, "import metadata is not supported for event-stream-connections yet")
}

func TestLoadImportMetadata(t *testing.T) {
	h := NewHandler()
	assert.NoError(t, h.LoadImportMetadata(nil))
	assert.NoError(t, h.LoadImportMetadata(&specs.WorkspacesImportMetadata{}))

	t.Run("entries for other kinds pass through", func(t *testing.T) {
		assert.NoError(t, h.LoadImportMetadata(&specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{{
				WorkspaceID: "ws-1",
				Resources: []specs.ImportIds{
					{URN: "event-stream-source:my-source", RemoteID: "remote-1"},
				},
			}},
		}))
	})

	t.Run("entries targeting connections are rejected", func(t *testing.T) {
		err := h.LoadImportMetadata(&specs.WorkspacesImportMetadata{
			Workspaces: []specs.WorkspaceImportMetadata{{
				WorkspaceID: "ws-1",
				Resources: []specs.ImportIds{
					{URN: "event-stream-connection:android-to-s3", RemoteID: "remote-1"},
				},
			}},
		})
		assert.ErrorContains(t, err, "import is not supported for event-stream-connections yet")
	})
}

func TestLifecycleNotImplemented(t *testing.T) {
	h := NewHandler()

	_, err := h.Create(t.Context(), "id", resources.ResourceData{})
	assert.ErrorIs(t, err, ErrNotImplemented)
	_, err = h.Update(t.Context(), "id", resources.ResourceData{}, resources.ResourceData{})
	assert.ErrorIs(t, err, ErrNotImplemented)
	err = h.Delete(t.Context(), "id", resources.ResourceData{})
	assert.ErrorIs(t, err, ErrNotImplemented)
	_, err = h.List(t.Context(), nil)
	assert.ErrorIs(t, err, ErrNotImplemented)
	_, err = h.Import(t.Context(), "id", resources.ResourceData{}, "remote-id")
	assert.ErrorIs(t, err, ErrNotImplemented)
}

func TestRemoteLoadsAreEmptyUntilImplemented(t *testing.T) {
	h := NewHandler()

	remote, err := h.LoadResourcesFromRemote(t.Context())
	require.NoError(t, err)
	assert.Empty(t, remote.GetAll(ResourceType))

	mapped, err := h.MapRemoteToState(resources.NewRemoteResources())
	require.NoError(t, err)
	assert.Empty(t, mapped.Resources)

	importable, err := h.LoadImportable(t.Context(), nil)
	require.NoError(t, err)
	assert.Empty(t, importable.GetAll(ResourceType))

	entities, entries, err := h.FormatForExport(resources.NewRemoteResources(), nil, nil)
	require.NoError(t, err)
	assert.Nil(t, entities)
	assert.Nil(t, entries)
}
