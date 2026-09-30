package dev

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/cmderrors"
)

// usageObject runs a command that must fail on its flags and returns the
// error object. Flag errors never need a server.
func usageObject(t *testing.T, args ...string) map[string]any {
	t.Helper()
	stdout, stderr, err := runDev(t, args...)
	var silent *cmderrors.SilentError
	require.ErrorAs(t, err, &silent)
	require.Empty(t, stdout)
	obj := errorObject(t, stderr)
	require.Equal(t, "usage", obj["code"], stderr)
	return obj
}

func TestUnknownFlagOnASiblingNamesTheSibling(t *testing.T) {
	t.Parallel()

	obj := usageObject(t, "requests", "list", "--json", "--since", "0", "--event", "A")

	require.Equal(t, "rudder-cli dev events --since 0 --event VALUE --json", obj["next"])
	require.Equal(t, "event", obj["param"])
}

func TestMistypedFlagNamesTheCloseOne(t *testing.T) {
	t.Parallel()

	obj := usageObject(t, "events", "list", "--json", "--events", "A")

	require.Equal(t, "rudder-cli dev events list --event VALUE --json", obj["next"])
	require.Contains(t, obj["message"], "did you mean --event?")
	require.Contains(t, obj["details"].(map[string]any)["validFlags"], "--event")
}

func TestBadFlagValueNamesTheFlag(t *testing.T) {
	t.Parallel()

	obj := usageObject(t, "events", "list", "--json", "--view", "compact", "--since", "abc")

	require.Equal(t, "since", obj["param"])
	require.Equal(t, "rudder-cli dev events list --view compact --since N --json", obj["next"])
}

func TestOtherParseErrorsNameTheFirstExample(t *testing.T) {
	t.Parallel()
	cmd := findCommand(t, []string{"events", "list"})

	err := usageFor(cmd, errBoom)

	require.Equal(t, &cliError{Code: "usage", Message: "boom", Next: firstExample(cmd)}, err)
	require.Contains(t, err.Next, "rudder-cli dev events list")
}

var errBoom = errors.New("boom")
