package connection

import (
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
// client, so it stays nil and any accidental call panics.
func newHandler() *Handler {
	return NewHandler(nil)
}

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

	config, err := configToMap(jsonMapperConfig())
	require.NoError(t, err)

	// The destination ref carries a Resolve func no struct comparison can
	// reach; TestConnectionRefResolution exercises it.
	destinationRef := resource.Destination
	resource.Destination = nil
	assert.Equal(t, &connectionResource{
		LocalID:        "users-to-webhook",
		Source:         &resources.PropertyRef{URN: "retl-source-sql-model:users", Property: "id"},
		Enabled:        true,
		Config:         config,
		ImportMetadata: map[string]*WorkspaceRemoteIDMapping{},
	}, resource)
	assert.Equal(t, "destination:webhook", destinationRef.URN)
	assert.Equal(t, "id", destinationRef.Property)
}

// TestLoadSpecStrictDecoding pins that an unknown key is refused at any depth:
// destination_config in particular is a real API field the spec cannot express.
func TestLoadSpecStrictDecoding(t *testing.T) {
	t.Parallel()

	withDestinationConfig := specEntry(nil)
	withDestinationConfig["config"].(map[string]any)["destination_config"] = map[string]any{"listId": "42"}

	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{"unknown top level key", map[string]any{ConnectionsKey: []any{}, "links": []any{}}, "invalid keys: links"},
		{"unknown entry key", specBody(specEntry(map[string]any{"enbaled": true})), "invalid keys: enbaled"},
		{"unknown nested config key", specBody(withDestinationConfig), "invalid keys: destination_config"},
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

	h := newHandler()
	require.NoError(t, h.LoadSpec("first.yaml", connectionsSpec(specBody(specEntry(nil)))))
	err := h.LoadSpec("second.yaml", connectionsSpec(specBody(specEntry(nil))))
	assert.ErrorContains(t, err, "rETL connection with id users-to-webhook already exists")
}

func TestLoadSpecReferenceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		body    map[string]any
		wantErr string
	}{
		{
			name:    "source of the wrong family",
			body:    specBody(specEntry(map[string]any{"source": "#event-stream-source:android"})),
			wantErr: `connection "users-to-webhook": parsing source reference: source reference "#event-stream-source:android" is not a rETL source`,
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
	config, err := configToMap(jsonMapperConfig())
	require.NoError(t, err)

	assert.Equal(t, "retl-connection:users-to-webhook", resource.URN())
	assert.Equal(t, resources.ResourceData{
		SourceKey:      h.resources["users-to-webhook"].Source,
		DestinationKey: h.resources["users-to-webhook"].Destination,
		EnabledKey:     true,
		ConfigKey:      config,
	}, resource.Data())
	assert.Equal(t, "#retl-connections:users-to-webhook", resource.FileMetadata().MetadataRef)
	assert.Nil(t, resource.ImportMetadata())

	imported := newHandler()
	require.NoError(t, imported.LoadSpec("", importedSpec()))
	graphResources, err = imported.GetResources()
	require.NoError(t, err)
	require.Len(t, graphResources, 1)
	assert.Equal(t, "conn-remote-9", graphResources[0].ImportMetadata().RemoteId)
	assert.Equal(t, "ws-1", graphResources[0].ImportMetadata().WorkspaceId)
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

	config, err := configToMap(jsonMapperConfig())
	require.NoError(t, err)
	assert.Equal(t, resources.ResourceData{
		SourceKey:      "src-remote-1",
		DestinationKey: "dst-remote-1",
		EnabledKey:     true,
		ConfigKey:      config,
	}, dereferenced)
}
