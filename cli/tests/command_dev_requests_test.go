package tests

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// The rejection hint of the summary is a curl line that runs and returns
// the refused request with its reason; its events link back by messageId.
func TestDevListenRequests(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	url := p.url()
	require.Equal(t, http.StatusOK, post(t, url+"/v1/batch",
		`{"batch":[{"type":"track","userId":"u1","event":"Order Completed","messageId":"m-1"}]}`))
	require.Equal(t, http.StatusBadRequest, post(t, url+"/v1/track", `{"event":"Order Completed"}`))

	summary := curl(t, url+"/_dev/v1/events?view=counts")
	next := gjson.Get(summary, `summary.diagnosis.#(code=="body_rejected").next`).Str
	require.Equal(t, "curl -fsS '"+url+"/_dev/v1/requests?since=0&failed=true&view=compact'", next)

	failed := curl(t, strings.TrimSuffix(strings.TrimPrefix(next, "curl -fsS '"), "'"))
	require.Equal(t, int64(1), gjson.Get(failed, "total").Int())
	require.Equal(t, "identity", gjson.Get(failed, "requests.0.rejection.stage").Str)

	record := curl(t, url+"/_dev/v1/requests?messageId=m-1&view=full")
	require.Equal(t, int64(1), gjson.Get(record, "requests.0.seq").Int())
	require.Equal(t, `{"type":"track","userId":"u1","event":"Order Completed","messageId":"m-1"}`,
		gjson.Get(record, "requests.0.events.0.message").Raw)
}

// dev --help prints the guide that the listener serves, so an agent reads
// the same text from either.
func TestDevListenHelpIsTheGuide(t *testing.T) {
	t.Parallel()
	p := startListen(t)
	cmd := exec.Command(cliBinPath, "dev", "--help")
	cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), "RUDDERSTACK_CLI_EXPERIMENTAL=true", "RUDDERSTACK_X_DEV_LISTEN=true")
	help, err := cmd.Output()
	require.NoError(t, err)

	guide := curl(t, p.url()+"/_dev/v1/guide")

	require.True(t, strings.HasPrefix(string(help), strings.TrimRight(guide, "\n")+"\n"))
	require.Greater(t, len(guide), 8000)
}
