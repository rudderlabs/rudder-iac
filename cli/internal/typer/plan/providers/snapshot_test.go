package providers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
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
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "an existing file keeps its mode, the snapshot holds workspace and user ids")
}

func TestWriteSnapshotNewFileGetsTheDefaultMode(t *testing.T) {
	var (
		store = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
		dir   = t.TempDir()
		path  = filepath.Join(dir, "plan.snapshot.json")
	)

	// A file created the plain way shows what the process umask allows
	reference := filepath.Join(dir, "reference")
	f, err := os.OpenFile(reference, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	want, err := os.Stat(reference)
	require.NoError(t, err)

	_, err = providers.WriteSnapshot(context.Background(), store, "tp", 0, path)
	require.NoError(t, err)

	got, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, want.Mode().Perm(), got.Mode().Perm())
}

func TestWriteSnapshotThroughASymlinkReplacesTheTarget(t *testing.T) {
	var (
		store  = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
		dir    = t.TempDir()
		target = filepath.Join(dir, "real.json")
		link   = filepath.Join(dir, "plan.snapshot.json")
	)
	require.NoError(t, os.WriteFile(target, []byte("old"), 0o600))
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}

	_, err := providers.WriteSnapshot(context.Background(), store, "tp", 0, link)
	require.NoError(t, err)

	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "the link must stay a link")

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"formatVersion": 1`)
}

func TestWriteSnapshotThroughADanglingSymlinkCreatesTheTarget(t *testing.T) {
	var (
		store  = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
		dir    = t.TempDir()
		target = filepath.Join(dir, "created.json")
		link   = filepath.Join(dir, "plan.snapshot.json")
	)
	if err := os.Symlink("created.json", link); err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}

	_, err := providers.WriteSnapshot(context.Background(), store, "tp", 0, link)
	require.NoError(t, err)

	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&os.ModeSymlink, "the link must stay a link")

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"formatVersion": 1`)
}

func TestWriteSnapshotConcurrentWritersToANewPath(t *testing.T) {
	const (
		writers = 4
		rounds  = 30
	)
	golden := loadGolden(t)

	for round := range rounds {
		var (
			path    = filepath.Join(t.TempDir(), fmt.Sprintf("plan-%d.snapshot.json", round))
			started sync.WaitGroup
			done    atomic.Bool
			empty   atomic.Int32
			errs    = make(chan error, writers)
		)

		// A reader must never see the file empty, or fail to create it
		reader := make(chan struct{})
		go func() {
			defer close(reader)
			for !done.Load() {
				if data, err := os.ReadFile(path); err == nil && len(data) == 0 {
					empty.Add(1)
				}
			}
		}()

		started.Add(writers)
		var finished sync.WaitGroup
		finished.Add(writers)
		for range writers {
			go func() {
				defer finished.Done()
				// Each writer has its own store, because the mock is not safe to share
				store := &mockTrackingPlanStore{trackingPlanWithSchemas: golden}
				started.Done()
				started.Wait()
				_, err := providers.WriteSnapshot(context.Background(), store, "tp", 0, path)
				errs <- err
			}()
		}
		finished.Wait()
		done.Store(true)
		<-reader
		close(errs)

		for err := range errs {
			require.NoError(t, err, "round %d", round)
		}
		assert.Zero(t, empty.Load(), "round %d: a reader saw an empty file", round)
	}
}

func TestWriteSnapshotUpdatesAWritableFileInAReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions work differently on Windows")
	}

	var (
		store = &mockTrackingPlanStore{trackingPlanWithSchemas: loadGolden(t)}
		dir   = t.TempDir()
		path  = filepath.Join(dir, "plan.snapshot.json")
	)
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	_, err := providers.WriteSnapshot(context.Background(), store, "tp", 0, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"formatVersion": 1`)
}

func TestWriteSnapshotIsByteStableWhateverTheEventOrder(t *testing.T) {
	var (
		ctx     = context.Background()
		golden  = loadGolden(t)
		dir     = t.TempDir()
		first   = filepath.Join(dir, "first.json")
		second  = filepath.Join(dir, "second.json")
		shuffle = *golden
	)
	require.Greater(t, len(golden.Events), 1)

	// The API returns events in the order its pages arrive, which varies
	shuffle.Events = slices.Clone(golden.Events)
	slices.Reverse(shuffle.Events)

	_, err := providers.WriteSnapshot(ctx, &mockTrackingPlanStore{trackingPlanWithSchemas: golden}, "tp", 0, first)
	require.NoError(t, err)
	_, err = providers.WriteSnapshot(ctx, &mockTrackingPlanStore{trackingPlanWithSchemas: &shuffle}, "tp", 0, second)
	require.NoError(t, err)

	a, err := os.ReadFile(first)
	require.NoError(t, err)
	b, err := os.ReadFile(second)
	require.NoError(t, err)
	assert.Equal(t, string(a), string(b))
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
