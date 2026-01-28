package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoader_Load(t *testing.T) {
	setupTestDir := func(t *testing.T, files map[string]string) string {
		t.Helper()
		tmpDir, err := os.MkdirTemp("", "loader_test_")
		require.NoError(t, err)

		for name, content := range files {
			filePath := filepath.Join(tmpDir, name)
			err := os.MkdirAll(filepath.Dir(filePath), 0755)
			require.NoError(t, err)
			err = os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}
		return tmpDir
	}

	testContent := "version: rudder/0.1\nkind: source"

	t.Run("Loads YAML and YML files", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"source.yaml":           testContent,
			"destination.yml":       testContent,
			"subdir/nested.yaml":    testContent,
			"subdir/deep/file.yml": testContent,
		})
		defer os.RemoveAll(tmpDir)

		l := &loader.Loader{}
		specs, err := l.Load(tmpDir)

		require.NoError(t, err)
		assert.Len(t, specs, 4)

		for path, rawSpec := range specs {
			assert.NotNil(t, rawSpec)
			assert.NotEmpty(t, rawSpec.Data)
			assert.Equal(t, testContent, string(rawSpec.Data))
			assert.True(t, filepath.IsAbs(path))
			assert.Contains(t, path, tmpDir)
		}
	})

	t.Run("Empty directory returns empty map", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{})
		defer os.RemoveAll(tmpDir)

		l := &loader.Loader{}
		specs, err := l.Load(tmpDir)

		require.NoError(t, err)
		assert.Empty(t, specs)
	})

	t.Run("Ignores non-YAML files", func(t *testing.T) {
		tmpDir := setupTestDir(t, map[string]string{
			"notes.txt":   "some notes",
			"image.png":   "binarydata",
			"README.md":   "readme",
			"config.json": "{}",
		})
		defer os.RemoveAll(tmpDir)

		l := &loader.Loader{}
		specs, err := l.Load(tmpDir)

		require.NoError(t, err)
		assert.Empty(t, specs)
	})

	t.Run("Non-existent location returns error", func(t *testing.T) {
		nonExistentPath := filepath.Join(os.TempDir(), "non_existent_loader_test_dir_12345abcde")
		l := &loader.Loader{}
		_, err := l.Load(nonExistentPath)

		require.Error(t, err)
		assert.Contains(t, err.Error(), nonExistentPath)
	})
}
