package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
	t.Parallel()

	t.Run("writes single file successfully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		file := File{
			Path:    "test.txt",
			Content: "Hello, World!",
		}

		err := WriteFile(tempDir, file)
		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(content))
	})

	t.Run("creates nested directories", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		file := File{
			Path:    "nested/dir/test.txt",
			Content: "Nested content",
		}

		err := WriteFile(tempDir, file)
		require.NoError(t, err)

		// Verify file was created in nested directory
		content, err := os.ReadFile(filepath.Join(tempDir, "nested", "dir", "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Nested content", string(content))
	})

	t.Run("empty output directory", func(t *testing.T) {
		t.Parallel()

		file := File{
			Path:    "test.txt",
			Content: "content",
		}

		err := WriteFile("", file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "output directory cannot be empty")
	})

	t.Run("empty file path", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		file := File{
			Path:    "",
			Content: "content",
		}

		err := WriteFile(tempDir, file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})

	t.Run("atomic write on failure", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		file := File{
			Path:    "existing.txt",
			Content: "Original content",
		}

		// Write the file first
		err := WriteFile(tempDir, file)
		require.NoError(t, err)

		// Create a read-only directory to cause write failure
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err = os.MkdirAll(readOnlyDir, 0444) // Read-only
		require.NoError(t, err)

		// Try to write to read-only directory
		file.Path = "readonly/test.txt"
		err = WriteFile(tempDir, file)

		// Should fail, but original file should remain intact
		require.Error(t, err)

		// Verify original file is still intact
		content, err := os.ReadFile(filepath.Join(tempDir, "existing.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Original content", string(content))
	})
}
