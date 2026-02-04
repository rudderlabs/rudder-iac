package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers"
)

func TestToImportSpec(t *testing.T) {
	t.Run("creates valid spec with import metadata", func(t *testing.T) {
		kind := "transformation"
		metadataName := "my-transformation"
		workspaceMetadata := specs.WorkspaceImportMetadata{
			WorkspaceID: "workspace-123",
		}
		specData := map[string]any{
			"id":       "trans-1",
			"name":     "Test Transformation",
			"language": "javascript",
			"code":     "export function transformEvent(event) { return event; }",
		}

		result, err := handlers.ToImportSpec(kind, metadataName, workspaceMetadata, specData)

		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Equal(t, specs.SpecVersionV1, result.Version)
		assert.Equal(t, "transformation", result.Kind)
		assert.Equal(t, specData, result.Spec)
		assert.Equal(t, "my-transformation", result.Metadata["name"])

		importMetadata, ok := result.Metadata["import"].(map[string]any)
		require.True(t, ok)

		workspaces, ok := importMetadata["workspaces"].([]any)
		require.True(t, ok)
		require.Len(t, workspaces, 1)

		workspace, ok := workspaces[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "workspace-123", workspace["workspace_id"])
	})
}
