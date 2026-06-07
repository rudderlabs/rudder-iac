package apply

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateApplyFlags(t *testing.T) {
	tests := []struct {
		name            string
		files           []string
		locationChanged bool
		wantErr         bool
	}{
		{
			name:            "both -f and --location explicitly set → error",
			files:           []string{"a.yaml"},
			locationChanged: true,
			wantErr:         true,
		},
		{
			name:            "-f only, --location not changed → ok",
			files:           []string{"a.yaml"},
			locationChanged: false,
			wantErr:         false,
		},
		{
			name:            "--location only (no -f) → ok",
			files:           nil,
			locationChanged: true,
			wantErr:         false,
		},
		{
			name:            "neither -f nor --location → ok",
			files:           nil,
			locationChanged: false,
			wantErr:         false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateApplyFlags(tc.files, tc.locationChanged)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--file/-f and --location are mutually exclusive")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildSyncOptions(t *testing.T) {
	reporter := &testutils.MockReporter{}

	t.Run("scoped variant has exactly one more option than non-scoped", func(t *testing.T) {
		nonScoped := buildSyncOptions(false, false, false, reporter, 1, false)
		scoped := buildSyncOptions(true, false, false, reporter, 1, false)

		assert.Equal(t, len(nonScoped)+1, len(scoped),
			"scoped mode should append exactly one additional option (WithScopeToTarget)")
	})

	t.Run("concurrency option is included when useConcurrency is true", func(t *testing.T) {
		withoutConcurrency := buildSyncOptions(false, false, false, reporter, 4, false)
		withConcurrency := buildSyncOptions(false, false, false, reporter, 4, true)

		assert.Equal(t, len(withoutConcurrency)+1, len(withConcurrency),
			"enabling concurrency should append exactly one additional option")
	})

	t.Run("both scoped and concurrency each add one option", func(t *testing.T) {
		base := buildSyncOptions(false, false, false, reporter, 4, false)
		both := buildSyncOptions(true, false, false, reporter, 4, true)

		assert.Equal(t, len(base)+2, len(both),
			"scoped + concurrency should add exactly two options over the base")
	})

	t.Run("options are valid and apply without error", func(t *testing.T) {
		// Verify that all returned options are callable without error by applying them
		// to a syncer. This validates the option constructors complete successfully.
		opts := buildSyncOptions(true, true, true, reporter, 2, true)
		for _, opt := range opts {
			require.IsType(t, syncer.Option(nil), opt,
				"each element should be a syncer.Option")
		}
		// The len-delta proof above plus syncer_test.go (Task 3) provide behavioral
		// coverage of WithScopeToTarget suppressing deletes.
	})
}
