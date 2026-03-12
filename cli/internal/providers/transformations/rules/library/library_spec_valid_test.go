package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLibrarySpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewLibrarySpecSyntaxValidRule()

	assert.Equal(t, "transformations/transformation-library/spec-syntax-valid", rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.Equal(t, "transformation library spec syntax must be valid", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns("transformation-library"), rule.AppliesTo())
}

func TestValidateLibrarySpec_ValidSpecs(t *testing.T) {
	t.Parallel()

	t.Run("inline code", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:          "lib-1",
			Name:        "My Library",
			ImportName:  "myLibrary",
			Language:    "javascript",
			Code:        "export function helper() { return 42; }",
			Description: "A helper library",
		}

		results := validateLibrarySpec("", "", "/tmp/spec.yaml", nil, spec)
		assert.Empty(t, results)
	})

	t.Run("relative file with correct extension", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir   = t.TempDir()
			specPath = filepath.Join(tmpDir, "libraries", "spec.yaml")
			specDir  = filepath.Dir(specPath)
			codeDir  = filepath.Join(specDir, "code")
		)

		require.NoError(t, os.MkdirAll(specDir, 0o755))
		require.NoError(t, os.MkdirAll(codeDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(codeDir, "lib.js"), []byte("export function helper() {}"), 0o644))

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			File:       "code/lib.js",
		}

		results := validateLibrarySpec("", "", specPath, nil, spec)
		assert.Empty(t, results)
	})

	t.Run("python file with correct extension", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir   = t.TempDir()
			specPath = filepath.Join(tmpDir, "spec.yaml")
		)

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "lib.py"), []byte("def helper():\n    return 42"), 0o644))

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "python",
			File:       "lib.py",
		}

		results := validateLibrarySpec("", "", specPath, nil, spec)
		assert.Empty(t, results)
	})
}

func TestValidateLibrarySpec_InvalidSpecs(t *testing.T) {
	t.Parallel()

	t.Run("missing required fields", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			spec              specs.TransformationLibrarySpec
			expectedReference string
			expectedMessage   string
		}{
			{
				name: "missing id",
				spec: specs.TransformationLibrarySpec{
					Name:       "My Library",
					ImportName: "myLibrary",
					Language:   "javascript",
					Code:       "export function helper() {}",
				},
				expectedReference: "/id",
				expectedMessage:   "'id' is required",
			},
			{
				name: "missing name",
				spec: specs.TransformationLibrarySpec{
					ID:         "lib-1",
					ImportName: "myLibrary",
					Language:   "javascript",
					Code:       "export function helper() {}",
				},
				expectedReference: "/name",
				expectedMessage:   "'name' is required",
			},
			{
				name: "missing import_name",
				spec: specs.TransformationLibrarySpec{
					ID:       "lib-1",
					Name:     "My Library",
					Language: "javascript",
					Code:     "export function helper() {}",
				},
				expectedReference: "/import_name",
				expectedMessage:   "'import_name' is required",
			},
			{
				name: "missing language",
				spec: specs.TransformationLibrarySpec{
					ID:         "lib-1",
					Name:       "My Library",
					ImportName: "myLibrary",
					Code:       "export function helper() {}",
				},
				expectedReference: "/language",
				expectedMessage:   "'language' is required",
			},
			{
				name: "invalid language",
				spec: specs.TransformationLibrarySpec{
					ID:         "lib-1",
					Name:       "My Library",
					ImportName: "myLibrary",
					Language:   "ruby",
					Code:       "def helper; end",
				},
				expectedReference: "/language",
				expectedMessage:   "'language' must be one of [javascript python]",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				results := validateLibrarySpec("", "", "/tmp/spec.yaml", nil, tt.spec)

				assert.Equal(t, []string{tt.expectedReference}, extractReferences(results))
				assert.Equal(t, []string{tt.expectedMessage}, extractMessages(results))
			})
		}
	})

	t.Run("missing code and file produces cross-field errors", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
		}

		results := validateLibrarySpec("", "", "/tmp/spec.yaml", nil, spec)

		assert.ElementsMatch(t, []string{"/code", "/file"}, extractReferences(results))
		assert.ElementsMatch(t, []string{
			"'code' is required when 'file' is not specified",
			"'file' is required when 'code' is not specified",
		}, extractMessages(results))
	})

	t.Run("code and file together are rejected", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationLibrarySpec{
			ID:         "lib-1",
			Name:       "My Library",
			ImportName: "myLibrary",
			Language:   "javascript",
			Code:       "export function helper() {}",
			File:       "lib.js",
		}

		results := validateLibrarySpec("", "", "/tmp/spec.yaml", nil, spec)

		assert.ElementsMatch(t, []string{
			"/code",
			"/file",
			"/file",
		}, extractReferences(results))

		assert.Contains(t, extractMessages(results), "'code' and 'file' cannot be specified together")
		assert.Contains(t, extractMessages(results), "'file' and 'code' cannot be specified together")
		assert.Contains(t, extractMessages(results), "path does not exist or is not accessible")
	})

	t.Run("import_name not camelCase of name", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			libraryName       string
			importName        string
			expectedReference string
			expectedMessage   string
		}{
			{
				name:              "wrong casing",
				libraryName:       "My Library",
				importName:        "MyLibrary",
				expectedReference: "/import_name",
				expectedMessage:   "'import_name' must be camelCase of 'name'",
			},
			{
				name:              "completely different",
				libraryName:       "Math Utils",
				importName:        "helper",
				expectedReference: "/import_name",
				expectedMessage:   "'import_name' must be camelCase of 'name'",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				spec := specs.TransformationLibrarySpec{
					ID:         "lib-1",
					Name:       tt.libraryName,
					ImportName: tt.importName,
					Language:   "javascript",
					Code:       "export function helper() {}",
				}

				results := validateLibrarySpec("", "", "/tmp/spec.yaml", nil, spec)

				assert.Contains(t, extractReferences(results), tt.expectedReference)
				assert.Contains(t, extractMessages(results), tt.expectedMessage)
			})
		}
	})

	t.Run("absolute and parent traversal paths are rejected", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			filePath          string
			expectedReference string
			expectedMessage   string
		}{
			{
				name:              "absolute path",
				filePath:          "/tmp/lib.js",
				expectedReference: "/file",
				expectedMessage:   "path must be relative to the spec file directory",
			},
			{
				name:              "parent traversal",
				filePath:          "../lib.js",
				expectedReference: "/file",
				expectedMessage:   "path must not contain '..' segments",
			},
			{
				name:              "nested parent traversal",
				filePath:          "code/../../../lib.js",
				expectedReference: "/file",
				expectedMessage:   "path must not contain '..' segments",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				spec := specs.TransformationLibrarySpec{
					ID:         "lib-1",
					Name:       "My Library",
					ImportName: "myLibrary",
					Language:   "javascript",
					File:       tt.filePath,
				}

				results := validateLibrarySpec("", "", "/tmp/spec.yaml", nil, spec)

				assert.Contains(t, extractReferences(results), tt.expectedReference)
				assert.Contains(t, extractMessages(results), tt.expectedMessage)
			})
		}
	})

	t.Run("file path must exist and be a regular file", func(t *testing.T) {
		t.Parallel()

		t.Run("non-existent file", func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			specPath := filepath.Join(tmpDir, "spec.yaml")

			spec := specs.TransformationLibrarySpec{
				ID:         "lib-1",
				Name:       "My Library",
				ImportName: "myLibrary",
				Language:   "javascript",
				File:       "lib.js",
			}

			results := validateLibrarySpec("", "", specPath, nil, spec)

			assert.Contains(t, extractReferences(results), "/file")
			assert.Contains(t, extractMessages(results), "path does not exist or is not accessible")
		})

		t.Run("directory instead of file", func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			specPath := filepath.Join(tmpDir, "spec.yaml")
			dirPath := filepath.Join(tmpDir, "code")
			require.NoError(t, os.MkdirAll(dirPath, 0o755))

			spec := specs.TransformationLibrarySpec{
				ID:         "lib-1",
				Name:       "My Library",
				ImportName: "myLibrary",
				Language:   "javascript",
				File:       "code",
			}

			results := validateLibrarySpec("", "", specPath, nil, spec)

			assert.ElementsMatch(t, []string{"/file", "/file"}, extractReferences(results))
			assert.Contains(t, extractMessages(results), "path must be a file")
			assert.Contains(t, extractMessages(results), "file extension must be '.js'")
		})
	})

	t.Run("file extension must match language", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			language          string
			fileName          string
			expectedReference string
			expectedMessage   string
		}{
			{
				name:              "javascript with python extension",
				language:          "javascript",
				fileName:          "lib.py",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.js'",
			},
			{
				name:              "python with javascript extension",
				language:          "python",
				fileName:          "lib.js",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.py'",
			},
			{
				name:              "javascript with wrong extension",
				language:          "javascript",
				fileName:          "lib.txt",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.js'",
			},
			{
				name:              "python with no extension",
				language:          "python",
				fileName:          "lib",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.py'",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tmpDir := t.TempDir()
				specPath := filepath.Join(tmpDir, "spec.yaml")
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, tt.fileName), []byte("code"), 0o644))

				spec := specs.TransformationLibrarySpec{
					ID:         "lib-1",
					Name:       "My Library",
					ImportName: "myLibrary",
					Language:   tt.language,
					File:       tt.fileName,
				}

				results := validateLibrarySpec("", "", specPath, nil, spec)

				assert.Contains(t, extractReferences(results), tt.expectedReference)
				assert.Contains(t, extractMessages(results), tt.expectedMessage)
			})
		}
	})
}

func extractReferences(results []vrules.ValidationResult) []string {
	references := make([]string, len(results))
	for idx, result := range results {
		references[idx] = result.Reference
	}

	return references
}

func extractMessages(results []vrules.ValidationResult) []string {
	messages := make([]string, len(results))
	for idx, result := range results {
		messages[idx] = result.Message
	}

	return messages
}
