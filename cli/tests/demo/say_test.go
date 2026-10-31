package demo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSayRecordsNarration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journal.jsonl")
	t.Setenv(EnvJournal, path)
	reset()

	Say(t, "The typed rule catches bad table specs")

	recs := readRecords(t, path)
	require.Len(t, recs, 1)
	assert.Equal(t, KindSay, recs[0].Kind)
	assert.Equal(t, "The typed rule catches bad table specs", recs[0].Text)
	assert.False(t, recs[0].Start.IsZero(), "narration needs a timestamp to be placed in the stream")
}

func TestSayIsNoOpWhenDisabled(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvJournal, "")
	reset()

	Say(t, "no journal configured")

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
