package dev

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

func TestProbeBrowserPrintsThePageAndTheCheck(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)

	stdout, _, err := runDev(t, "probe", "--browser", "--url", s.URL(), "--json")

	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got), stdout)
	require.Equal(t, s.URL()+"/_dev/v1/probe.html", got["url"])
	require.Equal(t, "rudder-cli dev summary --since 0 --expect 'dev probe=1' --json", got["next"])
	require.Contains(t, got["message"], "Open url in the app's browser")
}

func TestProbeWithoutBrowserNamesTheFlag(t *testing.T) {
	t.Parallel()

	obj := usageObject(t, "probe", "--json")

	require.Equal(t, "rudder-cli dev probe --browser --json", obj["next"])
}
