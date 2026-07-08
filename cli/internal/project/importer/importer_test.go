package importer

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
)

func TestCheckSyncStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		diff    *differ.Diff
		merge   bool
		wantErr bool
	}{
		{
			name:  "no merge, synced project",
			diff:  &differ.Diff{},
			merge: false,
		},
		{
			name:    "no merge, pending changes block import",
			diff:    &differ.Diff{NewResources: []string{"category:checkout"}},
			merge:   false,
			wantErr: true,
		},
		{
			name:  "merge allows pending additions",
			diff:  &differ.Diff{NewResources: []string{"category:checkout"}},
			merge: true,
		},
		{
			name:    "merge still blocks pending deletions",
			diff:    &differ.Diff{RemovedResources: []string{"category:legacy"}},
			merge:   true,
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := checkSyncStatus(tc.diff, tc.merge)

			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, ErrProjectNotSynced)
		})
	}
}
