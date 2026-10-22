package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/tests/demo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readJournal(t *testing.T, path string) []demo.Record {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var recs []demo.Record
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for dec.More() {
		var r demo.Record
		require.NoError(t, dec.Decode(&r))
		recs = append(recs, r)
	}

	return recs
}

func TestExecuteJournalsCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(demo.EnvJournal, path)
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute("echo", "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))

	recs := readJournal(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, demo.KindExec, recs[0].Kind)
	assert.Equal(t, []string{"echo", "hello"}, recs[0].Argv)
	assert.Equal(t, 0, recs[0].ExitCode)
	assert.Equal(t, "hello\n", recs[0].Output)
	assert.Equal(t, "true", recs[0].Env["RUDDERSTACK_X_RETL_TABLE_SUPPORT"])
	assert.False(t, recs[0].Start.IsZero())
	assert.False(t, recs[0].End.Before(recs[0].Start))
}

func TestExecuteRecordsNonZeroExit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(demo.EnvJournal, path)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	_, err = executor.Execute("sh", "-c", "exit 7")
	require.Error(t, err)

	recs := readJournal(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, 7, recs[0].ExitCode)
}

func TestExecuteRecordsStartFailureAsUnknownExitCode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(demo.EnvJournal, path)

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	_, err = executor.Execute("definitely-not-a-real-binary-xyz")
	require.Error(t, err)

	recs := readJournal(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, -1, recs[0].ExitCode, "a command that never started must not be journaled as exit 0")
}

// TestExecuteIsTransparentWhenJournalDisabled pins the framework's
// load-bearing property: an ordinary `make test-e2e` must be unaffected by the
// journal hook's existence — same bytes, same error, on both paths.
//
// It deliberately does not try to prove the hook's own Enabled() guard ran.
// "Disabled" is an empty RUDDER_DEMO_JOURNAL, and demo.Append carries the same
// guard, so removing the hook's guard changes nothing observable. If Append's
// guard is ever removed, this coupling breaks and the hook's guard becomes
// load-bearing — see the comment on journal().
func TestExecuteIsTransparentWhenJournalDisabled(t *testing.T) {
	t.Setenv(demo.EnvJournal, "")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	t.Run("success path", func(t *testing.T) {
		out, err := executor.Execute("echo", "hello")
		require.NoError(t, err)
		assert.Equal(t, "hello\n", string(out))
	})

	t.Run("failure path preserves both output and error", func(t *testing.T) {
		out, err := executor.Execute("sh", "-c", "echo boom >&2; exit 7")
		require.Error(t, err)

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		assert.Equal(t, 7, exitErr.ExitCode())
		assert.Equal(t, "boom\n", string(out), "combined output must reach the caller untouched")
	})
}
