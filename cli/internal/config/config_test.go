package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// initConfigWithSyncerConcurrency loads a real config file rather than calling
// viper.Set: an explicit Set outranks the env binding and would leak into every
// later test in the package.
func initConfigWithSyncerConcurrency(t *testing.T, syncer int) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	body := fmt.Sprintf(`{"concurrency": {"syncer": %d}}`, syncer)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	InitConfig(path)
}

// Concurrent syncs went GA in DEX-731, so concurrency.syncer is now read on
// every apply/destroy. A sub-1 value was previously inert for anyone who had
// not enabled the flag — WithConcurrency was never applied and the syncer used
// its own default of 1 — so those configs must keep working rather than start
// failing the command outright.
//
// The subtests stay serial on purpose: InitConfig mutates the viper singleton,
// so t.Parallel here would race the parallel tests elsewhere in the package.
func TestGetConfig_ClampsSyncerConcurrency(t *testing.T) {
	cases := []struct {
		name      string
		configure int
		want      int
	}{
		{name: "zero clamps to one", configure: 0, want: 1},
		{name: "negative clamps to one", configure: -4, want: 1},
		{name: "valid value is preserved", configure: 12, want: 12},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			initConfigWithSyncerConcurrency(t, tc.configure)

			assert.Equal(t, tc.want, GetConfig().Concurrency.Syncer)
		})
	}
}

// Removing the flag took RUDDERSTACK_X_CONCURRENT_SYNCS with it, leaving no way
// to tune sync parallelism per run; concurrency.syncer had no env binding of its
// own. (concurrency.dataGraph is still unbound — out of scope here.)
func TestGetConfig_SyncerConcurrencyBindsEnv(t *testing.T) {
	t.Setenv("RUDDERSTACK_CLI_CONCURRENCY_SYNCER", "7")
	initConfigWithSyncerConcurrency(t, 3)

	assert.Equal(t, 7, GetConfig().Concurrency.Syncer)
}

func TestInitConfig_SourcesListenBindsEnv(t *testing.T) {
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
	t.Setenv("RUDDERSTACK_X_SOURCES_LISTEN", "true")

	InitConfig(filepath.Join(t.TempDir(), "config.json"))

	require.True(t, GetConfig().ExperimentalFlags.SourcesListen)
}

// initTelemetryConfig loads a config file with the given saved setting.
// A file, not viper.Set, so the setting does not leak into later tests.
func initTelemetryConfig(t *testing.T, savedDisabled bool) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	body := fmt.Sprintf(`{"telemetry": {"disabled": %t}}`, savedDisabled)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	InitConfig(path)
}

func TestGetConfig_DoNotTrack(t *testing.T) {
	cases := []struct {
		value string
		saved bool
		want  bool
	}{
		{"1", false, true},
		{"true", false, true},
		{"TRUE", false, true},
		{"yes", false, true},
		{"Yes", false, true},
		{"on", false, true},
		{"ON", false, true},
		// Off values change nothing, so a saved opt-out survives.
		{"0", true, true},
		{"false", true, true},
		{"no", true, true},
		{"off", true, true},
		{"", true, true},
		{"0", false, false},
		{"false", false, false},
		{"no", false, false},
		{"off", false, false},
		{"", false, false},
		// An unknown value never crashes and changes nothing.
		{"maybe", false, false},
		{"maybe", true, true},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%q saved=%t", tc.value, tc.saved), func(t *testing.T) {
			t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "")
			t.Setenv("DO_NOT_TRACK", tc.value)
			initTelemetryConfig(t, tc.saved)

			assert.Equal(t, tc.want, GetConfig().Telemetry.Disabled)
		})
	}
}

func TestGetConfig_ToolVariable(t *testing.T) {
	cases := []struct {
		name  string
		tool  string
		dnt   string
		saved bool
		want  bool
	}{
		{"true disables", "true", "", false, true},
		{"yes disables", "yes", "", false, true},
		{"false enables over a saved opt-out", "false", "", true, false},
		// Open design choice: the tool variable beats DO_NOT_TRACK.
		{"false beats DO_NOT_TRACK=1", "false", "1", false, false},
		{"unknown value is ignored", "maybe", "", false, false},
		{"unknown value leaves DO_NOT_TRACK in force", "maybe", "1", false, true},
		{"unknown value leaves a saved opt-out", "maybe", "", true, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", tc.tool)
			t.Setenv("DO_NOT_TRACK", tc.dnt)
			initTelemetryConfig(t, tc.saved)

			assert.Equal(t, tc.want, GetConfig().Telemetry.Disabled)
		})
	}
}
