package demo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readRecords decodes every JSONL line the writer produced.
func readRecords(t *testing.T, path string) []Record {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var (
		dec  = json.NewDecoder(strings.NewReader(string(data)))
		recs []Record
	)
	for dec.More() {
		var r Record
		require.NoError(t, dec.Decode(&r))
		recs = append(recs, r)
	}
	return recs
}

func TestAppendWritesRecordWhenEnabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	start := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	Append(Record{
		Kind:     KindExec,
		Start:    start,
		End:      start.Add(2 * time.Second),
		Dir:      "/work",
		Argv:     []string{"rudder-cli", "apply", "-l", "project"},
		Env:      map[string]string{"RUDDERSTACK_X_RETL_TABLE_SUPPORT": "true"},
		ExitCode: 0,
		Output:   "applied",
	})

	assert.Equal(t, []Record{{
		Kind:     KindExec,
		Start:    start,
		End:      start.Add(2 * time.Second),
		Dir:      "/work",
		Argv:     []string{"rudder-cli", "apply", "-l", "project"},
		Env:      map[string]string{"RUDDERSTACK_X_RETL_TABLE_SUPPORT": "true"},
		ExitCode: 0,
		Output:   "applied",
	}}, readRecords(t, path))
}

func TestAppendIsNoOpWhenDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Append(Record{Kind: KindExec, Argv: []string{"rudder-cli", "apply"}})

	t.Setenv(EnvJournal, "")
	reset()
	assert.False(t, Enabled())
	Append(Record{Kind: KindExec, Argv: []string{"rudder-cli", "destroy"}})

	assert.Len(t, readRecords(t, path), 1, "the second Append must not have been written")
}

func TestAppendReopensWhenJournalPathChanges(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.jsonl")
	second := filepath.Join(dir, "second.jsonl")

	t.Setenv(EnvJournal, first)
	reset()
	Append(Record{Kind: KindExec, Argv: []string{"rudder-cli", "one"}})

	t.Setenv(EnvJournal, second)
	Append(Record{Kind: KindExec, Argv: []string{"rudder-cli", "two"}})

	assert.Len(t, readRecords(t, first), 1)
	require.Len(t, readRecords(t, second), 1)
	assert.Equal(t, []string{"rudder-cli", "two"}, readRecords(t, second)[0].Argv)
}

func TestAppendRedactsSecretEnvByPattern(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Append(Record{
		Kind: KindExec,
		Argv: []string{"rudder-cli", "apply"},
		Env: map[string]string{
			"RUDDERSTACK_ACCESS_TOKEN":            "pat_live_abc123",
			"RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY": "wk_xyz",
			"RUDDERSTACK_X_RETL_TABLE_SUPPORT":    "true",
			"RUDDERSTACK_API_URL":                 "http://localhost:15580",
		},
	})

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, map[string]string{
		"RUDDERSTACK_ACCESS_TOKEN":            Redacted,
		"RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY": Redacted,
		"RUDDERSTACK_X_RETL_TABLE_SUPPORT":    "true",
		"RUDDERSTACK_API_URL":                 "http://localhost:15580",
	}, recs[0].Env)
}

func TestAppendRedactsRegisteredLiteralsFromOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	// Stand-ins for the fixture credentials TestDestinationsApply registers.
	// Deliberately low-entropy so secret scanners do not flag this file.
	RedactLiterals("not-a-real-api-key", "not-a-real-access-key-id")
	Append(Record{
		Kind:   KindExec,
		Argv:   []string{"rudder-cli", "apply"},
		Output: "accessKeyId=not-a-real-access-key-id apiKey=not-a-real-api-key ok",
	})

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, "accessKeyId=<redacted> apiKey=<redacted> ok", recs[0].Output)
}

func TestAppendCapsOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Append(Record{Kind: KindExec, Argv: []string{"x"}, Output: strings.Repeat("a", MaxOutputBytes+500)})

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Len(t, recs[0].Output, MaxOutputBytes+len(truncationMarker))
	assert.True(t, strings.HasSuffix(recs[0].Output, truncationMarker))
}

func TestCollectEnvKeepsOnlyRudderstackPrefix(t *testing.T) {
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")
	t.Setenv("HOME", "/home/someone")
	t.Setenv("PATH", "/usr/bin")

	got := CollectEnv()

	assert.Equal(t, "true", got["RUDDERSTACK_X_RETL_TABLE_SUPPORT"])
	assert.NotContains(t, got, "HOME")
	assert.NotContains(t, got, "PATH")
}
