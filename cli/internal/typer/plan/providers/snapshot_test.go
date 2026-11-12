package providers_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/typer/plan/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadGolden(t *testing.T) *catalog.TrackingPlanWithSchemas {
	t.Helper()

	data, err := os.ReadFile("testdata/remote_tracking_plan.golden.json")
	require.NoError(t, err)

	var golden catalog.TrackingPlanWithSchemas
	require.NoError(t, json.Unmarshal(data, &golden))
	return &golden
}

func TestSnapshotRoundTripMatchesRemotePlan(t *testing.T) {
	var (
		ctx    = context.Background()
		golden = loadGolden(t)
		store  = &mockTrackingPlanStore{trackingPlanWithSchemas: golden}
		path   = filepath.Join(t.TempDir(), "plan.snapshot.json")
	)

	_, err := providers.WriteSnapshot(ctx, store, "tp", 0, path)
	require.NoError(t, err)

	want, err := providers.NewJSONSchemaPlanProvider("tp", store).GetTrackingPlan(ctx)
	require.NoError(t, err)

	got, err := providers.NewSnapshotPlanProvider(path).GetTrackingPlan(ctx)
	require.NoError(t, err)

	assert.Equal(t, want, got)
}

func TestSnapshotRejectsFilesItCannotRead(t *testing.T) {
	for name, tc := range map[string]struct {
		content string
		wantErr string
	}{
		"newer format version": {
			content: `{"formatVersion": 2, "trackingPlan": {"events": []}}`,
			wantErr: "upgrade rudder-cli",
		},
		"no format version, such as a raw API response": {
			content: `{"id": "tp-1", "events": []}`,
			wantErr: "not a snapshot this CLI can read",
		},
		"no tracking plan": {
			content: `{"formatVersion": 1}`,
			wantErr: "has no tracking plan",
		},
		"null tracking plan": {
			content: `{"formatVersion": 1, "trackingPlan": null}`,
			wantErr: "has no tracking plan",
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "plan.snapshot.json")
			require.NoError(t, os.WriteFile(path, []byte(tc.content), 0o644))

			_, err := providers.NewSnapshotPlanProvider(path).GetTrackingPlan(context.Background())

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestSnapshotAcceptsAPlanWithNoEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.snapshot.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"formatVersion": 1, "trackingPlan": {"id": "tp-1", "name": "Empty", "events": []}}`), 0o644))

	got, err := providers.NewSnapshotPlanProvider(path).GetTrackingPlan(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "Empty", got.Name)
	assert.Empty(t, got.Rules)
}

func TestWriteSnapshotReplacesTheFileAtomically(t *testing.T) {
	var (
		ctx   = context.Background()
		store = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
		dir   = t.TempDir()
		path  = filepath.Join(dir, "plan.snapshot.json")
	)
	require.NoError(t, os.WriteFile(path, []byte("previous"), 0o600))

	_, err := providers.WriteSnapshot(ctx, store, "tp", 0, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"formatVersion": 1`)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "the temporary file must not be left behind")

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestWriteSnapshotFailureKeepsNoTemporaryFile(t *testing.T) {
	var (
		store = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
		dir   = t.TempDir()
		// A directory cannot be replaced by a file, so the final rename fails
		path = filepath.Join(dir, "target")
	)
	require.NoError(t, os.Mkdir(path, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "keep"), []byte("x"), 0o644))

	_, err := providers.WriteSnapshot(context.Background(), store, "tp", 0, path)
	require.Error(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "target", entries[0].Name())
}

func TestSnapshotMissingFile(t *testing.T) {
	_, err := providers.NewSnapshotPlanProvider(filepath.Join(t.TempDir(), "nope.json")).GetTrackingPlan(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading snapshot")
}

func TestPinnedVersionUsesVersionedFetch(t *testing.T) {
	var (
		ctx    = context.Background()
		golden = loadGolden(t)
	)

	t.Run("provider", func(t *testing.T) {
		store := &mockTrackingPlanStore{trackingPlanWithSchemas: golden}

		_, err := providers.NewJSONSchemaPlanProvider("tp", store).WithVersion(4).GetTrackingPlan(ctx)

		require.NoError(t, err)
		assert.Equal(t, 4, store.requestedVersion)
		assert.Equal(t, 1, store.versionedCalls)
		assert.Zero(t, store.latestCalls)
	})

	t.Run("snapshot", func(t *testing.T) {
		store := &mockTrackingPlanStore{trackingPlanWithSchemas: golden}
		path := filepath.Join(t.TempDir(), "plan.snapshot.json")

		_, err := providers.WriteSnapshot(ctx, store, "tp", 4, path)

		require.NoError(t, err)
		assert.Equal(t, 4, store.requestedVersion)
		assert.Equal(t, 1, store.versionedCalls)
		assert.Zero(t, store.latestCalls)
	})

	t.Run("latest by default", func(t *testing.T) {
		store := &mockTrackingPlanStore{trackingPlanWithSchemas: golden}

		_, err := providers.NewJSONSchemaPlanProvider("tp", store).GetTrackingPlan(ctx)

		require.NoError(t, err)
		assert.Equal(t, 1, store.latestCalls)
		assert.Zero(t, store.versionedCalls, "the versioned fetch must not run for the latest plan")
	})
}
