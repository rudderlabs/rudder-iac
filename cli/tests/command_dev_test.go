package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The dev group is unhidden and let run only after the root loads the config,
// so only the binary shows that each way of setting devListen reaches it.
func TestDevListenGate(t *testing.T) {
	flagFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(flagFile, []byte(`{"experimental":true,"flags":{"devListen":true}}`), 0o600))

	for _, tc := range []struct {
		name    string
		env     map[string]string
		setup   [][]string
		args    []string
		enabled bool
	}{
		{name: "off", enabled: false},
		{name: "umbrella only", env: map[string]string{"RUDDERSTACK_CLI_EXPERIMENTAL": "true"}, enabled: false},
		{
			name:    "environment",
			env:     map[string]string{"RUDDERSTACK_CLI_EXPERIMENTAL": "true", "RUDDERSTACK_X_DEV_LISTEN": "true"},
			enabled: true,
		},
		{
			name:    "experimental enable",
			env:     map[string]string{"RUDDERSTACK_CLI_EXPERIMENTAL": "true"},
			setup:   [][]string{{"experimental", "enable", "devListen"}},
			enabled: true,
		},
		{name: "config flag", args: []string{"--config", flagFile}, enabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
			t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", tc.env["RUDDERSTACK_CLI_EXPERIMENTAL"])
			t.Setenv("RUDDERSTACK_X_DEV_LISTEN", tc.env["RUDDERSTACK_X_DEV_LISTEN"])
			executor, err := NewCmdExecutor("")
			require.NoError(t, err)
			for _, args := range tc.setup {
				out, err := executor.Execute(cliBinPath, args...)
				require.NoError(t, err, "%s", out)
			}

			help, err := executor.Execute(cliBinPath, tc.args...)
			require.NoError(t, err, "%s", help)
			dev, devErr := executor.Execute(cliBinPath, append(tc.args, "dev")...)

			if !tc.enabled {
				require.NotContains(t, string(help), "\n  dev ")
				require.Error(t, devErr)
				require.Contains(t, string(dev), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
				require.Contains(t, string(dev), "RUDDERSTACK_X_DEV_LISTEN=true")
				require.Contains(t, string(dev), "Next: rudder-cli experimental enable devListen")
				return
			}
			require.Contains(t, string(help), "\n  dev ")
			require.NoError(t, devErr, "%s", dev)
			require.Contains(t, string(dev), "Usage:")
		})
	}
}
