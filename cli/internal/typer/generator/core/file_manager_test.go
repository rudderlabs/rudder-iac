package core

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManager_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("writes single file successfully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		file := File{
			Path:    "test.txt",
			Content: "Hello, World!",
		}

		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Verify file was created with correct content
		content, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", string(content))

		// Verify file permissions
		info, err := os.Stat(filepath.Join(tempDir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, fm.FileMode, info.Mode().Perm())
	})

	t.Run("creates nested directories", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		file := File{
			Path:    "nested/deep/dir/test.txt",
			Content: "Nested content",
		}

		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Verify file was created in nested directory
		content, err := os.ReadFile(filepath.Join(tempDir, "nested", "deep", "dir", "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Nested content", string(content))

		// Verify directory permissions
		info, err := os.Stat(filepath.Join(tempDir, "nested"))
		require.NoError(t, err)
		assert.Equal(t, fm.DirMode, info.Mode().Perm())
	})

	t.Run("overwrites existing file atomically", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		filePath := filepath.Join(tempDir, "test.txt")

		// Create initial file
		file := File{Path: "test.txt", Content: "Initial content"}
		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Verify initial content
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "Initial content", string(content))

		// Overwrite with new content
		file.Content = "Updated content"
		err = fm.WriteFile(file)
		require.NoError(t, err)

		// Verify updated content
		content, err = os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "Updated content", string(content))
	})

	t.Run("handles large file content", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)

		// Create large content (1MB)
		largeContent := strings.Repeat("A", 1024*1024)
		file := File{
			Path:    "large.txt",
			Content: largeContent,
		}

		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Verify large file was written correctly
		content, err := os.ReadFile(filepath.Join(tempDir, "large.txt"))
		require.NoError(t, err)
		assert.Equal(t, largeContent, string(content))
	})

	t.Run("custom file and directory modes", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := &FileManager{
			BaseDir:  tempDir,
			FileMode: 0600, // Owner read/write only
			DirMode:  0700, // Owner access only
		}

		file := File{
			Path:    "secure/test.txt",
			Content: "Secure content",
		}

		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Verify file permissions
		info, err := os.Stat(filepath.Join(tempDir, "secure", "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		// Verify directory permissions
		info, err = os.Stat(filepath.Join(tempDir, "secure"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
	})
}

func TestFileManager_WriteFiles(t *testing.T) {
	t.Parallel()

	t.Run("writes multiple files successfully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		files := []File{
			{Path: "file1.txt", Content: "Content 1"},
			{Path: "dir/file2.txt", Content: "Content 2"},
			{Path: "dir/subdir/file3.txt", Content: "Content 3"},
		}

		err := fm.WriteFiles(files)
		require.NoError(t, err)

		// Verify all files were created
		for _, file := range files {
			content, err := os.ReadFile(filepath.Join(tempDir, file.Path))
			require.NoError(t, err)
			assert.Equal(t, file.Content, string(content))
		}
	})

	t.Run("handles empty file list", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)

		err := fm.WriteFiles([]File{})
		require.NoError(t, err)
	})

	t.Run("fails fast on invalid file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		files := []File{
			{Path: "valid.txt", Content: "Valid content"},
			{Path: "", Content: "Invalid - empty path"},
			{Path: "another.txt", Content: "Another valid"},
		}

		err := fm.WriteFiles(files)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file 1")
		assert.Contains(t, err.Error(), "file path cannot be empty")

		// Verify no files were created
		_, err = os.Stat(filepath.Join(tempDir, "valid.txt"))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestFileManager_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty base directory", func(t *testing.T) {
		t.Parallel()

		fm := &FileManager{BaseDir: ""}
		file := File{Path: "test.txt", Content: "content"}

		err := fm.WriteFile(file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "base directory cannot be empty")
	})

	t.Run("empty file path", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		file := File{Path: "", Content: "content"}

		err := fm.WriteFile(file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file path cannot be empty")
	})

	t.Run("path traversal protection", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)

		testCases := []string{
			"../outside.txt",
			"dir/../../../outside.txt",
			"./dir/../outside.txt",
		}

		for _, path := range testCases {
			file := File{Path: path, Content: "content"}
			err := fm.WriteFile(file)
			require.Error(t, err, "Path should be rejected: %s", path)
			assert.Contains(t, err.Error(), "cannot contain '..'")
		}
	})

	t.Run("absolute path protection", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)

		var absolutePath string
		if runtime.GOOS == "windows" {
			absolutePath = "C:\\temp\\file.txt"
		} else {
			absolutePath = "/tmp/file.txt"
		}

		file := File{Path: absolutePath, Content: "content"}
		err := fm.WriteFile(file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file path must be relative")
	})
}

func TestFileManager_AtomicOperations(t *testing.T) {
	t.Parallel()

	t.Run("atomic write preserves existing file on failure", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)

		// Create initial file
		file := File{Path: "test.txt", Content: "Original content"}
		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Create a read-only directory to cause write failure
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err = os.MkdirAll(readOnlyDir, 0444) // Read-only
		require.NoError(t, err)

		// Try to write to read-only directory (should fail)
		file.Path = "readonly/test.txt"
		err = fm.WriteFile(file)
		require.Error(t, err)

		// Verify original file is still intact
		content, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Original content", string(content))
	})

	t.Run("no temporary files left behind", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		fm := NewFileManager(tempDir)
		file := File{Path: "test.txt", Content: "Test content"}

		err := fm.WriteFile(file)
		require.NoError(t, err)

		// Check for any temporary files
		entries, err := os.ReadDir(tempDir)
		require.NoError(t, err)

		for _, entry := range entries {
			assert.False(t, strings.Contains(entry.Name(), ".tmp"),
				"Temporary file found: %s", entry.Name())
		}
	})
}

func TestWriteFile_LegacyFunction(t *testing.T) {
	t.Parallel()

	t.Run("legacy function still works", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		file := File{
			Path:    "legacy.txt",
			Content: "Legacy content",
		}

		err := WriteFile(tempDir, file)
		require.NoError(t, err)

		// Verify file was created
		content, err := os.ReadFile(filepath.Join(tempDir, "legacy.txt"))
		require.NoError(t, err)
		assert.Equal(t, "Legacy content", string(content))
	})
}

func TestFileManager_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles directory creation failure", func(t *testing.T) {
		t.Parallel()

		// Skip on Windows as permission handling is different
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		tempDir := t.TempDir()

		// Create a file where we want to create a directory
		conflictFile := filepath.Join(tempDir, "conflict")
		err := os.WriteFile(conflictFile, []byte("blocking"), 0644)
		require.NoError(t, err)

		fm := NewFileManager(tempDir)
		file := File{
			Path:    "conflict/test.txt", // This should fail because "conflict" is a file, not a directory
			Content: "content",
		}

		err = fm.WriteFile(file)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating parent directory")
	})
}
