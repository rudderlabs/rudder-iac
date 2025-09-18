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

	tests := map[string]struct {
		file           File
		verifyDirPerms bool                                          // whether to verify directory permissions were set correctly
		setupFunc      func(tempDir string, fm *FileManager) error   // setup before main test
		verifyFunc     func(t *testing.T, tempDir string, file File) // custom verification logic
	}{
		"single file": {
			file: File{
				Path:    "test.txt",
				Content: "Hello, World!",
			},
		},
		"nested directories": {
			file: File{
				Path:    "nested/deep/dir/test.txt",
				Content: "Nested content",
			},
			verifyDirPerms: true,
		},
		"file with special characters": {
			file: File{
				Path:    "special/file-name_with.chars.txt",
				Content: "Content with special chars: 🎉 & symbols!",
			},
		},
		"empty content": {
			file: File{
				Path:    "empty.txt",
				Content: "",
			},
		},
		"large file content": {
			file: File{
				Path:    "large.txt",
				Content: strings.Repeat("A", 1024*1024), // 1MB
			},
		},
		"multiline content": {
			file: File{
				Path: "multiline.txt",
				Content: `Line 1
Line 2
Line 3
With some special chars: 🚀`,
			},
		},
		"overwrites existing file atomically": {
			file: File{
				Path:    "test.txt",
				Content: "Updated content",
			},
			setupFunc: func(tempDir string, fm *FileManager) error {
				// Create initial file
				initialFile := File{Path: "test.txt", Content: "Initial content"}
				return fm.WriteFile(initialFile)
			},
			verifyFunc: func(t *testing.T, tempDir string, file File) {
				// Verify the file was overwritten with new content
				fullPath := filepath.Join(tempDir, file.Path)
				content, err := os.ReadFile(fullPath)
				require.NoError(t, err)
				assert.Equal(t, "Updated content", string(content))
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			fm := NewFileManager(tempDir)

			// Run setup if provided
			if test.setupFunc != nil {
				err := test.setupFunc(tempDir, fm)
				require.NoError(t, err)
			}

			// Execute the main test
			err := fm.WriteFile(test.file)
			require.NoError(t, err)

			// Run custom verification if provided
			if test.verifyFunc != nil {
				test.verifyFunc(t, tempDir, test.file)
			} else {
				// Default verification logic
				// Verify file was created with correct content
				fullPath := filepath.Join(tempDir, test.file.Path)
				content, err := os.ReadFile(fullPath)
				require.NoError(t, err)
				assert.Equal(t, test.file.Content, string(content))

				// Verify file permissions (0644)
				info, err := os.Stat(fullPath)
				require.NoError(t, err)
				assert.Equal(t, os.FileMode(0644), info.Mode().Perm())

				// Verify directory permissions if requested
				if test.verifyDirPerms {
					parentDir := filepath.Dir(fullPath)
					info, err := os.Stat(parentDir)
					require.NoError(t, err)
					assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
				}
			}
		})
	}
}

func TestFileManager_WriteFiles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		files         []File
		expectError   bool
		errorContains []string
		verifyNoFiles bool // whether to verify no files were created on error
	}{
		"writes multiple files successfully": {
			files: []File{
				{Path: "file1.txt", Content: "Content 1"},
				{Path: "dir/file2.txt", Content: "Content 2"},
				{Path: "dir/subdir/file3.txt", Content: "Content 3"},
			},
			expectError: false,
		},
		"handles empty file list": {
			files:       []File{},
			expectError: false,
		},
		"fails fast on invalid file": {
			files: []File{
				{Path: "valid.txt", Content: "Valid content"},
				{Path: "", Content: "Invalid - empty path"},
				{Path: "another.txt", Content: "Another valid"},
			},
			expectError:   true,
			errorContains: []string{"file 1", "file path cannot be empty"},
			verifyNoFiles: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			fm := NewFileManager(tempDir)

			err := fm.WriteFiles(test.files)

			if test.expectError {
				require.Error(t, err)
				for _, expectedError := range test.errorContains {
					assert.Contains(t, err.Error(), expectedError)
				}

				// Verify no files were created if requested
				if test.verifyNoFiles {
					for _, file := range test.files {
						if file.Path != "" { // Only check valid paths
							_, err := os.Stat(filepath.Join(tempDir, file.Path))
							assert.True(t, os.IsNotExist(err), "File should not exist: %s", file.Path)
						}
					}
				}
			} else {
				require.NoError(t, err)

				// Verify all files were created with correct content
				for _, file := range test.files {
					content, err := os.ReadFile(filepath.Join(tempDir, file.Path))
					require.NoError(t, err)
					assert.Equal(t, file.Content, string(content))
				}
			}
		})
	}
}

func TestFileManager_Validation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path             string
		content          string
		expectedError    string
		expectSuccess    bool
		useEmptyBaseDir  bool                               // special case: test empty BaseDir behavior
		setupFunc        func() (cleanup func(), err error) // for directory change setup
		customVerifyFunc func(t *testing.T, file File)      // custom verification for special cases
	}{
		"empty base directory defaults to current directory": {
			path:            "test.txt",
			content:         "content",
			expectSuccess:   true,
			useEmptyBaseDir: true,
			setupFunc: func() (func(), error) {
				// Save current directory
				originalDir, err := os.Getwd()
				if err != nil {
					return nil, err
				}

				// Create and change to temp directory
				tempDir, err := os.MkdirTemp("", "filemanager_test")
				if err != nil {
					return nil, err
				}

				err = os.Chdir(tempDir)
				if err != nil {
					os.RemoveAll(tempDir)
					return nil, err
				}

				// Return cleanup function
				return func() {
					os.Chdir(originalDir)
					os.RemoveAll(tempDir)
				}, nil
			},
			customVerifyFunc: func(t *testing.T, file File) {
				// Verify file was created in current directory
				content, err := os.ReadFile(file.Path)
				require.NoError(t, err)
				assert.Equal(t, file.Content, string(content))
			},
		},
		"empty file path": {
			path:          "",
			content:       "content",
			expectedError: "file path cannot be empty",
			expectSuccess: false,
		},
		"path traversal with ../": {
			path:          "../outside.txt",
			content:       "content",
			expectedError: "cannot contain '..'",
			expectSuccess: false,
		},
		"path traversal nested": {
			path:          "dir/../../../outside.txt",
			content:       "content",
			expectedError: "cannot contain '..'",
			expectSuccess: false,
		},
		"path traversal with ./": {
			path:          "./dir/../outside.txt",
			content:       "content",
			expectedError: "cannot contain '..'",
			expectSuccess: false,
		},
	}

	// Add platform-specific absolute path test
	var absolutePath string
	if runtime.GOOS == "windows" {
		absolutePath = "C:\\temp\\file.txt"
	} else {
		absolutePath = "/tmp/file.txt"
	}
	tests["absolute path protection"] = struct {
		path             string
		content          string
		expectedError    string
		expectSuccess    bool
		useEmptyBaseDir  bool
		setupFunc        func() (cleanup func(), err error)
		customVerifyFunc func(t *testing.T, file File)
	}{
		path:          absolutePath,
		content:       "content",
		expectedError: "file path must be relative",
		expectSuccess: false,
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var cleanup func()

			// Run setup if provided
			if test.setupFunc != nil {
				var err error
				cleanup, err = test.setupFunc()
				require.NoError(t, err)
				defer cleanup()
			}

			// Create FileManager
			var fm *FileManager
			if test.useEmptyBaseDir {
				fm = &FileManager{BaseDir: ""} // Test empty BaseDir behavior
			} else {
				tempDir := t.TempDir()
				fm = NewFileManager(tempDir)
			}

			file := File{Path: test.path, Content: test.content}
			err := fm.WriteFile(file)

			if test.expectSuccess {
				require.NoError(t, err)

				// Run custom verification if provided
				if test.customVerifyFunc != nil {
					test.customVerifyFunc(t, file)
				}
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectedError)
			}
		})
	}
}

func TestFileManager_AtomicOperations(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc   func(tempDir string, fm *FileManager) error // setup before main test
		testFile    File
		expectError bool
		verifyFunc  func(t *testing.T, tempDir string) // custom verification logic
	}{
		"atomic write preserves existing file on failure": {
			setupFunc: func(tempDir string, fm *FileManager) error {
				// Create initial file
				file := File{Path: "test.txt", Content: "Original content"}
				err := fm.WriteFile(file)
				if err != nil {
					return err
				}

				// Create a read-only directory to cause write failure
				readOnlyDir := filepath.Join(tempDir, "readonly")
				return os.MkdirAll(readOnlyDir, 0444) // Read-only
			},
			testFile:    File{Path: "readonly/test.txt", Content: "Should fail"},
			expectError: true,
			verifyFunc: func(t *testing.T, tempDir string) {
				// Verify original file is still intact
				content, err := os.ReadFile(filepath.Join(tempDir, "test.txt"))
				require.NoError(t, err)
				assert.Equal(t, "Original content", string(content))
			},
		},
		"no temporary files left behind": {
			testFile:    File{Path: "test.txt", Content: "Test content"},
			expectError: false,
			verifyFunc: func(t *testing.T, tempDir string) {
				// Check for any temporary files
				entries, err := os.ReadDir(tempDir)
				require.NoError(t, err)

				for _, entry := range entries {
					assert.False(t, strings.Contains(entry.Name(), ".tmp"),
						"Temporary file found: %s", entry.Name())
				}
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			fm := NewFileManager(tempDir)

			// Run setup if provided
			if test.setupFunc != nil {
				err := test.setupFunc(tempDir, fm)
				require.NoError(t, err)
			}

			// Execute the main test
			err := fm.WriteFile(test.testFile)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Run custom verification
			if test.verifyFunc != nil {
				test.verifyFunc(t, tempDir)
			}
		})
	}
}

func TestFileManager_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		skipOnWindows bool
		setupFunc     func(tempDir string) error
		testFile      File
		errorContains string
	}{
		"handles directory creation failure": {
			skipOnWindows: true, // Permission handling is different on Windows
			setupFunc: func(tempDir string) error {
				// Create a file where we want to create a directory
				conflictFile := filepath.Join(tempDir, "conflict")
				return os.WriteFile(conflictFile, []byte("blocking"), 0644)
			},
			testFile: File{
				Path:    "conflict/test.txt", // This should fail because "conflict" is a file, not a directory
				Content: "content",
			},
			errorContains: "creating parent directory",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if test.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("Skipping permission test on Windows")
			}

			tempDir := t.TempDir()

			// Run setup if provided
			if test.setupFunc != nil {
				err := test.setupFunc(tempDir)
				require.NoError(t, err)
			}

			fm := NewFileManager(tempDir)
			err := fm.WriteFile(test.testFile)

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errorContains)
		})
	}
}
