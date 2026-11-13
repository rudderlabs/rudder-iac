package typer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestGenerateTracksRunFailuresAsErrored(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	for name, tc := range map[string]struct {
		args     []string
		platform string
		local    bool
	}{
		"unsupported platform": {
			args:     []string{"--platform", "cobol"},
			platform: "cobol",
		},
		"local plan fails to load": {
			args:     []string{"--platform", platformKotlin, "--local", "--location", missing, "--tracking-plan-id", "tp"},
			platform: platformKotlin,
			local:    true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			calls := telemetrytest.Record(t)

			require.Error(t, telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, tc.args))
			assert.Equal(t, []telemetrytest.Call{{
				Command: "typer generate",
				Errored: true,
				Extras: []telemetry.KV{
					{K: "platform", V: tc.platform},
					{K: "local", V: tc.local},
					{K: "snapshot", V: false},
				},
			}}, *calls)
		})
	}
}

func TestGenerateFromSnapshotNeedsNoCredentials(t *testing.T) {
	golden, err := os.ReadFile(filepath.Join("..", "..", "typer", "plan", "providers", "testdata", "remote_tracking_plan.golden.json"))
	require.NoError(t, err)

	snapshot, err := json.Marshal(map[string]any{"formatVersion": 1, "trackingPlan": json.RawMessage(golden)})
	require.NoError(t, err)

	var (
		dir          = t.TempDir()
		snapshotPath = filepath.Join(dir, "plan.snapshot.json")
		outDir       = filepath.Join(dir, "out")
	)
	require.NoError(t, os.WriteFile(snapshotPath, snapshot, 0o644))

	// Point config at an empty home so any attempt to read credentials fails
	t.Setenv("HOME", dir)
	t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
	telemetrytest.Record(t)

	require.NoError(t, telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, []string{
		"--platform", platformKotlin, "--snapshot", snapshotPath, "-o", outDir,
	}))

	assert.FileExists(t, filepath.Join(outDir, "Main.kt"))
}

func TestGenerateRejectsLocalWithSnapshot(t *testing.T) {
	telemetrytest.Record(t)

	err := telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, []string{
		"--platform", platformKotlin, "--local", "--snapshot", "plan.json",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestTrackingPlanVersionFlagValidation(t *testing.T) {
	for name, tc := range map[string]struct {
		args    []string
		wantErr string
	}{
		"negative version": {
			args:    []string{"--platform", platformKotlin, "--tracking-plan-id", "tp", "--tracking-plan-version", "-1"},
			wantErr: "must not be negative",
		},
		"version with local specs": {
			args:    []string{"--platform", platformKotlin, "--local", "--tracking-plan-version", "2"},
			wantErr: "only applies when generating from the remote workspace",
		},
		"version with a snapshot": {
			args:    []string{"--platform", platformKotlin, "--snapshot", "plan.json", "--tracking-plan-version", "2"},
			wantErr: "only applies when generating from the remote workspace",
		},
	} {
		t.Run(name, func(t *testing.T) {
			telemetrytest.Record(t)

			err := telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, tc.args)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	t.Run("snapshot rejects a negative version", func(t *testing.T) {
		telemetrytest.Record(t)

		err := telemetrytest.Execute(newCmdSnapshot(), []string{"typer"}, []string{"--tracking-plan-id", "tp", "--tracking-plan-version", "-2"})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be negative")
	})
}

// useRecordingCatalogAPI serves one tracking plan and records the URL of every
// catalog request, so a test can assert what the command asked the API for.
func useRecordingCatalogAPI(t *testing.T) func() []string {
	t.Helper()

	var (
		mu       sync.Mutex
		requests []string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.URL.RequestURI())
		mu.Unlock()

		switch {
		case r.URL.Path == "/v2/workspace":
			w.Write([]byte(`{"workspace":{"id":"ws-test"}}`))
		case r.URL.Path == "/v2/catalog/tracking-plans/tp-1/events":
			w.Write([]byte(`{"data": [], "total": 0, "currentPage": 1, "pageSize": 50}`))
		case r.URL.Query().Get("version") == "3":
			w.Write([]byte(`{"id": "tp-1", "name": "Pinned", "version": 3, "events": []}`))
		case r.URL.Path == "/v2/catalog/tracking-plans/tp-1":
			w.Write([]byte(`{"id": "tp-1", "name": "Latest", "version": 7}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	testutils.UseFakeAPI(t)
	testutils.SetConfig(t, "apiURL", srv.URL)

	return func() []string {
		mu.Lock()
		defer mu.Unlock()

		var catalog []string
		for _, r := range requests {
			if strings.HasPrefix(r, "/v2/catalog/") {
				catalog = append(catalog, r)
			}
		}
		return catalog
	}
}

func TestTrackingPlanVersionReachesTheAPI(t *testing.T) {
	t.Run("generate with a version", func(t *testing.T) {
		catalogRequests := useRecordingCatalogAPI(t)
		telemetrytest.Record(t)

		require.NoError(t, telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, []string{
			"--platform", platformKotlin, "--tracking-plan-id", "tp-1", "--tracking-plan-version", "3", "-o", t.TempDir(),
		}))

		assert.Equal(t, []string{"/v2/catalog/tracking-plans/tp-1?version=3&fetchSchemas=true"}, catalogRequests())
	})

	t.Run("generate without a version asks for the latest plan", func(t *testing.T) {
		catalogRequests := useRecordingCatalogAPI(t)
		telemetrytest.Record(t)

		require.NoError(t, telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, []string{
			"--platform", platformKotlin, "--tracking-plan-id", "tp-1", "-o", t.TempDir(),
		}))

		for _, request := range catalogRequests() {
			assert.NotContains(t, request, "version=", "the default path must not send a version")
			assert.NotContains(t, request, "fetchSchemas", "the default path must not use the pinned request")
		}
	})

	t.Run("snapshot with a version", func(t *testing.T) {
		catalogRequests := useRecordingCatalogAPI(t)
		telemetrytest.Record(t)
		out := filepath.Join(t.TempDir(), "plan.snapshot.json")

		require.NoError(t, telemetrytest.Execute(newCmdSnapshot(), []string{"typer"}, []string{
			"--tracking-plan-id", "tp-1", "--tracking-plan-version", "3", "-o", out,
		}))

		assert.Equal(t, []string{"/v2/catalog/tracking-plans/tp-1?version=3&fetchSchemas=true"}, catalogRequests())

		data, err := os.ReadFile(out)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"version": 3`)
	})
}

func TestGenerateRejectsTrackingPlanIDWithSnapshot(t *testing.T) {
	telemetrytest.Record(t)

	err := telemetrytest.Execute(newCmdGenerate(), []string{"typer"}, []string{
		"--platform", platformKotlin, "--snapshot", "plan.json", "--tracking-plan-id", "tp-other",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "--tracking-plan-id does not apply with --snapshot")
}
