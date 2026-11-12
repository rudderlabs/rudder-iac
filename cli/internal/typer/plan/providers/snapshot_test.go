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

func TestSnapshotRejectsUnknownFormatVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.snapshot.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"formatVersion": 99, "trackingPlan": {}}`), 0o644))

	_, err := providers.NewSnapshotPlanProvider(path).GetTrackingPlan(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "format version 99")
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
	})

	t.Run("snapshot", func(t *testing.T) {
		store := &mockTrackingPlanStore{trackingPlanWithSchemas: golden}
		path := filepath.Join(t.TempDir(), "plan.snapshot.json")

		_, err := providers.WriteSnapshot(ctx, store, "tp", 4, path)

		require.NoError(t, err)
		assert.Equal(t, 4, store.requestedVersion)
	})

	t.Run("latest by default", func(t *testing.T) {
		store := &mockTrackingPlanStore{trackingPlanWithSchemas: golden}

		_, err := providers.NewJSONSchemaPlanProvider("tp", store).GetTrackingPlan(ctx)

		require.NoError(t, err)
		assert.Zero(t, store.requestedVersion, "the versioned fetch must not run for the latest plan")
	})
}
