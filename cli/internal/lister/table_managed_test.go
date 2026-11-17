package lister

import (
	"testing"

	"github.com/charmbracelet/bubbles/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The "Managed as" column is keyed on the data, so a resource type that does
// not carry adoption state keeps the three-column table it had.
func TestHasManaged(t *testing.T) {
	t.Parallel()

	assert.False(t, hasManaged(nil))
	assert.False(t, hasManaged([]resources.ResourceData{{"id": "src-1", "name": "Example"}}))
	assert.True(t, hasManaged([]resources.ResourceData{{"id": "acct-1", "managed": "-"}}))
	assert.True(t, hasManaged([]resources.ResourceData{
		{"id": "src-1", "name": "Example"},
		{"id": "acct-1", "managed": "google-ads-prod"},
	}), "one account carrying the key is enough to add the column")
}

// Without the key the table keeps its three columns and three-cell rows.
func TestTableDataWithoutManaged(t *testing.T) {
	t.Parallel()

	columns, rows := tableData([]resources.ResourceData{
		{"id": "src-1", "name": "Example"},
		{"id": "src-2"},
	}, nil)

	assert.Equal(t, []table.Column{
		{Title: "#", Width: 4},
		{Title: "ID", Width: 27},
		{Title: "Name", Width: 30},
	}, columns)
	assert.Equal(t, []table.Row{
		{"1", "src-1", "Example"},
		{"2", "src-2", "- not set -"},
	}, rows)
}

// With it, every row gains a cell — including the rows of another resource type
// listed alongside, which would otherwise be a cell short of the header.
func TestTableDataWithManaged(t *testing.T) {
	t.Parallel()

	columns, rows := tableData([]resources.ResourceData{
		{"id": "acct-1", "name": "Google Ads (prod)", "managed": "google-ads-prod"},
		{"id": "acct-2", "name": "Google Ads (dev)", "managed": "-"},
		{"id": "acct-3", "name": "Unlabelled", "managed": ""},
		{"id": "acct-4", "name": "No key at all"},
	}, map[string]int{"managed": 12})

	require.Len(t, columns, 4)
	assert.Equal(t, table.Column{Title: "Managed as", Width: 12}, columns[3])
	assert.Equal(t, []table.Row{
		{"1", "acct-1", "Google Ads (prod)", "google-ads-prod"},
		{"2", "acct-2", "Google Ads (dev)", "-"},
		{"3", "acct-3", "Unlabelled", "-"},
		{"4", "acct-4", "No key at all", "-"},
	}, rows)
}
