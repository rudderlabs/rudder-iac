package tests

import (
	"encoding/json"
	"os"
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

func TestExecuteWritesNothingWhenJournalDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(demo.EnvJournal, "")

	executor, err := NewCmdExecutor("")
	require.NoError(t, err)

	out, err := executor.Execute("echo", "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(out), "output must be identical whether or not journaling is on")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
