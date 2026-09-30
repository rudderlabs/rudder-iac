package dev

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
)

// TestHelperSDK is the child process of the exec tests: it posts the events
// named in DEV_EXEC_EVENTS to RUDDERSTACK_DATA_PLANE_URL, then exits with
// DEV_EXEC_EXIT.
func TestHelperSDK(t *testing.T) {
	if os.Getenv("DEV_EXEC_HELPER") != "1" {
		t.Skip("child process of the exec tests")
	}
	url := os.Getenv("RUDDERSTACK_DATA_PLANE_URL")
	require.Equal(t, url, os.Getenv("RUDDERSTACK_DEV_URL"))
	for _, name := range strings.Split(os.Getenv("DEV_EXEC_EVENTS"), ",") {
		req, err := http.NewRequest(http.MethodPost, url+"/v1/track",
			strings.NewReader(`{"event":"`+name+`","userId":"u1"}`))
		require.NoError(t, err)
		req.SetBasicAuth("dev", "")
		resp, err := testHTTP.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
	}
	if os.Getenv("DEV_EXEC_EXIT") == "3" {
		os.Exit(3)
	}
}

func helperArgs() []string {
	return []string{"--", os.Args[0], "-test.run=^TestHelperSDK$"}
}

func TestExecRunsTheCommandAgainstAFreshListener(t *testing.T) {
	t.Setenv("DEV_EXEC_HELPER", "1")
	t.Setenv("DEV_EXEC_EVENTS", "A,B")

	stdout, _, err := runDev(t, append([]string{"exec", "--settle", "10ms", "--expect", "A", "--json"}, helperArgs()...)...)

	require.NoError(t, err)
	var summary struct {
		Requests struct{ Total int }       `json:"requests"`
		Expected []struct{ Status string } `json:"expected"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &summary), stdout)
	require.Equal(t, 2, summary.Requests.Total)
	require.Equal(t, "present", summary.Expected[0].Status)
}

func TestExecFailsWhenAnExpectedEventIsMissing(t *testing.T) {
	t.Setenv("DEV_EXEC_HELPER", "1")
	t.Setenv("DEV_EXEC_EVENTS", "A")

	stdout, stderr, err := runDev(t, append([]string{"exec", "--settle", "10ms", "--expect", "A", "--expect", "Z",
		"--json"}, helperArgs()...)...)

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	require.Equal(t, 1, silent.Code)
	require.Contains(t, stdout, `"expected_missing"`)
	obj := errorObject(t, lastLine(stderr))
	require.Equal(t, "expected_missing", obj["code"])
}

func TestExecReturnsTheChildExitCode(t *testing.T) {
	t.Setenv("DEV_EXEC_HELPER", "1")
	t.Setenv("DEV_EXEC_EVENTS", "A")
	t.Setenv("DEV_EXEC_EXIT", "3")

	stdout, stderr, err := runDev(t, append([]string{"exec", "--settle", "10ms", "--json"}, helperArgs()...)...)

	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	require.Equal(t, 3, silent.Code)
	require.Contains(t, stdout, `"total":1`)
	require.Equal(t, "command_failed", errorObject(t, lastLine(stderr))["code"])
}

func TestExecNeedsTheCommandAfterADash(t *testing.T) {
	t.Parallel()

	obj := usageObject(t, "exec", "--json", "node", "app.mjs")

	require.Equal(t, "rudder-cli dev exec --json -- node", obj["next"])
}
