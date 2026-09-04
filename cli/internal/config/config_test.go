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
// every apply/destroy. Configs carrying a sub-1 value were previously inert —
// WithConcurrency was never applied and the syncer used its own default of 1 —
// so they must keep working rather than fail the command outright.
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

// concurrency.syncer was the only concurrency key without an env binding, which
// left no way to tune sync parallelism per-run once RUDDERSTACK_X_CONCURRENT_SYNCS
// went away with the flag.
func TestGetConfig_SyncerConcurrencyBindsEnv(t *testing.T) {
	t.Setenv("RUDDERSTACK_CLI_CONCURRENCY_SYNCER", "7")
	initConfigWithSyncerConcurrency(t, 3)

	assert.Equal(t, 7, GetConfig().Concurrency.Syncer)
}
