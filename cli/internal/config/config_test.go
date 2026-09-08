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
