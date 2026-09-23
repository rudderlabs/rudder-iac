package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExpectedExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		language string
		expected string
	}{
		{
			name:     "javascript",
			language: LanguageJavaScript,
			expected: ExtJavaScript,
		},
		{
			name:     "python",
			language: LanguagePython,
			expected: ExtPython,
		},
		{
			name:     "unknown language",
			language: "ruby",
			expected: "",
		},
		{
			name:     "empty language",
			language: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := GetExpectedExtension(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveSpecRelativePath(t *testing.T) {
	t.Parallel()

	t.Run("valid relative paths", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name         string
			specFilePath string
			targetPath   string
			expected     string
		}{
			{
				name:         "simple relative path",
				specFilePath: "/tmp/project/spec.yaml",
				targetPath:   "code.js",
				expected:     "/tmp/project/code.js",
			},
			{
				name:         "relative path with subdirectory",
				specFilePath: "/tmp/project/spec.yaml",
				targetPath:   "src/code.js",
				expected:     "/tmp/project/src/code.js",
			},
			{
				name:         "spec in subdirectory",
				specFilePath: "/tmp/project/specs/spec.yaml",
				targetPath:   "code.js",
				expected:     "/tmp/project/specs/code.js",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := ResolveSpecRelativePath(tt.specFilePath, tt.targetPath)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("invalid paths", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name          string
			specFilePath  string
			targetPath    string
			expectedError string
		}{
			{
				name:          "absolute path",
				specFilePath:  "/tmp/project/spec.yaml",
				targetPath:    "/absolute/path/code.js",
				expectedError: "path must be relative to the spec file directory",
			},
			{
				name:          "parent traversal",
				specFilePath:  "/tmp/project/spec.yaml",
				targetPath:    "../code.js",
				expectedError: "path must not contain '..' segments",
			},
			{
				name:          "parent traversal in middle",
				specFilePath:  "/tmp/project/spec.yaml",
				targetPath:    "src/../code.js",
				expectedError: "path must not contain '..' segments",
			},
			{
				name:          "multiple parent traversals",
				specFilePath:  "/tmp/project/spec.yaml",
				targetPath:    "../../code.js",
				expectedError: "path must not contain '..' segments",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				result, err := ResolveSpecRelativePath(tt.specFilePath, tt.targetPath)
				require.Error(t, err)
				assert.Equal(t, "", result)
				assert.Equal(t, tt.expectedError, err.Error())
			})
		}
	})
}

func TestValidateSpecFile(t *testing.T) {
	t.Parallel()

	t.Run("valid file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "code.js")
		require.NoError(t, os.WriteFile(filePath, []byte("code"), 0o644))

		results := ValidateSpecFile(filePath)
		assert.Empty(t, results)
	})

	t.Run("non-existent file", func(t *testing.T) {
		t.Parallel()

		results := ValidateSpecFile("/tmp/non-existent-file.js")
		require.Len(t, results, 1)
		assert.Equal(t, "/file", results[0].Reference)
		assert.Equal(t, "path does not exist or is not accessible", results[0].Message)
	})

	t.Run("directory instead of file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		results := ValidateSpecFile(tmpDir)
		require.Len(t, results, 1)
		assert.Equal(t, "/file", results[0].Reference)
		assert.Equal(t, "path must be a file", results[0].Message)
	})
}
