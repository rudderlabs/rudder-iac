package dev

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten"
	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

// --jq reads the whole page, not the first 24000 bytes of it.
func TestJQSeesThePageBeyondTheDefaultCap(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	pad := strings.Repeat("p", 600)
	for range 60 {
		track(t, s, `{"event":"A","userId":"u1","properties":{"pad":"`+pad+`"}}`)
	}

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--json", "--jq", ".events | length")

	require.NoError(t, err)
	require.Equal(t, "60\n", stdout)
	require.NotContains(t, stderr, "truncated")
}

// --max-bytes applies to what jq prints: whole lines up to the cap, then
// output_limit with the command that lifts it.
func TestMaxBytesCapsTheJQOutput(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	for range 20 {
		track(t, s, `{"event":"Checkout Started","userId":"u1"}`)
	}

	stdout, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--max-bytes", "100", "--json",
		"--jq", ".events[].event")

	require.Error(t, err)
	require.Equal(t, strings.Repeat("Checkout Started\n", 5), stdout)
	require.True(t, strings.HasPrefix(stderr, "cursor=20 "), "the trailer comes first")
	obj := errorObject(t, lastLine(stderr))
	require.Equal(t, "output_limit", obj["code"])
	require.Equal(t, "rudder-cli dev events list --jq '.events[].event' --json --url "+s.URL()+" --max-bytes 0",
		obj["next"])
}

func TestTruncatedNoteUnderJQNamesTheCeiling(t *testing.T) {
	t.Parallel()
	note := pageNote(false, &devlisten.Truncated{Next: "rudder-cli dev events list --since 9 --json"}, true)
	require.Equal(t, "truncated: --jq read the page up to its 4194304-byte ceiling; next: "+
		"rudder-cli dev events list --since 9 --json", note)
}
