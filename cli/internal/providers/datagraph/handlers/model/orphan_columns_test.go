package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindOrphanedColumns guards the v1 "partial-merge has no delete path"
// warning contract: any column-metadata entry present in the remote-derived
// resource but absent from the local yaml-derived resource is orphaned and
// must be surfaced so the user knows the remote row will persist after apply.
func TestFindOrphanedColumns(t *testing.T) {
	tests := []struct {
		name   string
		local  []map[string]any
		remote []map[string]any
		want   []string
	}{
		{
			name: "no orphan when local includes all remote columns",
			local: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
				{"name": "email", "display_name": "Email"},
				{"name": "created_at", "display_name": "Created"},
			},
			remote: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
				{"name": "email", "display_name": "Email"},
			},
			want: nil,
		},
		{
			name: "no warning when remote is empty",
			local: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
			},
			remote: nil,
			want:   nil,
		},
		{
			name: "warn for orphan when local omits one remote column",
			local: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
				{"name": "email", "display_name": "Email"},
			},
			remote: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
				{"name": "email", "display_name": "Email"},
				{"name": "created_at", "display_name": "Created"},
			},
			want: []string{"created_at"},
		},
		{
			// When local is empty the user has removed every column-metadata
			// row from their yaml. v1's partial-merge contract leaves every
			// remote row in place — surface each as an orphan so the gap
			// is visible.
			name:  "warn for every remote column when local is empty",
			local: nil,
			remote: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
			},
			want: []string{"user_id"},
		},
		{
			name: "multiple orphans returned sorted and deduplicated by name",
			local: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
			},
			remote: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
				{"name": "created_at", "display_name": "Created"},
				{"name": "email", "display_name": "Email"},
				// Duplicate (name, display_name) pair from the remote should
				// still collapse to a single orphan warning per name.
				{"name": "email", "display_name": "Email Address"},
			},
			want: []string{"created_at", "email"},
		},
		{
			name:   "no warning when both local and remote are empty",
			local:  nil,
			remote: nil,
			want:   nil,
		},
		{
			// Entries with non-string or missing names are skipped — the
			// helper only attests to names it can compare reliably.
			name: "entries without a string name are ignored",
			local: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
			},
			remote: []map[string]any{
				{"name": "user_id", "display_name": "User ID"},
				{"display_name": "Stray"},
				{"name": 42, "display_name": "Numeric"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindOrphanedColumns(tt.local, tt.remote)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestFormatOrphanColumnWarning pins the LLD-specified wording verbatim.
// Any change here is a user-visible message change and should be discussed
// against the data-graph-column-metadata-clients-lld.md before being made.
func TestFormatOrphanColumnWarning(t *testing.T) {
	assert.Equal(
		t,
		"metadata for created_at will remain in the workspace; v1 has no clear/delete path",
		FormatOrphanColumnWarning("created_at"),
	)
}
