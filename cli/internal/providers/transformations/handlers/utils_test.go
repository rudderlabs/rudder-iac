package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers"
)

func TestToImportSpec(t *testing.T) {
	t.Run("creates clean spec without inline import metadata", func(t *testing.T) {
		kind := "transformation"
		metadataName := "my-transformation"
		specData := map[string]any{
			"id":       "trans-1",
			"name":     "Test Transformation",
			"language": "javascript",
			"code":     "export function transformEvent(event) { return event; }",
		}

		result, err := handlers.ToImportSpec(kind, metadataName, specData)

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, specs.SpecVersionV1, result.Version)
		assert.Equal(t, "transformation", result.Kind)
		assert.Equal(t, specData, result.Spec)
		assert.Equal(t, "my-transformation", result.Metadata["name"])

		// Import metadata no longer lives inline; it travels via the ImportEntry
		// slice returned from FormatForExport and is aggregated into import-manifest.yaml.
		_, hasImport := result.Metadata["import"]
		assert.False(t, hasImport, "emitted specs must not carry inline metadata.import")
	})
}
