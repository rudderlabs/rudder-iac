package importreport_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importreport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var lost = []importreport.Lost{
	{Path: "consentManagement.amp", Value: "(object, 2 keys)"},
	{Path: "legacy", Value: "(value)"},
}

func TestReport_Uncovered(t *testing.T) {
	t.Run("no entry covers nothing", func(t *testing.T) {
		assert.Equal(t, lost, (&importreport.Report{}).Uncovered("d1", lost))
		assert.Equal(t, lost, (*importreport.Report)(nil).Uncovered("d1", lost))
	})

	t.Run("an unacknowledged entry covers nothing", func(t *testing.T) {
		r := &importreport.Report{Entries: []importreport.Entry{{RemoteID: "d1", Lost: lost}}}
		assert.Equal(t, lost, r.Uncovered("d1", lost))
	})

	t.Run("an acknowledged entry covers its own paths", func(t *testing.T) {
		r := &importreport.Report{Entries: []importreport.Entry{{RemoteID: "d1", Lost: lost, Acknowledged: true}}}
		assert.Empty(t, r.Uncovered("d1", lost))
	})

	t.Run("an acknowledged entry does not cover a path that appeared later", func(t *testing.T) {
		r := &importreport.Report{Entries: []importreport.Entry{{RemoteID: "d1", Lost: lost[:1], Acknowledged: true}}}
		assert.Equal(t, lost[1:], r.Uncovered("d1", lost))
	})

	t.Run("an entry for another destination covers nothing", func(t *testing.T) {
		r := &importreport.Report{Entries: []importreport.Entry{{RemoteID: "d2", Lost: lost, Acknowledged: true}}}
		assert.Equal(t, lost, r.Uncovered("d1", lost))
	})
}

func TestWriteLoad_KeepsAReviewedEntryAcrossImports(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, importreport.Write(dir, []importreport.Entry{
		{URN: "destination:a", RemoteID: "d1", Lost: lost},
	}))

	// A person acknowledges d1.
	loaded, err := importreport.Load(dir)
	require.NoError(t, err)
	loaded.Entries[0].Acknowledged = true
	require.NoError(t, os.Remove(filepath.Join(dir, importreport.FileName)))
	require.NoError(t, importreport.Write(dir, loaded.Entries))

	// A second import reports only d2.
	require.NoError(t, importreport.Write(dir, []importreport.Entry{
		{URN: "destination:b", RemoteID: "d2", Lost: lost},
	}))

	got, err := importreport.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, &importreport.Report{Entries: []importreport.Entry{
		{URN: "destination:a", RemoteID: "d1", Lost: lost, Acknowledged: true},
		{URN: "destination:b", RemoteID: "d2", Lost: lost},
	}}, got)
}

func TestLoad_FindsReportsInSubdirectoriesAndNoneIsEmpty(t *testing.T) {
	root := t.TempDir()

	got, err := importreport.Load(root)
	require.NoError(t, err)
	assert.Empty(t, got.Entries)

	require.NoError(t, importreport.Write(filepath.Join(root, "imported"), []importreport.Entry{
		{URN: "destination:a", RemoteID: "d1", Lost: lost},
	}))
	got, err = importreport.Load(root)
	require.NoError(t, err)
	assert.Len(t, got.Entries, 1)
}
