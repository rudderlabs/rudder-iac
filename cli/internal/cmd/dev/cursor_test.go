package dev

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/pkg/exp/devlisten/devlistentest"
)

func TestCursorPrintsTheCursorAndServerID(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)

	stdout, _, err := runDev(t, "cursor", "--url", s.URL())
	require.NoError(t, err)
	require.Equal(t, `{"cursor":1,"serverId":"`+s.Ready().ServerID+`"}`+"\n", stdout)

	stdout, _, err = runDev(t, "cursor", "--url", s.URL(), "--jq", ".cursor")
	require.NoError(t, err)
	require.Equal(t, "1\n", stdout)
}

// A --jq projection drops the envelope, so stderr keeps the cursor.
func TestJQPrintsTheCursorTrailerOnStderr(t *testing.T) {
	t.Parallel()
	s := devlistentest.Start(t)
	track(t, s, `{"event":"A","userId":"u1"}`)
	sid := s.Ready().ServerID

	_, stderr, err := runDev(t, "events", "list", "--url", s.URL(), "--json", "--jq", ".events[].event")
	require.NoError(t, err)
	require.Equal(t, "cursor=1 serverId="+sid+" hasMore=false timedOut=false\n", stderr)

	_, stderr, err = runDev(t, "summary", "--url", s.URL(), "--json", "--jq", ".requests.total")
	require.NoError(t, err)
	require.Equal(t, "cursor=1 serverId="+sid+"\n", stderr)
}
