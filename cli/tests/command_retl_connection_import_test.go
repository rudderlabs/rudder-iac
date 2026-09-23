package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	retlClient "github.com/rudderlabs/rudder-iac/api/client/retl"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
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
	if os.Getenv("RUN_RETL_E2E") != "1" {
		t.Skip("set RUN_RETL_E2E=1; this suite applies to a live workspace")
	}
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
		model         = managedRETLSource(t, retlClient.ModelSourceType, connectionImportModelExternalID)
		destinationID = managedDestinationID(t, connectionImportDestinationExternalID)
		workspaceID   = projectWorkspaceID(t)
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
	// Anchors both the claim assertion below and removeSeededConnections, which
	// reads the same listing: were the HasExternalID filter to regress to an
	// empty result, the NotContains would pass vacuously and cleanup would
	// silently no-op, leaving unmanaged seeds that break every other suite.
	require.Contains(t, unmanagedConnectionIDs(t, store, destinationID), seeded.ID,
		"the seed must start in the importable set")

	projectDir := t.TempDir()
	copyFixtureDir(t, baseDir, projectDir)
	writeConnectionClaimSpec(t, filepath.Join(projectDir, "connection.yaml"), workspaceID, seeded.ID)

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

// projectWorkspaceID is the workspace the project resolves against, which is
// the one the import metadata has to be filed under: an entry for the wrong
// workspace turns the claim into a silent create.
func projectWorkspaceID(t *testing.T) string {
	t.Helper()

	workspace, err := newAccountsAPIClient(t).Workspaces.GetByAuthToken(context.Background())
	require.NoError(t, err, "reading the workspace")
	require.NotEmpty(t, workspace.ID, "workspace has no id")
	return workspace.ID
}

// removeSeededConnections deletes the unmanaged connections on the fixture
// destination, whichever run seeded them. A claimed connection is managed and
// left to destroy. Every deletion is attempted before any failure is reported,
// so cleanup gets as far as it can, but a failure is asserted rather than
// logged: this also runs as a precondition, and a seed left behind on a
// managed destination breaks the opening destroy of every other suite, far
// from the cause.
func removeSeededConnections(t *testing.T, store retlClient.RETLStore) {
	t.Helper()

	destinations, err := store.GetDestinations(context.Background())
	require.NoError(t, err, "listing destinations")

	var failures []error
	for _, destination := range destinations {
		if destination.ExternalID != connectionImportDestinationExternalID {
			continue
		}
		for _, id := range unmanagedConnectionIDs(t, store, destination.ID) {
			if err := store.DeleteConnection(context.Background(), id); err != nil {
				failures = append(failures, fmt.Errorf("deleting seeded RETL connection %s: %w", id, err))
			}
		}
	}
	assert.NoError(t, errors.Join(failures...), "seeded RETL connections left behind")
}

func unmanagedConnectionIDs(t *testing.T, store retlClient.RETLStore, destinationID string) []string {
	t.Helper()

	connections := listRETLConnections(t, store, &retlClient.ListRETLConnectionsRequest{
		DestinationID: destinationID, HasExternalID: lo.ToPtr(false),
	})
	return lo.Map(connections, func(c retlClient.RETLConnection, _ int) string { return c.ID })
}

// writeConnectionClaimSpec writes the connection spec export would emit for the
// seeded row: the import metadata ties the local id to the remote one. It goes
// through specs.ToImportSpec rather than hand-rolling the metadata map, so a
// yaml-tag change or a field rename cannot leave this green while real exports
// stop claiming.
func writeConnectionClaimSpec(t *testing.T, path, workspaceID, remoteID string) {
	t.Helper()

	spec, err := specs.ToImportSpec(
		"retl-connections",
		"retl-connections",
		specs.WorkspaceImportMetadata{
			WorkspaceID: workspaceID,
			Resources: []specs.ImportIds{{
				URN:      connection.ResourceType + ":" + connectionImportLocalID,
				RemoteID: remoteID,
			}},
		},
		map[string]any{
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
	)
	require.NoError(t, err, "building the claim spec")

	raw, err := yaml.Marshal(spec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o644))
}

func copyFixtureDir(t *testing.T, from, to string) {
	t.Helper()

	require.NoError(t, os.CopyFS(to, os.DirFS(from)), "copying %s", from)
}
