package typer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/telemetry/telemetrytest"
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
			wantErr: "must be a positive number",
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
		assert.Contains(t, err.Error(), "must be a positive number")
	})
}
