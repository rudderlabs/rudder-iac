package connection

import (
	"errors"
	"strings"
	"testing"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNamer kebab-cases the human name, mirroring the real namer's strategy.
type mockNamer struct{}

func (m *mockNamer) Name(input namer.ScopeName) (string, error) {
	return strings.ToLower(strings.ReplaceAll(input.Name, " ", "-")), nil
}

func (m *mockNamer) Load(_ []namer.ScopeName) error {
	return nil
}

// mockResolver resolves entityType/remoteID pairs from a canned map and errors
// on anything else, like the real ImportRefResolver does for resources that are
// neither importable nor CLI-managed.
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
	t.Parallel()

	t.Run("names connections after their endpoints", func(t *testing.T) {
		t.Parallel()

		conn := remoteRow("conn-1", "", "src-1", "dst-1")
		mock := remoteClient([]retlClient.RETLConnection{
			conn,
			remoteRow("conn-specific", "", "src-1", "dst-specific"),
		})
		config, err := configFromRemote(&conn)
		require.NoError(t, err)

		collection, err := remoteHandler(mock, t).LoadImportable(t.Context(), &mockNamer{})
		require.NoError(t, err)

		require.Len(t, mock.ListCalls, 1)
		assert.Equal(t, lo.ToPtr(false), mock.ListCalls[0].HasExternalID, "must list only connections without an externalId")
		assert.Equal(t, map[string]*resources.RemoteResource{
			"conn-1": {
				ID:         "conn-1",
				ExternalID: "users-to-webhook",
				Reference:  "#retl-connections:users-to-webhook",
				Data: &RemoteConnection{
					RETLConnection:        conn,
					Config:                config,
					WorkspaceID:           "ws-1",
					SourceKind:            SourceKinds[0],
					SourceName:            "Users",
					SourceExternalID:      "users",
					DestinationName:       "Webhook",
					DestinationExternalID: "webhook",
				},
			},
		}, collection.GetAll(ResourceType))
	})
}

// importableConnection builds one importable remote connection the way
// LoadImportable stores it, config included — remoteConnection rebuilds it
// before the row is ever stored, so a fixture without it is not one the export
// could receive.
func importableConnection(t *testing.T, remoteID, externalID, sourceID, destinationID string) *resources.RemoteResource {
	t.Helper()

	row := remoteRow(remoteID, "", sourceID, destinationID)
	config, err := configFromRemote(&row)
	require.NoError(t, err)

	return &resources.RemoteResource{
		ID:         remoteID,
		ExternalID: externalID,
		Reference:  "#" + ResourceKind + ":" + externalID,
		Data: &RemoteConnection{
			RETLConnection: row,
			Config:         config,
			WorkspaceID:    "ws-1",
			SourceKind:     SourceKinds[0],
		},
	}
}

// exportedEntry is one connection as FormatForExport emits it, so a test can
// compare the whole emitted list instead of probing one field of it.
func exportedEntry(t *testing.T, id, source, destination string, enabled bool) map[string]any {
	t.Helper()

	config, err := configToMap(jsonMapperConfig())
	require.NoError(t, err)
	return map[string]any{
		"id":          id,
		"source":      source,
		"destination": destination,
		"enabled":     enabled,
		"config":      config,
	}
}

func connectionCollection(remotes ...*resources.RemoteResource) *resources.RemoteResources {
	collection := resources.NewRemoteResources()
	resourceMap := make(map[string]*resources.RemoteResource, len(remotes))
	for _, remote := range remotes {
		resourceMap[remote.ID] = remote
	}
	collection.Set(ResourceType, resourceMap)
	return collection
}

func TestFormatForExport(t *testing.T) {
	t.Parallel()

	refs := map[string]string{
		sqlmodel.ResourceType + "/src-1":               "#retl-source-sql-model:users",
		sqlmodel.ResourceType + "/src-2":               "#retl-source-sql-model:orders",
		destination.DestinationResourceType + "/dst-1": "#destination:webhook",
	}

	t.Run("writes one spec per run sorted by id", func(t *testing.T) {
		t.Parallel()

		// This row mixes the two endpoint cases: its source is imported in the
		// same run, so the resolver serves it, while its destination is already
		// CLI-managed and the resolver cannot — its externalId builds the ref.
		mixedEndpoints := importableConnection(t, "conn-2", "orders-to-webhook", "src-2", "dst-9")
		mixed := mixedEndpoints.Data.(*RemoteConnection)
		mixed.DestinationExternalID = "webhook"
		mixed.Enabled = false

		entities, entries, err := remoteHandler(remoteClient(), t).FormatForExport(
			connectionCollection(mixedEndpoints, importableConnection(t, "conn-1", "users-to-webhook", "src-1", "dst-1")),
			&mockNamer{},
			&mockResolver{refs: refs},
		)
		require.NoError(t, err)

		// Nested snake_case config, and no sync_settings: the row carries the
		// settings a create would have defaulted to anyway.
		config, err := configToMap(jsonMapperConfig())
		require.NoError(t, err)
		assert.Equal(t, []writer.FormattableEntity{{
			RelativePath: "retl/connections.yaml",
			Content: &specs.Spec{
				Version: specs.SpecVersionV1,
				Kind:    ResourceKind,
				Metadata: map[string]any{
					"name": MetadataName,
					"import": map[string]any{
						"workspaces": []any{
							map[string]any{
								"workspace_id": "ws-1",
								"resources": []any{
									map[string]any{"urn": "retl-connection:orders-to-webhook", "remote_id": "conn-2"},
									map[string]any{"urn": "retl-connection:users-to-webhook", "remote_id": "conn-1"},
								},
							},
						},
					},
				},
				Spec: map[string]any{
					ConnectionsKey: []map[string]any{
						{
							"id":          "orders-to-webhook",
							"source":      "#retl-source-sql-model:orders",
							"destination": "#destination:webhook",
							"enabled":     false,
							"config":      config,
						},
						{
							"id":          "users-to-webhook",
							"source":      "#retl-source-sql-model:users",
							"destination": "#destination:webhook",
							"enabled":     true,
							"config":      config,
						},
					},
				},
			},
		}}, entities)

		assert.Equal(t, []importmanifest.ImportEntry{
			{WorkspaceID: "ws-1", URN: "retl-connection:orders-to-webhook", RemoteID: "conn-2"},
			{WorkspaceID: "ws-1", URN: "retl-connection:users-to-webhook", RemoteID: "conn-1"},
		}, entries)
	})

	t.Run("matched connections write manifest entries only", func(t *testing.T) {
		t.Parallel()

		matched := importableConnection(t, "conn-2", "existing-conn", "src-2", "dst-1")
		matched.MatchedWith = resources.NewResource("existing-conn", ResourceType, resources.ResourceData{}, []string{})

		entities, entries, err := remoteHandler(remoteClient(), t).FormatForExport(
			connectionCollection(importableConnection(t, "conn-1", "users-to-webhook", "src-1", "dst-1"), matched),
			&mockNamer{},
			&mockResolver{refs: refs},
		)
		require.NoError(t, err)

		require.Len(t, entities, 1)
		assert.Equal(t, []map[string]any{
			exportedEntry(t, "users-to-webhook", "#retl-source-sql-model:users", "#destination:webhook", true),
		}, entities[0].Content.(*specs.Spec).Spec[ConnectionsKey])
		assert.ElementsMatch(t, []importmanifest.ImportEntry{
			{WorkspaceID: "ws-1", URN: "retl-connection:users-to-webhook", RemoteID: "conn-1"},
			{WorkspaceID: "ws-1", URN: "retl-connection:existing-conn", RemoteID: "conn-2"},
		}, entries)
	})

	t.Run("skips connections whose endpoint cannot be resolved", func(t *testing.T) {
		t.Parallel()

		// dst-9 resolves to neither an importable nor a CLI-managed destination
		// and carries no externalId, so the connection cannot be expressed as
		// spec refs: no spec entry, no manifest entry, and no invented ref.
		entities, entries, err := remoteHandler(remoteClient(), t).FormatForExport(
			connectionCollection(
				importableConnection(t, "conn-1", "users-to-webhook", "src-1", "dst-1"),
				importableConnection(t, "conn-2", "users-to-unmanaged", "src-1", "dst-9"),
			),
			&mockNamer{},
			&mockResolver{refs: refs},
		)
		require.NoError(t, err)

		require.Len(t, entities, 1)
		assert.Equal(t, []map[string]any{
			exportedEntry(t, "users-to-webhook", "#retl-source-sql-model:users", "#destination:webhook", true),
		}, entities[0].Content.(*specs.Spec).Spec[ConnectionsKey])
		assert.Equal(t, []importmanifest.ImportEntry{
			{WorkspaceID: "ws-1", URN: "retl-connection:users-to-webhook", RemoteID: "conn-1"},
		}, entries)
	})

	// Export is only worth anything if the project it writes loads: the spec
	// goes straight back through this handler's own LoadSpec, import metadata
	// included, and has to land on the resource the connection started as.
	t.Run("the spec it writes loads back through LoadSpec", func(t *testing.T) {
		t.Parallel()

		entities, _, err := remoteHandler(remoteClient(), t).FormatForExport(
			connectionCollection(importableConnection(t, "conn-1", "users-to-webhook", "src-1", "dst-1")),
			&mockNamer{},
			&mockResolver{refs: refs},
		)
		require.NoError(t, err)
		require.Len(t, entities, 1)
		spec, ok := entities[0].Content.(*specs.Spec)
		require.True(t, ok)

		loaded := newHandler()
		require.NoError(t, loaded.LoadSpec("", spec))
		require.Len(t, loaded.resources, 1)

		config, err := configToMap(jsonMapperConfig())
		require.NoError(t, err)

		// The destination ref carries a Resolve func no struct comparison can
		// reach; TestConnectionRefResolution exercises it.
		resource := loaded.resources["users-to-webhook"]
		destinationRef := resource.Destination
		resource.Destination = nil
		assert.Equal(t, &connectionResource{
			LocalID: "users-to-webhook",
			Source:  &resources.PropertyRef{URN: "retl-source-sql-model:users", Property: "id"},
			Enabled: true,
			Config:  config,
			ImportMetadata: map[string]*WorkspaceRemoteIDMapping{
				"retl-connection:users-to-webhook": {WorkspaceID: "ws-1", RemoteID: "conn-1"},
			},
		}, resource)
		assert.Equal(t, "destination:webhook", destinationRef.URN)
	})

	t.Run("empty collection writes nothing", func(t *testing.T) {
		t.Parallel()

		entities, entries, err := remoteHandler(remoteClient(), t).FormatForExport(resources.NewRemoteResources(), &mockNamer{}, &mockResolver{})

		require.NoError(t, err)
		assert.Nil(t, entities)
		assert.Nil(t, entries)
	})

	t.Run("errors on foreign data in the collection", func(t *testing.T) {
		t.Parallel()

		collection := resources.NewRemoteResources()
		collection.Set(ResourceType, map[string]*resources.RemoteResource{
			"x": {ID: "x", ExternalID: "x", Data: "not a connection"},
		})

		_, _, err := remoteHandler(remoteClient(), t).FormatForExport(collection, &mockNamer{}, &mockResolver{refs: refs})

		assert.ErrorContains(t, err, "unable to cast remote resource to rETL connection")
	})

	t.Run("errors on connections from multiple workspaces", func(t *testing.T) {
		t.Parallel()

		other := importableConnection(t, "conn-2", "orders-to-webhook", "src-2", "dst-1")
		other.Data.(*RemoteConnection).WorkspaceID = "ws-2"

		_, _, err := remoteHandler(remoteClient(), t).FormatForExport(
			connectionCollection(importableConnection(t, "conn-1", "users-to-webhook", "src-1", "dst-1"), other),
			&mockNamer{},
			&mockResolver{refs: refs},
		)

		assert.ErrorContains(t, err, "cannot export resources from multiple workspaces into a single spec file")
	})
}

// importClient serves one remote row plus the shared endpoint catalogs.
func importClient(remote retlClient.RETLConnection) *MockConnectionClient {
	return &MockConnectionClient{
		Sources:      remoteSources(),
		Destinations: remoteDestinations(),
		GetFunc: func(id string) (*retlClient.RETLConnection, error) {
			conn := remote
			conn.ID = id
			return &conn, nil
		},
	}
}

func TestImport(t *testing.T) {
	t.Parallel()

	t.Run("an unchanged row is only claimed", func(t *testing.T) {
		t.Parallel()

		mock := importClient(remoteRow("conn-remote-1", "", "src-1", "dst-1"))

		result, err := remoteHandler(mock, t).Import(t.Context(), localID, graphData(t, jsonMapperConfig()), "conn-remote-1")
		require.NoError(t, err)

		assert.Empty(t, mock.UpdateCalls)
		assert.Empty(t, mock.CreateCalls)
		assert.Empty(t, mock.DeleteCalls)
		assert.Equal(t, []retlClient.SetRETLConnectionExternalIDRequest{{ID: "conn-remote-1", ExternalID: localID}}, mock.SetExternalIDCalls)
		assert.Equal(t, remoteIDs, result)
	})

	t.Run("a mutable difference is reconciled before the claim", func(t *testing.T) {
		t.Parallel()

		remote := remoteRow("conn-remote-1", "", "src-1", "dst-1")
		remote.Enabled = false
		mock := importClient(remote)
		mock.UpdateFunc = func(id string, _ *retlClient.UpdateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			return &retlClient.RETLConnection{ID: id, SourceID: "src-1", DestinationID: "dst-1"}, nil
		}

		result, err := remoteHandler(mock, t).Import(t.Context(), localID, graphData(t, jsonMapperConfig()), "conn-remote-1")
		require.NoError(t, err)

		assert.Equal(t, []string{"conn-remote-1"}, mock.UpdateCalls)
		assert.Equal(t, []retlClient.SetRETLConnectionExternalIDRequest{{ID: "conn-remote-1", ExternalID: localID}}, mock.SetExternalIDCalls)
		assert.Equal(t, remoteIDs, result)
	})

	// An endpoint change is the only difference a replacement answers, and the
	// create body it goes through already carries the externalId, so the row it
	// returns is this connection's without a separate claim.
	t.Run("an endpoint change replaces the row and claims nothing", func(t *testing.T) {
		t.Parallel()

		// The spec aims the connection at a destination the remote row does not
		// carry, which is what makes this a replacement.
		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"

		mock := importClient(remoteRow("conn-remote-1", "", "src-1", "dst-1"))
		// A recreated row is a row of its own, hence an id of its own: the pair
		// it belongs to is one this connection has never held.
		mock.CreateFunc = func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			return &retlClient.RETLConnection{
				ID:            "conn-remote-2",
				SourceID:      req.SourceID,
				DestinationID: req.DestinationID,
				ExternalID:    req.ExternalID,
			}, nil
		}

		result, err := remoteHandler(mock, t).Import(t.Context(), localID, data, "conn-remote-1")
		require.NoError(t, err)

		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, localID, mock.CreateCalls[0].ExternalID)
		assert.Equal(t, []string{"conn-remote-1"}, mock.DeleteCalls, "only the replacement deletes")
		assert.Empty(t, mock.SetExternalIDCalls, "the create body carried the external id")
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-2",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-2",
		}, result)
	})

	// Nothing stops the backend handing a replacement the id it just freed, so
	// the returned id cannot tell a replacement from an adoption. Which path ran
	// is the only reliable answer, and this pins it: deciding on
	// claimed != remote.ID instead would claim a row Create already owns.
	t.Run("a replacement that revives the same id still claims nothing", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		data[DestinationKey] = "dst-2"

		mock := importClient(remoteRow("conn-remote-1", "", "src-1", "dst-1"))
		mock.CreateFunc = func(req *retlClient.CreateRETLConnectionRequest) (*retlClient.RETLConnection, error) {
			return &retlClient.RETLConnection{
				ID:            "conn-remote-1",
				SourceID:      req.SourceID,
				DestinationID: req.DestinationID,
				ExternalID:    req.ExternalID,
			}, nil
		}

		result, err := remoteHandler(mock, t).Import(t.Context(), localID, data, "conn-remote-1")
		require.NoError(t, err)

		require.Len(t, mock.CreateCalls, 1)
		assert.Equal(t, localID, mock.CreateCalls[0].ExternalID)
		assert.Empty(t, mock.SetExternalIDCalls, "the create body carried the external id, revived row or not")
		assert.Equal(t, &resources.ResourceData{
			IDKey:            "conn-remote-1",
			SourceIDKey:      "src-1",
			DestinationIDKey: "dst-2",
		}, result)
	})

	// An immutable field that is not an endpoint triggers no replacement and
	// fits in no PUT, so the import is refused whole rather than adopting a row
	// that would diff on every apply. TestUpdate covers the fields themselves.
	t.Run("an immutable difference fails the import without touching the row", func(t *testing.T) {
		t.Parallel()

		data := graphData(t, jsonMapperConfig())
		configMap(data)["cursor_column"] = "updated_at"

		mock := importClient(remoteRow("conn-remote-1", "", "src-1", "dst-1"))

		_, err := remoteHandler(mock, t).Import(t.Context(), localID, data, "conn-remote-1")

		assert.EqualError(t, err, `updating rETL connection during import: connection "users-to-webhook": connection update: cursor_column is immutable ("" -> "updated_at"); delete and recreate the connection to apply it`)
		assert.Empty(t, mock.UpdateCalls)
		assert.Empty(t, mock.CreateCalls)
		assert.Empty(t, mock.DeleteCalls)
		assert.Empty(t, mock.SetExternalIDCalls)
	})

	t.Run("a failed claim leaves the connection alone", func(t *testing.T) {
		t.Parallel()

		mock := importClient(remoteRow("conn-remote-1", "", "src-1", "dst-1"))
		mock.SetExternalIDFunc = func(_ *retlClient.SetRETLConnectionExternalIDRequest) error {
			return errors.New("gateway timeout")
		}

		_, err := remoteHandler(mock, t).Import(t.Context(), localID, graphData(t, jsonMapperConfig()), "conn-remote-1")

		assert.EqualError(t, err, "setting external ID for rETL connection during import: gateway timeout")
		assert.Empty(t, mock.DeleteCalls, "a pre-existing connection must never be deleted to undo a failed claim")
	})

	t.Run("refuses before mutating anything", func(t *testing.T) {
		t.Parallel()

		tests := map[string]struct {
			remote  retlClient.RETLConnection
			wantErr string
		}{
			"a destination-specific flow": {
				remote:  remoteRow("conn-remote-1", "", "src-1", "dst-specific"),
				wantErr: "uses a destination-specific rETL flow",
			},
			"a source that is not a rETL source": {
				remote:  remoteRow("conn-remote-1", "", "src-gone", "dst-1"),
				wantErr: `source "src-gone" is not a rETL source in this workspace`,
			},
			"a destination version the CLI does not know": {
				remote:  remoteRow("conn-remote-1", "", "src-1", "dst-old"),
				wantErr: "destination definition for apiType HTTP version 9 not found",
			},
			"a row already managed under another id": {
				remote:  remoteRow("conn-remote-1", "orders-to-webhook", "src-1", "dst-1"),
				wantErr: `connection "conn-remote-1" is already managed by the CLI as "orders-to-webhook"`,
			},
		}

		for name, test := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				mock := importClient(test.remote)

				_, err := remoteHandler(mock, t).Import(t.Context(), localID, graphData(t, jsonMapperConfig()), "conn-remote-1")

				assert.ErrorContains(t, err, test.wantErr)
				assert.Empty(t, mock.UpdateCalls)
				assert.Empty(t, mock.CreateCalls)
				assert.Empty(t, mock.DeleteCalls)
				assert.Empty(t, mock.SetExternalIDCalls)
			})
		}
	})

	// The syncer runs one Import per connection, so a catalog read per row is a
	// catalog read per connection in the project.
	t.Run("reads the endpoint catalogs once for the whole run", func(t *testing.T) {
		t.Parallel()

		mock := importClient(remoteRow("conn-remote-1", "", "src-1", "dst-1"))
		h := remoteHandler(mock, t)

		_, err := h.Import(t.Context(), localID, graphData(t, jsonMapperConfig()), "conn-remote-1")
		require.NoError(t, err)
		_, err = h.Import(t.Context(), "orders-to-webhook", graphData(t, jsonMapperConfig()), "conn-remote-2")
		require.NoError(t, err)

		assert.Equal(t, 1, mock.SourceListCalls)
		assert.Equal(t, 1, mock.DestinationsCalls)
	})

	t.Run("surfaces get errors", func(t *testing.T) {
		t.Parallel()

		mock := &MockConnectionClient{
			GetFunc: func(_ string) (*retlClient.RETLConnection, error) { return nil, errors.New("boom") },
		}

		_, err := remoteHandler(mock, t).Import(t.Context(), localID, graphData(t, jsonMapperConfig()), "conn-remote-1")

		assert.EqualError(t, err, "getting rETL connection during import: boom")
	})
}
