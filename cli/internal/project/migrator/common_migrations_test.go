package migrator

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionUpdateMigration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		inputVersion    string
		expectedVersion string
		shouldChange    bool
	}{
		{
			name:            "updates rudder/0.1 to rudder/v1",
			inputVersion:    specs.SpecVersionV0_1,
			expectedVersion: specs.SpecVersionV1,
			shouldChange:    true,
		},
		{
			name:            "leaves rudder/v1 unchanged",
			inputVersion:    specs.SpecVersionV1,
			expectedVersion: specs.SpecVersionV1,
			shouldChange:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := &specs.Spec{
				Version:  tc.inputVersion,
				Kind:     "test",
				Metadata: map[string]any{"name": "test"},
				Spec:     map[string]any{"id": "test"},
			}

			migration := VersionUpdateMigration{}
			err := migration.Apply(spec)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedVersion, spec.Version)
		})
	}
}

func TestApplyCommonMigrations(t *testing.T) {
	t.Parallel()

	t.Run("applies all migrations in order", func(t *testing.T) {
		t.Parallel()

		migrator := Migrator{
			commonMigrations: CommonMigrations{
				VersionUpdateMigration{},
			},
		}

		spec := &specs.Spec{
			Version:  specs.SpecVersionV0_1Variant, // rudder/v0.1
			Kind:     "test",
			Metadata: map[string]any{"name": "test"},
			Spec:     map[string]any{"id": "test"},
		}

		err := migrator.applyCommonMigrations(spec)

		require.NoError(t, err)
		assert.Equal(t, specs.SpecVersionV1, spec.Version)
	})

	t.Run("handles empty migration list", func(t *testing.T) {
		t.Parallel()

		migrator := Migrator{}

		spec := &specs.Spec{
			Version:  specs.SpecVersionV0_1,
			Kind:     "test",
			Metadata: map[string]any{"name": "test"},
			Spec:     map[string]any{"id": "test"},
		}
		err := migrator.applyCommonMigrations(spec)

		require.NoError(t, err)
		assert.Equal(t, specs.SpecVersionV0_1, spec.Version) // Unchanged
	})
}

func TestMigrationPipeline(t *testing.T) {
	t.Parallel()

	t.Run("full migration pipeline from rudder/v0.1 to rudder/v1", func(t *testing.T) {
		t.Parallel()

		migrator := Migrator{
			commonMigrations: CommonMigrations{
				VersionUpdateMigration{},
			},
		}
		spec := &specs.Spec{
			Version:  specs.SpecVersionV0_1Variant, // Start with variant version
			Kind:     "test",
			Metadata: map[string]any{"name": "test"},
			Spec:     map[string]any{"id": "test"},
		}

		// Simulate the full migration pipeline
		// Phase 1: Pre-migrations
		err := migrator.applyCommonMigrations(spec)
		require.NoError(t, err)
		assert.Equal(t, specs.SpecVersionV1, spec.Version, "spec.Version should be updated to v1 after common migrations")

		// Phase 2: Provider-specific migrations would happen here
		// (skipped in this test) - to be added soon

	})
}
