package testorchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

func TestResolveTestDefinitions(t *testing.T) {
	t.Run("no tests defined - returns default events", func(t *testing.T) {
		transformation := &model.TransformationResource{
			ID:    "trans-1",
			Tests: []specs.TransformationTest{},
		}

		result, err := ResolveTestDefinitions(transformation)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "default-events", result[0].Name)
		assert.NotEmpty(t, result[0].Input)
		assert.Nil(t, result[0].ExpectedOutput)
	})

	t.Run("suite with no resolvable input files - returns default events", func(t *testing.T) {
		tmpDir := t.TempDir()

		transformation := &model.TransformationResource{
			ID: "trans-1",
			Tests: []specs.TransformationTest{
				{
					Name:    "Empty Suite",
					SpecDir: tmpDir,
					Input:   filepath.Join(tmpDir, "nonexistent"),
				},
			},
		}

		result, err := ResolveTestDefinitions(transformation)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "default-events", result[0].Name)
	})

	t.Run("suite with input files only", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		require.NoError(t, os.MkdirAll(inputDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(inputDir, "event.json"),
			[]byte(`[{"type":"track","event":"Clicked"}]`),
			0644,
		))

		transformation := &model.TransformationResource{
			ID: "trans-1",
			Tests: []specs.TransformationTest{
				{
					Name:    "My Suite",
					SpecDir: tmpDir,
					Input:   inputDir,
				},
			},
		}

		result, err := ResolveTestDefinitions(transformation)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "My Suite/event", result[0].Name)
		require.Len(t, result[0].Input, 1)
		assert.Nil(t, result[0].ExpectedOutput)
	})

	t.Run("suite with matching input and output files", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		outputDir := filepath.Join(tmpDir, "output")
		require.NoError(t, os.MkdirAll(inputDir, 0755))
		require.NoError(t, os.MkdirAll(outputDir, 0755))

		require.NoError(t, os.WriteFile(
			filepath.Join(inputDir, "event.json"),
			[]byte(`[{"type":"track"}]`),
			0644,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(outputDir, "event.json"),
			[]byte(`[{"type":"track","processed":true}]`),
			0644,
		))

		transformation := &model.TransformationResource{
			ID: "trans-1",
			Tests: []specs.TransformationTest{
				{
					Name:    "Suite",
					SpecDir: tmpDir,
					Input:   inputDir,
					Output:  outputDir,
				},
			},
		}

		result, err := ResolveTestDefinitions(transformation)

		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "Suite/event", result[0].Name)
		assert.NotNil(t, result[0].ExpectedOutput)
		assert.Len(t, result[0].ExpectedOutput, 1)
	})

	t.Run("multiple suites produce separate test definitions", func(t *testing.T) {
		tmpDir := t.TempDir()
		suite1 := filepath.Join(tmpDir, "suite1")
		suite2 := filepath.Join(tmpDir, "suite2")
		require.NoError(t, os.MkdirAll(suite1, 0755))
		require.NoError(t, os.MkdirAll(suite2, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(suite1, "a.json"), []byte(`[{}]`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(suite2, "b.json"), []byte(`[{}]`), 0644))

		transformation := &model.TransformationResource{
			ID: "trans-1",
			Tests: []specs.TransformationTest{
				{Name: "Suite One", SpecDir: tmpDir, Input: suite1},
				{Name: "Suite Two", SpecDir: tmpDir, Input: suite2},
			},
		}

		result, err := ResolveTestDefinitions(transformation)

		require.NoError(t, err)
		require.Len(t, result, 2)

		names := []string{result[0].Name, result[1].Name}
		assert.Contains(t, names, "Suite One/a")
		assert.Contains(t, names, "Suite Two/b")
	})

	t.Run("transformation-specific input overrides common input", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		specificDir := filepath.Join(inputDir, "trans-1", "input")
		require.NoError(t, os.MkdirAll(inputDir, 0755))
		require.NoError(t, os.MkdirAll(specificDir, 0755))

		// Common file and a file to be overridden
		require.NoError(t, os.WriteFile(
			filepath.Join(inputDir, "common.json"),
			[]byte(`[{"source":"common"}]`),
			0644,
		))
		require.NoError(t, os.WriteFile(
			filepath.Join(inputDir, "shared.json"),
			[]byte(`[{"source":"common-shared"}]`),
			0644,
		))
		// Specific version overrides the shared one
		require.NoError(t, os.WriteFile(
			filepath.Join(specificDir, "shared.json"),
			[]byte(`[{"source":"specific-shared"}]`),
			0644,
		))

		transformation := &model.TransformationResource{
			ID: "trans-1",
			Tests: []specs.TransformationTest{
				{Name: "Suite", SpecDir: tmpDir, Input: inputDir},
			},
		}

		result, err := ResolveTestDefinitions(transformation)

		require.NoError(t, err)
		require.Len(t, result, 2)

		byName := make(map[string]*transformations.TestDefinition, len(result))
		for _, r := range result {
			byName[r.Name] = r
		}

		require.Contains(t, byName, "Suite/common")
		require.Contains(t, byName, "Suite/shared")

		sharedInput := byName["Suite/shared"].Input[0].(map[string]any)
		assert.Equal(t, "specific-shared", sharedInput["source"])
	})

	t.Run("invalid JSON in input file returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		require.NoError(t, os.MkdirAll(inputDir, 0755))
		require.NoError(t, os.WriteFile(
			filepath.Join(inputDir, "bad.json"),
			[]byte(`{not valid json`),
			0644,
		))

		transformation := &model.TransformationResource{
			ID: "trans-1",
			Tests: []specs.TransformationTest{
				{Name: "Suite", SpecDir: tmpDir, Input: inputDir},
			},
		}

		_, err := ResolveTestDefinitions(transformation)

		require.Error(t, err)
	})
}

func TestParseJSONFile(t *testing.T) {
	t.Run("parses array of events", func(t *testing.T) {
		f := writeTempJSON(t, `[{"type":"track"},{"type":"identify"}]`)

		events, err := parseJSONFile(f)

		require.NoError(t, err)
		assert.Len(t, events, 2)
	})

	t.Run("wraps single object in array", func(t *testing.T) {
		f := writeTempJSON(t, `{"type":"track"}`)

		events, err := parseJSONFile(f)

		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, "track", events[0].(map[string]any)["type"])
	})

	t.Run("empty array returns empty slice", func(t *testing.T) {
		f := writeTempJSON(t, `[]`)

		events, err := parseJSONFile(f)

		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		f := writeTempJSON(t, `{bad`)

		_, err := parseJSONFile(f)

		require.Error(t, err)
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, err := parseJSONFile("/no/such/file.json")

		require.Error(t, err)
	})
}

func TestMergeInputFiles(t *testing.T) {
	t.Run("returns common files when specific dir absent", func(t *testing.T) {
		tmpDir := t.TempDir()
		commonDir := filepath.Join(tmpDir, "common")
		require.NoError(t, os.MkdirAll(commonDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(commonDir, "a.json"), []byte(`[]`), 0644))

		result, err := mergeInputFiles(commonDir, filepath.Join(tmpDir, "missing"))

		require.NoError(t, err)
		assert.Contains(t, result, "a.json")
	})

	t.Run("returns specific files when common dir absent", func(t *testing.T) {
		tmpDir := t.TempDir()
		specificDir := filepath.Join(tmpDir, "specific")
		require.NoError(t, os.MkdirAll(specificDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(specificDir, "b.json"), []byte(`[]`), 0644))

		result, err := mergeInputFiles(filepath.Join(tmpDir, "missing"), specificDir)

		require.NoError(t, err)
		assert.Contains(t, result, "b.json")
	})

	t.Run("specific file path wins over common for same filename", func(t *testing.T) {
		tmpDir := t.TempDir()
		commonDir := filepath.Join(tmpDir, "common")
		specificDir := filepath.Join(tmpDir, "specific")
		require.NoError(t, os.MkdirAll(commonDir, 0755))
		require.NoError(t, os.MkdirAll(specificDir, 0755))

		commonFile := filepath.Join(commonDir, "event.json")
		specificFile := filepath.Join(specificDir, "event.json")
		require.NoError(t, os.WriteFile(commonFile, []byte(`[]`), 0644))
		require.NoError(t, os.WriteFile(specificFile, []byte(`[]`), 0644))

		result, err := mergeInputFiles(commonDir, specificDir)

		require.NoError(t, err)
		assert.Equal(t, specificFile, result["event.json"])
	})

	t.Run("both dirs absent returns empty map", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := mergeInputFiles(
			filepath.Join(tmpDir, "a"),
			filepath.Join(tmpDir, "b"),
		)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestListJSONFiles(t *testing.T) {
	t.Run("returns json files by base name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.json"), []byte(`[]`), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("text"), 0644))

		result, err := listJSONFiles(tmpDir)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "a.json")
		assert.Equal(t, filepath.Join(tmpDir, "a.json"), result["a.json"])
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir.json"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "real.json"), []byte(`[]`), 0644))

		result, err := listJSONFiles(tmpDir)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Contains(t, result, "real.json")
	})

	t.Run("non-existent directory returns nil without error", func(t *testing.T) {
		result, err := listJSONFiles("/no/such/dir")

		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("empty directory returns empty map", func(t *testing.T) {
		tmpDir := t.TempDir()

		result, err := listJSONFiles(tmpDir)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestDirExists(t *testing.T) {
	t.Run("returns true for existing directory", func(t *testing.T) {
		assert.True(t, dirExists(t.TempDir()))
	})

	t.Run("returns false for missing path", func(t *testing.T) {
		assert.False(t, dirExists("/no/such/path"))
	})

	t.Run("returns false for empty string", func(t *testing.T) {
		assert.False(t, dirExists(""))
	})

	t.Run("returns false for a file path", func(t *testing.T) {
		tmpDir := t.TempDir()
		f := filepath.Join(tmpDir, "file.txt")
		require.NoError(t, os.WriteFile(f, []byte("x"), 0644))

		assert.False(t, dirExists(f))
	})
}

func TestResolveDir(t *testing.T) {
	t.Run("empty path returns empty string", func(t *testing.T) {
		assert.Equal(t, "", resolveDir("/base", ""))
	})

	t.Run("absolute path is returned as-is", func(t *testing.T) {
		assert.Equal(t, "/abs/path", resolveDir("/base", "/abs/path"))
	})

	t.Run("relative path is joined with base", func(t *testing.T) {
		assert.Equal(t, "/base/relative", resolveDir("/base", "relative"))
	})
}

// writeTempJSON writes content to a temporary .json file and returns its path.
func writeTempJSON(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(f, []byte(content), 0644))
	return f
}
