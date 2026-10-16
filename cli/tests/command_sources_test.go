package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The dev group is unhidden and let run only after the root loads the config,
// so only the binary shows that each way of setting sourcesListen reaches it.
func TestSourcesListenGate(t *testing.T) {
	flagFile := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(flagFile, []byte(`{"experimental":true,"flags":{"sourcesListen":true}}`), 0o600))

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
			env:     map[string]string{"RUDDERSTACK_CLI_EXPERIMENTAL": "true", "RUDDERSTACK_X_SOURCES_LISTEN": "true"},
			enabled: true,
		},
		{
			name:    "experimental enable",
			env:     map[string]string{"RUDDERSTACK_CLI_EXPERIMENTAL": "true"},
			setup:   [][]string{{"experimental", "enable", "sourcesListen"}},
			enabled: true,
		},
		{name: "config flag", args: []string{"--config", flagFile}, enabled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
			t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", tc.env["RUDDERSTACK_CLI_EXPERIMENTAL"])
			t.Setenv("RUDDERSTACK_X_SOURCES_LISTEN", tc.env["RUDDERSTACK_X_SOURCES_LISTEN"])
			executor, err := NewCmdExecutor("")
			require.NoError(t, err)
			for _, args := range tc.setup {
				out, err := executor.Execute(cliBinPath, args...)
				require.NoError(t, err, "%s", out)
			}

			help, err := executor.Execute(cliBinPath, tc.args...)
			require.NoError(t, err, "%s", help)
			dev, devErr := executor.Execute(cliBinPath, append(tc.args, "sources")...)

			if !tc.enabled {
				require.NotContains(t, string(help), "\n  sources ")
				require.Error(t, devErr)
				require.Contains(t, string(dev), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
				require.Contains(t, string(dev), "RUDDERSTACK_X_SOURCES_LISTEN=true")
				require.Contains(t, string(dev), "Next: rudder-cli experimental enable sourcesListen")
				return
			}
			require.Contains(t, string(help), "\n  sources ")
			require.NoError(t, devErr, "%s", dev)
			require.Contains(t, string(dev), "Usage:")
		})
	}
}

// An agent or a CI job meets the gate first, so its JSON error has a code of
// its own.
func TestSourcesListenGateOffJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("RUDDERSTACK_CLI_TELEMETRY_DISABLED", "true")
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "")
	t.Setenv("RUDDERSTACK_X_SOURCES_LISTEN", "")
	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute(cliBinPath, "sources", "events", "--json")

	require.Error(t, err)
	var got struct {
		Error struct {
			Code string `json:"code"`
			Next string `json:"next"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(out, &got), "%s", out)
	require.Equal(t, "experimental_disabled", got.Error.Code)
	require.Equal(t, "rudder-cli experimental enable sourcesListen", got.Error.Next)
}
