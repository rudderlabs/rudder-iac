package connection

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newHandler builds a spec-side handler: loading and the graph never reach the
// client or the registry, so both stay nil and any accidental call panics.
func newHandler() *Handler {
	return NewHandler(nil, "retl", nil)
}

// connectionsSpec wraps a spec body in the specs.Spec shape handlers receive.
func connectionsSpec(body map[string]any) *specs.Spec {
	return &specs.Spec{Version: specs.SpecVersionV1, Kind: ResourceKind, Spec: body}
}

// specEntry is one connection entry whose config is exactly jsonMapperConfig(),
// so what the handler stores can be compared against the shared fixture.
func specEntry(overrides map[string]any) map[string]any {
	entry := map[string]any{
		"id":          "users-to-webhook",
		"source":      "#retl-source-sql-model:users",
		"destination": "#destination:webhook",
		"config": map[string]any{
			"sync_behaviour": "upsert",
			"schedule":       map[string]any{"type": "basic", "every_minutes": 30},
			"identifiers":    []any{map[string]any{"from": "id", "to": "user_id"}},
			"mappings":       []any{map[string]any{"from": "email", "to": "traits.email"}},
		},
	}
	for key, value := range overrides {
		if value == nil {
			delete(entry, key)
			continue
		}
		entry[key] = value
	}
	return entry
}

func specBody(entries ...map[string]any) map[string]any {
	items := make([]any, len(entries))
	for i, entry := range entries {
		items[i] = entry
	}
	return map[string]any{ConnectionsKey: items}
}

func TestParseSpec(t *testing.T) {
	t.Parallel()

	t.Run("one urn per entry", func(t *testing.T) {
		t.Parallel()

		parsed, err := newHandler().ParseSpec("", connectionsSpec(specBody(
			specEntry(map[string]any{"id": "one"}),
			specEntry(map[string]any{"id": "two"}),
		)))
		require.NoError(t, err)

		// No LegacyResourceType: the kind never had a rudder/0.1 form, so a
		// local_id import must not resolve against it.
		assert.Equal(t, &specs.ParsedSpec{URNs: []specs.URNEntry{
			{URN: "retl-connection:one", JSONPointerPath: "/spec/connections/0/id"},
			{URN: "retl-connection:two", JSONPointerPath: "/spec/connections/1/id"},
		}}, parsed)
	})

	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{"missing connections", map[string]any{}, "connections not found in rETL connections spec"},
		{"entry is not a map", map[string]any{ConnectionsKey: []any{"nope"}}, "connection at index 0 is not a map"},
		{"entry without an id", map[string]any{ConnectionsKey: []any{map[string]any{"source": "#retl-source-sql-model:users"}}}, "id not found in connection at index 0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := newHandler().ParseSpec("", connectionsSpec(test.body))
			assert.ErrorContains(t, err, test.wantErr)
			assert.Nil(t, parsed)
		})
	}
}

func TestLoadSpec(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(specBody(specEntry(nil)))))
	require.Len(t, h.resources, 1)
	resource := h.resources["users-to-webhook"]

	config, err := configToData(jsonMapperConfig())
	require.NoError(t, err)

	assert.Equal(t, "users-to-webhook", resource.LocalID)
	assert.True(t, resource.Enabled)
	assert.Equal(t, config, resource.Config)
	assert.Equal(t, &resources.PropertyRef{URN: "retl-source-sql-model:users", Property: "id"}, resource.Source)
	// The destination ref carries a Resolve func, which no struct comparison
	// can reach; its behaviour is pinned by TestDestinationRefResolve.
	assert.Equal(t, "destination:webhook", resource.Destination.URN)
	assert.Equal(t, "id", resource.Destination.Property)
	require.NotNil(t, resource.Destination.Resolve)
}

// TestLoadSpecStrictDecoding pins that an unknown key is refused however deep
// it sits. A dropped key is a setting the user believes is applied, and
// destination_config in particular names a real API field the spec cannot
// express at all.
func TestLoadSpecStrictDecoding(t *testing.T) {
	t.Parallel()

	config := func(overrides map[string]any) map[string]any {
		entry := specEntry(nil)
		body := entry["config"].(map[string]any)
		for key, value := range overrides {
			body[key] = value
		}
		return entry
	}

	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name:    "unknown top level key",
			body:    map[string]any{ConnectionsKey: []any{}, "links": []any{}},
			wantErr: "invalid keys: links",
		},
		{
			name:    "unknown entry key",
			body:    specBody(specEntry(map[string]any{"enbaled": true})),
			wantErr: "invalid keys: enbaled",
		},
		{
			name:    "destination config has no spec equivalent",
			body:    specBody(config(map[string]any{"destination_config": map[string]any{"listId": "42"}})),
			wantErr: "invalid keys: destination_config",
		},
		{
			name:    "unknown schedule key",
			body:    specBody(config(map[string]any{"schedule": map[string]any{"type": "basic", "every_minute": 30}})),
			wantErr: "invalid keys: every_minute",
		},
		{
			name:    "unknown event key",
			body:    specBody(config(map[string]any{"event": map[string]any{"type": "track", "nmae": "signup"}})),
			wantErr: "invalid keys: nmae",
		},
		{
			name: "unknown sync settings key three levels down",
			body: specBody(config(map[string]any{
				"sync_settings": map[string]any{"sync_logs": map[string]any{"retetion_days": 90}},
			})),
			wantErr: "invalid keys: retetion_days",
		},
		{
			name: "unknown key inside a list member",
			body: specBody(config(map[string]any{
				"identifiers": []any{map[string]any{"form": "id", "to": "user_id"}},
			})),
			wantErr: "invalid keys: form",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			h := newHandler()
			assert.ErrorContains(t, h.LoadSpec("", connectionsSpec(test.body)), test.wantErr)
			assert.Empty(t, h.resources)
		})
	}
}

func TestLoadSpecEnabledDefaultsToTrue(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(specBody(
		specEntry(map[string]any{"id": "omitted"}),
		specEntry(map[string]any{"id": "disabled", "enabled": false}),
	))))

	assert.True(t, h.resources["omitted"].Enabled)
	assert.False(t, h.resources["disabled"].Enabled)
}

func TestLoadSpecRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()

	t.Run("within one spec", func(t *testing.T) {
		t.Parallel()

		err := newHandler().LoadSpec("", connectionsSpec(specBody(specEntry(nil), specEntry(nil))))
		assert.ErrorContains(t, err, "rETL connection with id users-to-webhook already exists")
	})

	t.Run("across spec files", func(t *testing.T) {
		t.Parallel()

		h := newHandler()
		require.NoError(t, h.LoadSpec("first.yaml", connectionsSpec(specBody(specEntry(nil)))))
		err := h.LoadSpec("second.yaml", connectionsSpec(specBody(specEntry(nil))))
		assert.ErrorContains(t, err, "rETL connection with id users-to-webhook already exists")
	})
}

func TestLoadSpecReferenceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name:    "missing source",
			body:    specBody(specEntry(map[string]any{"source": nil})),
			wantErr: `connection "users-to-webhook": parsing source reference: invalid source reference ""`,
		},
		{
			name:    "source of the wrong family",
			body:    specBody(specEntry(map[string]any{"source": "#event-stream-source:android"})),
			wantErr: `connection "users-to-webhook": parsing source reference: source reference "#event-stream-source:android" is not a rETL source`,
		},
		{
			name:    "missing destination",
			body:    specBody(specEntry(map[string]any{"destination": nil})),
			wantErr: `connection "users-to-webhook": parsing destination reference: invalid reference "": expected format #destination:<id>`,
		},
		{
			name:    "destination of the wrong kind",
			body:    specBody(specEntry(map[string]any{"destination": "#retl-source-sql-model:users"})),
			wantErr: `connection "users-to-webhook": parsing destination reference: invalid reference "#retl-source-sql-model:users": expected format #destination:<id>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			h := newHandler()
			assert.ErrorContains(t, h.LoadSpec("", connectionsSpec(test.body)), test.wantErr)
			assert.Empty(t, h.resources)
		})
	}
}

func TestDestinationRefResolve(t *testing.T) {
	t.Parallel()

	ref, err := parseDestinationRef("#destination:webhook")
	require.NoError(t, err)

	t.Run("resolves the remote id from the destination state", func(t *testing.T) {
		t.Parallel()

		value, err := ref.Resolve(&destination.DestinationState{ID: "dst-remote-1"})
		require.NoError(t, err)
		assert.Equal(t, "dst-remote-1", value)
	})

	t.Run("empty remote id errors", func(t *testing.T) {
		t.Parallel()

		_, err := ref.Resolve(&destination.DestinationState{})
		assert.ErrorContains(t, err, "destination state has empty ID")
	})

	t.Run("unexpected state type errors", func(t *testing.T) {
		t.Parallel()

		_, err := ref.Resolve("not-a-destination-state")
		assert.ErrorContains(t, err, "invalid resource data type")
	})
}

// importedSpec is the example spec with the inline metadata.import block the
// import command writes back.
func importedSpec() *specs.Spec {
	spec := connectionsSpec(specBody(specEntry(nil)))
	spec.Metadata = map[string]any{
		"name": MetadataName,
		"import": map[string]any{
			"workspaces": []any{
				map[string]any{
					"workspace_id": "ws-1",
					"resources": []any{
						map[string]any{"urn": "retl-connection:users-to-webhook", "remote_id": "conn-remote-9"},
					},
				},
			},
		},
	}
	return spec
}

func TestLoadSpecImportMetadata(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", importedSpec()))

	assert.Equal(t,
		map[string]*WorkspaceRemoteIDMapping{
			"retl-connection:users-to-webhook": {WorkspaceID: "ws-1", RemoteID: "conn-remote-9"},
		},
		h.resources["users-to-webhook"].ImportMetadata,
	)
}

func TestLoadImportMetadata(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(specBody(specEntry(nil)))))
	require.NoError(t, h.LoadImportMetadata(nil))
	assert.Empty(t, h.resources["users-to-webhook"].ImportMetadata)

	require.NoError(t, h.LoadImportMetadata(&specs.WorkspacesImportMetadata{
		Workspaces: []specs.WorkspaceImportMetadata{{
			WorkspaceID: "ws-1",
			Resources: []specs.ImportIds{
				{URN: "retl-connection:users-to-webhook", RemoteID: "conn-remote-9"},
				// A legacy entry names the local id only; it has to expand to
				// the same URN the graph looks up.
				{LocalID: "orders-to-webhook", RemoteID: "conn-remote-8"},
			},
		}},
	}))

	assert.Equal(t,
		map[string]*WorkspaceRemoteIDMapping{
			"retl-connection:users-to-webhook":  {WorkspaceID: "ws-1", RemoteID: "conn-remote-9"},
			"retl-connection:orders-to-webhook": {WorkspaceID: "ws-1", RemoteID: "conn-remote-8"},
		},
		h.resources["users-to-webhook"].ImportMetadata,
	)
}

func TestGetResources(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(specBody(specEntry(nil)))))

	graphResources, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, graphResources, 1)

	resource := graphResources[0]
	config, err := configToData(jsonMapperConfig())
	require.NoError(t, err)

	assert.Equal(t, "retl-connection:users-to-webhook", resource.URN())
	assert.Equal(t, config, resource.Data()[ConfigKey])
	assert.Equal(t, true, resource.Data()[EnabledKey])
	assert.Equal(t, h.resources["users-to-webhook"].Source, resource.Data()[SourceKey])
	assert.Equal(t, h.resources["users-to-webhook"].Destination, resource.Data()[DestinationKey])
	assert.Equal(t, "#retl-connections:users-to-webhook", resource.FileMetadata().MetadataRef)
	assert.Nil(t, resource.ImportMetadata())
}

func TestGetResourcesImportMetadata(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", importedSpec()))

	graphResources, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, graphResources, 1)

	assert.Equal(t, "conn-remote-9", graphResources[0].ImportMetadata().RemoteId)
	assert.Equal(t, "ws-1", graphResources[0].ImportMetadata().WorkspaceId)
}

// TestGetResourcesGraphDependencies proves the graph derives its edges from the
// endpoint PropertyRefs: the connection is created after both endpoints and
// deleted before them.
func TestGetResourcesGraphDependencies(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(specBody(specEntry(nil)))))
	graphResources, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, graphResources, 1)

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("users", sqlmodel.ResourceType, resources.ResourceData{}, []string{}))
	graph.AddResource(resources.NewResource("webhook", destination.DestinationResourceType, resources.ResourceData{}, []string{}))
	graph.AddResource(graphResources[0])

	connectionURN := "retl-connection:users-to-webhook"
	assert.ElementsMatch(t,
		[]string{"retl-source-sql-model:users", "destination:webhook"},
		graph.GetDependencies(connectionURN),
	)
	assert.Equal(t, []string{connectionURN}, graph.GetDependents("retl-source-sql-model:users"))
	assert.Equal(t, []string{connectionURN}, graph.GetDependents("destination:webhook"))

	cycle, err := graph.DetectCycles()
	require.NoError(t, err)
	assert.Nil(t, cycle)
}

// TestConnectionRefResolution proves both endpoint references survive the
// dereference the syncer applies before the lifecycle runs: the SQL model
// through its output map, the destination through its typed state.
func TestConnectionRefResolution(t *testing.T) {
	t.Parallel()

	h := newHandler()
	require.NoError(t, h.LoadSpec("", connectionsSpec(specBody(specEntry(nil)))))
	graphResources, err := h.GetResources()
	require.NoError(t, err)
	require.Len(t, graphResources, 1)

	st := state.EmptyState()
	st.AddResource(&state.ResourceState{
		ID:     "users",
		Type:   sqlmodel.ResourceType,
		Output: map[string]any{"id": "src-remote-1"},
	})
	st.AddResource(&state.ResourceState{
		ID:        "webhook",
		Type:      destination.DestinationResourceType,
		OutputRaw: &destination.DestinationState{ID: "dst-remote-1"},
	})

	dereferenced, err := state.Dereference(graphResources[0].Data(), st)
	require.NoError(t, err)

	config, err := configToData(jsonMapperConfig())
	require.NoError(t, err)
	assert.Equal(t, resources.ResourceData{
		SourceKey:      "src-remote-1",
		DestinationKey: "dst-remote-1",
		EnabledKey:     true,
		ConfigKey:      config,
	}, dereferenced)
}

// TestUnsupportedMethodsReturnErrors pins the three operations the connection
// handler refuses outright: preview is a SQL model capability, and both import
// paths wait on DEX-827.
func TestUnsupportedMethodsReturnErrors(t *testing.T) {
	t.Parallel()

	h := newHandler()
	ctx := context.Background()

	_, err := h.Import(ctx, "users-to-webhook", resources.ResourceData{}, "conn-remote-1")
	assert.ErrorContains(t, err, "importing rETL connections is not supported yet")

	_, err = h.Preview(ctx, "users-to-webhook", resources.ResourceData{}, 10)
	assert.ErrorContains(t, err, "preview is not supported for rETL connections")

	_, err = h.FetchImportData(ctx, specs.ImportIds{URN: "retl-connection:users-to-webhook", RemoteID: "conn-remote-1"})
	assert.ErrorContains(t, err, "importing a single rETL connection is not supported")
}
