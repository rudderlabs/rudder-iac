package transformation

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

func TestNewTransformationSpecSyntaxValidRule_Metadata(t *testing.T) {
	t.Parallel()

	rule := NewTransformationSpecSyntaxValidRule()

	assert.Equal(t, "transformations/transformation/spec-syntax-valid", rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.Equal(t, "transformation spec syntax must be valid", rule.Description())
	assert.Equal(t, prules.V1VersionPatterns("transformation"), rule.AppliesTo())
}

func TestValidateTransformationSpec_ValidSpecs(t *testing.T) {
	t.Parallel()

	t.Run("inline code with omitted input and output paths", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			Tests: []specs.TransformationTest{
				{Name: "Suite 1"},
			},
		}

		results := validateTransformationSpec("", "", "/tmp/spec.yaml", nil, spec)
		assert.Empty(t, results)
	})

	t.Run("relative file and explicit test directories", func(t *testing.T) {
		t.Parallel()

		var (
			tmpDir    = t.TempDir()
			specPath  = filepath.Join(tmpDir, "transformations", "spec.yaml")
			specDir   = filepath.Dir(specPath)
			codeDir   = filepath.Join(specDir, "scripts")
			inputDir  = filepath.Join(specDir, "fixtures", "input")
			outputDir = filepath.Join(specDir, "fixtures", "output")
		)

		require.NoError(t, os.MkdirAll(specDir, 0o755))
		require.NoError(t, os.MkdirAll(codeDir, 0o755))
		require.NoError(t, os.MkdirAll(inputDir, 0o755))
		require.NoError(t, os.MkdirAll(outputDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(codeDir, "transform.js"), []byte("export function transformEvent(event, metadata) { return event; }"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "event.json"), []byte(`{"ok":true}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(outputDir, "result.json"), []byte(`[{"ok":true}]`), 0o644))

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			File:     "scripts/transform.js",
			Tests: []specs.TransformationTest{
				{
					Name:   "Suite 1",
					Input:  "fixtures/input",
					Output: "fixtures/output",
				},
			},
		}

		results := validateTransformationSpec("", "", specPath, nil, spec)
		assert.Empty(t, results)
	})
}

func TestValidateTransformationSpec_InvalidSpecs(t *testing.T) {
	t.Parallel()

	t.Run("missing required top-level fields and invalid language are rejected", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name              string
			spec              specs.TransformationSpec
			expectedReference string
			expectedMessage   string
		}{
			{
				name: "missing id",
				spec: specs.TransformationSpec{
					Name:     "Transformation",
					Language: "javascript",
					Code:     "export function transformEvent(event, metadata) { return event; }",
				},
				expectedReference: "/id",
				expectedMessage:   "'id' is required",
			},
			{
				name: "missing name",
				spec: specs.TransformationSpec{
					ID:       "trans-1",
					Language: "javascript",
					Code:     "export function transformEvent(event, metadata) { return event; }",
				},
				expectedReference: "/name",
				expectedMessage:   "'name' is required",
			},
			{
				name: "missing language",
				spec: specs.TransformationSpec{
					ID:   "trans-1",
					Name: "Transformation",
					Code: "export function transformEvent(event, metadata) { return event; }",
				},
				expectedReference: "/language",
				expectedMessage:   "'language' is required",
			},
			{
				name: "invalid language",
				spec: specs.TransformationSpec{
					ID:       "trans-1",
					Name:     "Transformation",
					Language: "ruby",
					Code:     "export function transformEvent(event, metadata) { return event; }",
				},
				expectedReference: "/language",
				expectedMessage:   "'language' must be one of [javascript python]",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				results := validateTransformationSpec("", "", "/tmp/spec.yaml", nil, tt.spec)

				assert.Equal(t, []string{tt.expectedReference}, extractReferences(results))
				assert.Equal(t, []string{tt.expectedMessage}, extractMessages(results))
			})
		}
	})

	t.Run("missing code and file produces cross-field errors", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
		}

		results := validateTransformationSpec("", "", "/tmp/spec.yaml", nil, spec)

		assert.ElementsMatch(t, []string{"/code", "/file"}, extractReferences(results))
		assert.ElementsMatch(t, []string{
			"'code' is required when 'file' is not specified",
			"'file' is required when 'code' is not specified",
		}, extractMessages(results))
	})

	t.Run("code and file together are rejected", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			File:     "transform.js",
		}

		results := validateTransformationSpec("", "", "/tmp/spec.yaml", nil, spec)

		assert.ElementsMatch(t, []string{
			"/code",
			"/file",
			"/file",
		}, extractReferences(results))

		assert.ElementsMatch(t, []string{
			"'code' and 'file' cannot be specified together",
			"'file' and 'code' cannot be specified together",
			"path does not exist or is not accessible",
		}, extractMessages(results))
	})

	t.Run("whitespace-only and invalid test names are rejected", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			Tests: []specs.TransformationTest{
				{Name: "   "},
				{Name: "Suite@1"},
			},
		}

		results := validateTransformationSpec("", "", "/tmp/spec.yaml", nil, spec)

		assert.ElementsMatch(t, []string{"/tests/0/name", "/tests/1/name"}, extractReferences(results))
		assert.Contains(t, extractMessages(results), "'name' must not be blank or whitespace-only")
		assert.Contains(t, extractMessages(results), `'name' must match '^[A-Za-z0-9 _/\-]+$'`)
	})

	t.Run("absolute and parent traversal test paths are rejected", func(t *testing.T) {
		t.Parallel()

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			Tests: []specs.TransformationTest{
				{
					Name:   "Suite 1",
					Input:  "/tmp/fixtures/input",
					Output: "../fixtures/output",
				},
			},
		}

		results := validateTransformationSpec("", "", "/tmp/spec.yaml", nil, spec)

		assert.ElementsMatch(t, []string{"/tests/0/input", "/tests/0/output"}, extractReferences(results))
		assert.Contains(t, extractMessages(results), "path must be relative to the spec file directory")
		assert.Contains(t, extractMessages(results), "path must not contain '..' segments")
	})

	t.Run("file path must exist and be a regular file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		specPath := filepath.Join(tmpDir, "spec.yaml")
		dirPath := filepath.Join(tmpDir, "scripts")
		require.NoError(t, os.MkdirAll(dirPath, 0o755))

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			File:     "scripts",
		}

		results := validateTransformationSpec("", "", specPath, nil, spec)

		assert.ElementsMatch(t, []string{"/file", "/file"}, extractReferences(results))
		assert.Contains(t, extractMessages(results), "path must be a file")
		assert.Contains(t, extractMessages(results), "file extension must be '.js' for language 'javascript', got ''")
	})

	t.Run("input and output must be directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		specPath := filepath.Join(tmpDir, "spec.yaml")
		inputFilePath := filepath.Join(tmpDir, "input.json")
		outputFilePath := filepath.Join(tmpDir, "output.json")
		require.NoError(t, os.WriteFile(inputFilePath, []byte(`{"ok":true}`), 0o644))
		require.NoError(t, os.WriteFile(outputFilePath, []byte(`{"ok":true}`), 0o644))

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			Tests: []specs.TransformationTest{
				{
					Name:   "Suite 1",
					Input:  "input.json",
					Output: "output.json",
				},
			},
		}

		results := validateTransformationSpec("", "", specPath, nil, spec)

		assert.ElementsMatch(t, []string{"/tests/0/input", "/tests/0/output"}, extractReferences(results))
		assert.Contains(t, extractMessages(results), `path must be a directory`)
		assert.Contains(t, extractMessages(results), `path must be a directory`)
	})

	t.Run("json files must contain top-level objects or arrays", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		specPath := filepath.Join(tmpDir, "spec.yaml")
		inputDir := filepath.Join(tmpDir, "input")
		require.NoError(t, os.MkdirAll(inputDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "null.json"), []byte(`null`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "string.json"), []byte(`"hello"`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "number.json"), []byte(`123`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "boolean.json"), []byte(`true`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "invalid.json"), []byte(`{"missing":`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "object.json"), []byte(`{"ok":true}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(inputDir, "array.json"), []byte(`[{"ok":true}]`), 0o644))

		spec := specs.TransformationSpec{
			ID:       "trans-1",
			Name:     "Transformation",
			Language: "javascript",
			Code:     "export function transformEvent(event, metadata) { return event; }",
			Tests: []specs.TransformationTest{
				{
					Name:  "Suite 1",
					Input: "input",
				},
			},
		}

		results := validateTransformationSpec("", "", specPath, nil, spec)

		assert.ElementsMatch(t, []string{
			"/tests/0/input",
			"/tests/0/input",
			"/tests/0/input",
			"/tests/0/input",
			"/tests/0/input",
		}, extractReferences(results))
		assert.ElementsMatch(t, []string{
			"file: null.json must contain valid object or array",
			"file: string.json must contain valid object or array",
			"file: number.json must contain valid object or array",
			"file: boolean.json must contain valid object or array",
			"file: invalid.json must contain valid object or array",
		}, extractMessages(results))
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
				fileName:          "transform.py",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.js' for language 'javascript', got '.py'",
			},
			{
				name:              "python with javascript extension",
				language:          "python",
				fileName:          "transform.js",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.py' for language 'python', got '.js'",
			},
			{
				name:              "javascript with wrong extension",
				language:          "javascript",
				fileName:          "transform.txt",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.js' for language 'javascript', got '.txt'",
			},
			{
				name:              "python with no extension",
				language:          "python",
				fileName:          "transform",
				expectedReference: "/file",
				expectedMessage:   "file extension must be '.py' for language 'python', got ''",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				tmpDir := t.TempDir()
				specPath := filepath.Join(tmpDir, "spec.yaml")
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, tt.fileName), []byte("code"), 0o644))

				spec := specs.TransformationSpec{
					ID:       "trans-1",
					Name:     "Transformation",
					Language: tt.language,
					File:     tt.fileName,
				}

				results := validateTransformationSpec("", "", specPath, nil, spec)

				assert.Contains(t, extractReferences(results), tt.expectedReference)
				assert.Contains(t, extractMessages(results), tt.expectedMessage)
			})
		}
	})
}

func TestJSONValidObjectOrArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "object", input: `{"ok":true}`, expected: true},
		{name: "array", input: `[{"ok":true}]`, expected: true},
		{name: "null", input: `null`, expected: false},
		{name: "string", input: `"hello"`, expected: false},
		{name: "number", input: `123`, expected: false},
		{name: "boolean", input: `true`, expected: false},
		{name: "invalid json", input: `{"missing":`, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, jsonValidObjectOrArray([]byte(tt.input)))
		})
	}
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
