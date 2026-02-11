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

func TestMigrateImportMetadataToURN(t *testing.T) {
	t.Parallel()

	t.Run("converts LocalID to URN format", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version: specs.SpecVersionV0_1,
			Kind:    "destination",
			Metadata: map[string]any{
				"name": "test-destination",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "workspace-123",
							"resources": []any{
								map[string]any{
									"local_id":  "dest-1",
									"remote_id": "remote-dest-1",
								},
								map[string]any{
									"local_id":  "dest-2",
									"remote_id": "remote-dest-2",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "destination")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 1)
		require.Len(t, metadata.Import.Workspaces[0].Resources, 2)

		// Verify first resource
		assert.Equal(t, "destination:dest-1", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)
		assert.Equal(t, "remote-dest-1", metadata.Import.Workspaces[0].Resources[0].RemoteID)

		// Verify second resource
		assert.Equal(t, "destination:dest-2", metadata.Import.Workspaces[0].Resources[1].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[1].LocalID)
		assert.Equal(t, "remote-dest-2", metadata.Import.Workspaces[0].Resources[1].RemoteID)
	})

	t.Run("skips resources that already have URN", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version: specs.SpecVersionV0_1,
			Kind:    "source",
			Metadata: map[string]any{
				"name": "test-source",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "workspace-123",
							"resources": []any{
								map[string]any{
									"urn":       "source:src-1",
									"remote_id": "remote-src-1",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "source")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 1)
		require.Len(t, metadata.Import.Workspaces[0].Resources, 1)

		// URN should remain unchanged
		assert.Equal(t, "source:src-1", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)
		assert.Equal(t, "remote-src-1", metadata.Import.Workspaces[0].Resources[0].RemoteID)
	})

	t.Run("handles spec with no import metadata", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version:  specs.SpecVersionV0_1,
			Kind:     "connection",
			Metadata: map[string]any{"name": "test-connection"},
			Spec:     map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "connection")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		assert.Nil(t, metadata.Import)
	})

	t.Run("handles multiple workspaces", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version: specs.SpecVersionV0_1,
			Kind:    "destination",
			Metadata: map[string]any{
				"name": "test-destination",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "workspace-1",
							"resources": []any{
								map[string]any{
									"local_id":  "dest-1",
									"remote_id": "remote-dest-1",
								},
							},
						},
						map[string]any{
							"workspace_id": "workspace-2",
							"resources": []any{
								map[string]any{
									"local_id":  "dest-2",
									"remote_id": "remote-dest-2",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "destination")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 2)

		// Verify first workspace
		assert.Equal(t, "workspace-1", metadata.Import.Workspaces[0].WorkspaceID)
		assert.Equal(t, "destination:dest-1", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)

		// Verify second workspace
		assert.Equal(t, "workspace-2", metadata.Import.Workspaces[1].WorkspaceID)
		assert.Equal(t, "destination:dest-2", metadata.Import.Workspaces[1].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[1].Resources[0].LocalID)
	})

	t.Run("handles empty resources list", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version: specs.SpecVersionV0_1,
			Kind:    "source",
			Metadata: map[string]any{
				"name": "test-source",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "workspace-123",
							"resources":    []any{},
						},
					},
				},
			},
			Spec: map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "source")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 1)
		assert.Len(t, metadata.Import.Workspaces[0].Resources, 0)
	})

	t.Run("handles mixed URN and LocalID resources", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version: specs.SpecVersionV0_1,
			Kind:    "destination",
			Metadata: map[string]any{
				"name": "test-destination",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "workspace-123",
							"resources": []any{
								map[string]any{
									"urn":       "destination:dest-1",
									"remote_id": "remote-dest-1",
								},
								map[string]any{
									"local_id":  "dest-2",
									"remote_id": "remote-dest-2",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "destination")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 1)
		require.Len(t, metadata.Import.Workspaces[0].Resources, 2)

		// First resource should remain unchanged (already has URN)
		assert.Equal(t, "destination:dest-1", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)

		// Second resource should be converted
		assert.Equal(t, "destination:dest-2", metadata.Import.Workspaces[0].Resources[1].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[1].LocalID)
	})

	t.Run("handles resource with empty LocalID", func(t *testing.T) {
		t.Parallel()

		spec := &specs.Spec{
			Version: specs.SpecVersionV0_1,
			Kind:    "source",
			Metadata: map[string]any{
				"name": "test-source",
				"import": map[string]any{
					"workspaces": []any{
						map[string]any{
							"workspace_id": "workspace-123",
							"resources": []any{
								map[string]any{
									"local_id":  "",
									"remote_id": "remote-src-1",
								},
							},
						},
					},
				},
			},
			Spec: map[string]any{"id": "test"},
		}

		err := MigrateImportMetadataToURN(spec, "source")
		require.NoError(t, err)

		metadata, err := spec.CommonMetadata()
		require.NoError(t, err)
		require.NotNil(t, metadata.Import)
		require.Len(t, metadata.Import.Workspaces, 1)
		require.Len(t, metadata.Import.Workspaces[0].Resources, 1)

		// Should not create URN for empty LocalID
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].URN)
		assert.Equal(t, "", metadata.Import.Workspaces[0].Resources[0].LocalID)
	})
}
