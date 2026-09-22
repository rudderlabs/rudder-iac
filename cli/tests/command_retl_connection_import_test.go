package tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
)

const (
	connectionImportModelExternalID       = "e2e-cimp-model"
	connectionImportDestinationExternalID = "e2e-cimp-http"
	connectionImportLocalID               = "e2e-cimp-connection"
)

// TestRETLConnectionImportClaim adopts a rETL connection created outside the
// CLI — the webapp's path — and checks that apply claims the existing row
// instead of creating a second one, which the destination would refuse anyway.
//
// A connection has no single-resource import command, and `import workspace`
// scaffolds every importable resource in the workspace, so it is not run
// against the shared one (see TestAccountsImportWorkspace). The test writes the
// spec that export would, import metadata included, which drives the same
// claim on apply. The spec's schedule and mapping differ from the seeded row on
// purpose, so the claim is followed by an update.
//
// The seed joins managed endpoints: an unmanaged connection on them would stop
// destroy from deleting them, so seeds on the fixture destination are removed
// before every destroy.
func TestRETLConnectionImportClaim(t *testing.T) {
	allowManagedResidue(t)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	var (
		fixtureDir  = filepath.Join("testdata", "retl_connection_import")
		credentials = filepath.Join(fixtureDir, "credentials.vars.yaml")
		baseDir     = filepath.Join(fixtureDir, "base")
		store       = retlClient.NewRudderRETLStore(newAccountsAPIClient(t))
	)

	removeSeededConnections(t, store)
	out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
	require.NoError(t, err, "destroy failed: %s", out)

	// Registered after the destroy so it runs before it.
	t.Cleanup(func() {
		out, err := executor.Execute(cliBinPath, "destroy", "--confirm=false")
		assert.NoError(t, err, "cleanup destroy failed: %s", out)
	})
	t.Cleanup(func() { removeSeededConnections(t, store) })

	applyRETLProject(t, executor, baseDir, credentials)

	var (
		model         = managedRETLSourceWithWorkspace(t, store, connectionImportModelExternalID)
		destinationID = managedDestinationID(t, connectionImportDestinationExternalID)
	)
	seeded, err := store.CreateConnection(context.Background(), &retlClient.CreateRETLConnectionRequest{
		SourceID:      model.ID,
		DestinationID: destinationID,
		Enabled:       lo.ToPtr(true),
		Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(60)},
		SyncBehaviour: lo.ToPtr(retlClient.SyncBehaviourUpsert),
		Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
		Mappings:      []retlClient.Mapping{{From: "email", To: "traits.email"}},
		Event:         &retlClient.Event{Type: retlClient.EventTypeIdentify},
	})
	require.NoError(t, err, "seeding an unmanaged RETL connection")
	require.NotEmpty(t, seeded.ID, "seeded RETL connection has no id")
	require.Empty(t, seeded.ExternalID, "a seed must be unmanaged")

	projectDir := t.TempDir()
	copyFixtureDir(t, baseDir, projectDir)
	writeConnectionClaimSpec(t, filepath.Join(projectDir, "connection.yaml"), model.WorkspaceID, seeded.ID)

	t.Run("apply claims the existing connection instead of creating one", func(t *testing.T) {
		applyRETLProject(t, executor, projectDir, credentials)

		id, claimed := managedRETLConnectionByExternalID(t, connectionImportLocalID)
		assert.Equal(t, seeded.ID, id, "the claim must adopt the seeded row, not create another")
		assert.Equal(t, retlClient.RETLConnection{
			SourceID:      model.ID,
			DestinationID: destinationID,
			Enabled:       true,
			ExternalID:    connectionImportLocalID,
			Schedule:      retlClient.Schedule{Type: retlClient.ScheduleTypeBasic, EveryMinutes: lo.ToPtr(30)},
			SyncSettings:  syncSettings(true, 30, 5, true),
			SyncBehaviour: retlClient.SyncBehaviourUpsert,
			Identifiers:   []retlClient.Mapping{{From: "id", To: "user_id"}},
			Mappings:      []retlClient.Mapping{{From: "email", To: "traits.emailAddress"}},
			Event:         &retlClient.Event{Type: retlClient.EventTypeIdentify},
		}, claimed)

		assert.NotContains(t, unmanagedConnectionIDs(t, store, destinationID), seeded.ID,
			"a claimed connection must leave the importable set")
	})

	t.Run("claimed connection converges", func(t *testing.T) {
		assertNoRETLConnectionChanges(t, executor, projectDir, credentials)
	})
}

// managedRETLSourceWithWorkspace is managedRETLSource without the workspace
// cleared: import metadata is filed under the workspace, and an entry for the
// wrong one would turn the claim into a silent create.
func managedRETLSourceWithWorkspace(t *testing.T, store retlClient.RETLStore, externalID string) retlClient.RETLSource {
	t.Helper()

	sources, err := store.ListRetlSources(context.Background(),
		retlClient.WithSourceType(string(retlClient.ModelSourceType)), retlClient.WithHasExternalId(lo.ToPtr(true)))
	require.NoError(t, err, "listing managed RETL sources")

	matches := lo.Filter(sources.Data, func(s retlClient.RETLSource, _ int) bool { return s.ExternalID == externalID })
	require.Len(t, matches, 1, "managed RETL sources claiming %q", externalID)
	require.NotEmpty(t, matches[0].WorkspaceID, "managed RETL source %q has no workspace id", externalID)
	return matches[0]
}

// removeSeededConnections deletes the unmanaged connections on the fixture
// destination, whichever run seeded them. A claimed connection is managed and
// left to destroy. Deletion errors are logged rather than failed, so cleanup
// always gets as far as it can.
func removeSeededConnections(t *testing.T, store retlClient.RETLStore) {
	t.Helper()

	destinations, err := store.GetDestinations(context.Background())
	require.NoError(t, err, "listing destinations")
	for _, destination := range destinations {
		if destination.ExternalID != connectionImportDestinationExternalID {
			continue
		}
		for _, id := range unmanagedConnectionIDs(t, store, destination.ID) {
			if err := store.DeleteConnection(context.Background(), id); err != nil {
				t.Logf("deleting seeded RETL connection %s: %v", id, err)
			}
		}
	}
}

func unmanagedConnectionIDs(t *testing.T, store retlClient.RETLStore, destinationID string) []string {
	t.Helper()

	var ids []string
	for page := 1; ; page++ {
		result, err := store.ListConnections(context.Background(), &retlClient.ListRETLConnectionsRequest{
			DestinationID: destinationID, HasExternalID: lo.ToPtr(false), Page: page, PageSize: 100,
		})
		require.NoError(t, err, "listing unmanaged RETL connections")
		ids = append(ids, lo.Map(result.Data, func(c retlClient.RETLConnection, _ int) string { return c.ID })...)
		if result.Paging.Next == "" {
			return ids
		}
	}
}

// writeConnectionClaimSpec writes the connection spec export would emit for the
// seeded row: the import metadata ties the local id to the remote one.
func writeConnectionClaimSpec(t *testing.T, path, workspaceID, remoteID string) {
	t.Helper()

	spec := &specs.Spec{
		Version: specs.SpecVersionV1,
		Kind:    "retl-connections",
		Metadata: map[string]any{
			"name": "retl-connections",
			"import": map[string]any{
				"workspaces": []any{map[string]any{
					"workspace_id": workspaceID,
					"resources": []any{map[string]any{
						"urn":       "retl-connection:" + connectionImportLocalID,
						"remote_id": remoteID,
					}},
				}},
			},
		},
		Spec: map[string]any{
			"connections": []any{map[string]any{
				"id":          connectionImportLocalID,
				"source":      "#retl-source-sql-model:" + connectionImportModelExternalID,
				"destination": "#destination:" + connectionImportDestinationExternalID,
				"enabled":     true,
				"config": map[string]any{
					"sync_behaviour": "upsert",
					"schedule":       map[string]any{"type": "basic", "every_minutes": 30},
					"identifiers":    []any{map[string]any{"from": "id", "to": "user_id"}},
					"mappings":       []any{map[string]any{"from": "email", "to": "traits.emailAddress"}},
					"event":          map[string]any{"type": "identify"},
				},
			}},
		},
	}

	raw, err := yaml.Marshal(spec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))
}

func copyFixtureDir(t *testing.T, from, to string) {
	t.Helper()

	entries, err := os.ReadDir(from)
	require.NoError(t, err, "reading %s", from)
	for _, entry := range entries {
		raw, err := os.ReadFile(filepath.Join(from, entry.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(to, entry.Name()), raw, 0o644))
	}
}
