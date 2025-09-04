package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileManager(t *testing.T) {
	t.Parallel()

	t.Run("successful creation", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		
		require.NoError(t, err)
		assert.NotNil(t, fm)
		assert.Equal(t, tempDir, fm.GetOutputDir())
	})

	t.Run("creates output directory if it doesn't exist", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		newDir := filepath.Join(tempDir, "new", "nested", "directory")
		
		fm, err := NewFileManager(newDir)
		
		require.NoError(t, err)
		assert.NotNil(t, fm)
		
		// Verify the directory was created
		info, err := os.Stat(newDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("empty output directory", func(t *testing.T) {
		t.Parallel()

		_, err := NewFileManager("")
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "output directory cannot be empty")
	})
}

func TestFileManager_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("writes single file successfully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		file := File{
			Path:    "test.txt",
			Content: "Hello, World!",
		}

		err = fm.WriteFile(file)
		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(content))
	})

	t.Run("creates nested directories", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		file := File{
			Path:    "nested/dir/test.txt",
			Content: "Nested content",
		}

		err = fm.WriteFile(file)
		require.NoError(t, err)

		// Verify file was created in nested directory
		content, err := os.ReadFile(filepath.Join(tempDir, "nested", "dir", "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Nested content", string(content))
	})

	t.Run("empty file path", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		file := File{
			Path:    "",
			Content: "content",
		}

		err = fm.WriteFile(file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})

	t.Run("atomic write on failure", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		// Create a file that will cause a write failure
		file := File{
			Path:    "existing.txt",
			Content: "Original content",
		}

		// Write the file first
		err = fm.WriteFile(file)
		require.NoError(t, err)

		// Create a read-only directory to cause write failure
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err = os.MkdirAll(readOnlyDir, 0444) // Read-only
		require.NoError(t, err)

		// Try to write to read-only directory
		file.Path = "readonly/test.txt"
		err = fm.WriteFile(file)
		
		// Should fail, but original file should remain intact
		require.Error(t, err)
		
		// Verify original file is still intact
		content, err := os.ReadFile(filepath.Join(tempDir, "existing.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Original content", string(content))
	})
}

func TestFileManager_WriteFiles(t *testing.T) {
	t.Parallel()

	t.Run("writes multiple files successfully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		files := []File{
			{
				Path:    "file1.txt",
				Content: "Content 1",
			},
			{
				Path:    "nested/file2.txt",
				Content: "Content 2",
			},
			{
				Path:    "file3.txt",
				Content: "Content 3",
			},
		}

		err = fm.WriteFiles(files)
		require.NoError(t, err)

		// Verify all files were created
		for i, file := range files {
			content, err := os.ReadFile(filepath.Join(tempDir, file.Path))
			require.NoError(t, err, "File %d failed", i)
			assert.Equal(t, file.Content, string(content), "File %d content mismatch", i)
		}
	})

	t.Run("empty files slice", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		err = fm.WriteFiles([]File{})
		require.NoError(t, err)
	})

	t.Run("stops on first error", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm, err := NewFileManager(tempDir)
		require.NoError(t, err)

		files := []File{
			{
				Path:    "valid.txt",
				Content: "Valid content",
			},
			{
				Path:    "", // Invalid - empty path
				Content: "Invalid content",
			},
			{
				Path:    "should_not_be_created.txt",
				Content: "This should not be created",
			},
		}

		err = fm.WriteFiles(files)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "writing file : file path cannot be empty")

		// Verify only the first file was created
		content, err := os.ReadFile(filepath.Join(tempDir, "valid.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Valid content", string(content))

		// Verify the third file was not created
		_, err = os.ReadFile(filepath.Join(tempDir, "should_not_be_created.txt"))
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestFileManager_EnsureOutputDir(t *testing.T) {
	t.Parallel()

	t.Run("creates output directory if it doesn't exist", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		newDir := filepath.Join(tempDir, "new", "directory")
		
		fm := &FileManager{outputDir: newDir}
		
		err := fm.EnsureOutputDir()
		require.NoError(t, err)
		
		// Verify the directory was created
		info, err := os.Stat(newDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("no error if directory already exists", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := &FileManager{outputDir: tempDir}
		
		err := fm.EnsureOutputDir()
		require.NoError(t, err)
	})
}

func TestFileManager_GetOutputDir(t *testing.T) {
	t.Parallel()

	expectedDir := "/test/output/dir"
	fm := &FileManager{outputDir: expectedDir}
	
	assert.Equal(t, expectedDir, fm.GetOutputDir())
}
