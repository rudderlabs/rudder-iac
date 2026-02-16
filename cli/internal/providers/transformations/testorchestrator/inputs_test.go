package testorchestrator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

func TestResolveTestCases(t *testing.T) {
	t.Run("no tests defined - uses defaults", func(t *testing.T) {
		transformation := &model.TransformationResource{
			ID:    "test-trans",
			Tests: []specs.TransformationTest{},
		}

		testCases, err := ResolveTestCases(transformation)

		require.NoError(t, err)
		require.Len(t, testCases, 1)
		assert.Equal(t, "default/all-events", testCases[0].Name)
		assert.NotEmpty(t, testCases[0].InputEvents)
		assert.Nil(t, testCases[0].ExpectedOutput)
	})

	t.Run("test suite with input files", func(t *testing.T) {
		// Create temp directory structure
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		err := os.MkdirAll(inputDir, 0755)
		require.NoError(t, err)

		// Create input file
		inputFile := filepath.Join(inputDir, "test1.json")
		inputData := `[{"type":"track","event":"Test Event"}]`
		err = os.WriteFile(inputFile, []byte(inputData), 0644)
		require.NoError(t, err)

		transformation := &model.TransformationResource{
			ID: "test-trans",
			Tests: []specs.TransformationTest{
				{
					Name:    "Test Suite",
					SpecDir: tmpDir,
					Input:   inputDir,
					Output:  filepath.Join(tmpDir, "output"),
				},
			},
		}

		testCases, err := ResolveTestCases(transformation)

		require.NoError(t, err)
		require.Len(t, testCases, 1)
		assert.Equal(t, "test-suite/test1", testCases[0].Name)
		assert.Len(t, testCases[0].InputEvents, 1)
	})

	t.Run("test suite with input and output files", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		outputDir := filepath.Join(tmpDir, "output")
		err := os.MkdirAll(inputDir, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(outputDir, 0755)
		require.NoError(t, err)

		// Create input and output files
		inputData := `[{"type":"track","event":"Test Event"}]`
		err = os.WriteFile(filepath.Join(inputDir, "test1.json"), []byte(inputData), 0644)
		require.NoError(t, err)

		outputData := `[{"type":"track","event":"Modified Event"}]`
		err = os.WriteFile(filepath.Join(outputDir, "test1.json"), []byte(outputData), 0644)
		require.NoError(t, err)

		transformation := &model.TransformationResource{
			ID: "test-trans",
			Tests: []specs.TransformationTest{
				{
					Name:    "Test Suite",
					SpecDir: tmpDir,
					Input:   inputDir,
					Output:  outputDir,
				},
			},
		}

		testCases, err := ResolveTestCases(transformation)

		require.NoError(t, err)
		require.Len(t, testCases, 1)
		assert.Equal(t, "test-suite/test1", testCases[0].Name)
		assert.Len(t, testCases[0].InputEvents, 1)
		assert.NotNil(t, testCases[0].ExpectedOutput)
		assert.Len(t, testCases[0].ExpectedOutput, 1)
	})

	t.Run("test suite with missing input directory - falls back to defaults", func(t *testing.T) {
		tmpDir := t.TempDir()

		transformation := &model.TransformationResource{
			ID: "test-trans",
			Tests: []specs.TransformationTest{
				{
					Name:    "Test Suite",
					SpecDir: tmpDir,
					Input:   filepath.Join(tmpDir, "nonexistent"),
				},
			},
		}

		testCases, err := ResolveTestCases(transformation)

		require.NoError(t, err)
		require.Len(t, testCases, 1)
		assert.Equal(t, "default/all-events", testCases[0].Name)
	})

	t.Run("multiple test suites", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create first suite
		suite1Dir := filepath.Join(tmpDir, "suite1")
		err := os.MkdirAll(suite1Dir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(suite1Dir, "test1.json"), []byte(`[{"type":"track"}]`), 0644)
		require.NoError(t, err)

		// Create second suite
		suite2Dir := filepath.Join(tmpDir, "suite2")
		err = os.MkdirAll(suite2Dir, 0755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(suite2Dir, "test2.json"), []byte(`[{"type":"identify"}]`), 0644)
		require.NoError(t, err)

		transformation := &model.TransformationResource{
			ID: "test-trans",
			Tests: []specs.TransformationTest{
				{
					Name:    "Suite 1",
					SpecDir: tmpDir,
					Input:   suite1Dir,
				},
				{
					Name:    "Suite 2",
					SpecDir: tmpDir,
					Input:   suite2Dir,
				},
			},
		}

		testCases, err := ResolveTestCases(transformation)

		require.NoError(t, err)
		require.Len(t, testCases, 2)
		assert.Equal(t, "suite-1/test1", testCases[0].Name)
		assert.Equal(t, "suite-2/test2", testCases[1].Name)
	})

	t.Run("common and specific inputs - specific overrides common", func(t *testing.T) {
		tmpDir := t.TempDir()
		testsDir := filepath.Join(tmpDir, "tests", "input")
		specificDir := filepath.Join(tmpDir, "specific")

		err := os.MkdirAll(testsDir, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(specificDir, 0755)
		require.NoError(t, err)

		// Create common file
		err = os.WriteFile(filepath.Join(testsDir, "common.json"), []byte(`[{"source":"common"}]`), 0644)
		require.NoError(t, err)

		// Create specific file that overrides
		err = os.WriteFile(filepath.Join(testsDir, "override.json"), []byte(`[{"source":"common-override"}]`), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(specificDir, "override.json"), []byte(`[{"source":"specific"}]`), 0644)
		require.NoError(t, err)

		// Create specific-only file
		err = os.WriteFile(filepath.Join(specificDir, "specific-only.json"), []byte(`[{"source":"specific"}]`), 0644)
		require.NoError(t, err)

		transformation := &model.TransformationResource{
			ID: "test-trans",
			Tests: []specs.TransformationTest{
				{
					Name:    "Test Suite",
					SpecDir: tmpDir,
					Input:   specificDir,
				},
			},
		}

		testCases, err := ResolveTestCases(transformation)

		require.NoError(t, err)
		require.Len(t, testCases, 3)

		// Verify common file is included
		var foundCommon bool
		for _, tc := range testCases {
			if tc.Name == "test-suite/common" {
				foundCommon = true
				event := tc.InputEvents[0].(map[string]any)
				assert.Equal(t, "common", event["source"])
			}
		}
		assert.True(t, foundCommon)

		// Verify override file uses specific version
		var foundOverride bool
		for _, tc := range testCases {
			if tc.Name == "test-suite/override" {
				foundOverride = true
				event := tc.InputEvents[0].(map[string]any)
				assert.Equal(t, "specific", event["source"])
			}
		}
		assert.True(t, foundOverride)

		// Verify specific-only file is included
		var foundSpecificOnly bool
		for _, tc := range testCases {
			if tc.Name == "test-suite/specific-only" {
				foundSpecificOnly = true
			}
		}
		assert.True(t, foundSpecificOnly)
	})

	t.Run("invalid JSON in input file - error", func(t *testing.T) {
		tmpDir := t.TempDir()
		inputDir := filepath.Join(tmpDir, "input")
		err := os.MkdirAll(inputDir, 0755)
		require.NoError(t, err)

		// Create invalid JSON file
		err = os.WriteFile(filepath.Join(inputDir, "invalid.json"), []byte(`{invalid json`), 0644)
		require.NoError(t, err)

		transformation := &model.TransformationResource{
			ID: "test-trans",
			Tests: []specs.TransformationTest{
				{
					Name:    "Test Suite",
					SpecDir: tmpDir,
					Input:   inputDir,
				},
			},
		}

		testCases, err := ResolveTestCases(transformation)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
		assert.Nil(t, testCases)
	})
}

func TestParseJSONFile(t *testing.T) {
	t.Run("array of events", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "events.json")
		content := `[{"type":"track"},{"type":"identify"}]`
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		events, err := parseJSONFile(filePath)

		require.NoError(t, err)
		assert.Len(t, events, 2)
	})

	t.Run("single event object - wrapped in array", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "event.json")
		content := `{"type":"track","event":"Test"}`
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		events, err := parseJSONFile(filePath)

		require.NoError(t, err)
		assert.Len(t, events, 1)
		eventMap := events[0].(map[string]any)
		assert.Equal(t, "track", eventMap["type"])
	})

	t.Run("invalid JSON - error", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "invalid.json")
		content := `{invalid`
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		events, err := parseJSONFile(filePath)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
		assert.Nil(t, events)
	})

	t.Run("file not found - error", func(t *testing.T) {
		events, err := parseJSONFile("/nonexistent/file.json")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading file")
		assert.Nil(t, events)
	})

	t.Run("empty array", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "empty.json")
		content := `[]`
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		events, err := parseJSONFile(filePath)

		require.NoError(t, err)
		assert.Empty(t, events)
	})
}

func TestMergeInputFiles(t *testing.T) {
	t.Run("common files only", func(t *testing.T) {
		tmpDir := t.TempDir()
		commonDir := filepath.Join(tmpDir, "common")
		err := os.MkdirAll(commonDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(commonDir, "file1.json"), []byte(`[]`), 0644)
		require.NoError(t, err)

		merged, err := mergeInputFiles(commonDir, filepath.Join(tmpDir, "nonexistent"))

		require.NoError(t, err)
		assert.Len(t, merged, 1)
		assert.Contains(t, merged, "file1.json")
	})

	t.Run("specific files only", func(t *testing.T) {
		tmpDir := t.TempDir()
		specificDir := filepath.Join(tmpDir, "specific")
		err := os.MkdirAll(specificDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(specificDir, "file1.json"), []byte(`[]`), 0644)
		require.NoError(t, err)

		merged, err := mergeInputFiles(filepath.Join(tmpDir, "nonexistent"), specificDir)

		require.NoError(t, err)
		assert.Len(t, merged, 1)
		assert.Contains(t, merged, "file1.json")
	})

	t.Run("specific overrides common", func(t *testing.T) {
		tmpDir := t.TempDir()
		commonDir := filepath.Join(tmpDir, "common")
		specificDir := filepath.Join(tmpDir, "specific")
		err := os.MkdirAll(commonDir, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(specificDir, 0755)
		require.NoError(t, err)

		commonFile := filepath.Join(commonDir, "shared.json")
		specificFile := filepath.Join(specificDir, "shared.json")
		err = os.WriteFile(commonFile, []byte(`[]`), 0644)
		require.NoError(t, err)
		err = os.WriteFile(specificFile, []byte(`[]`), 0644)
		require.NoError(t, err)

		merged, err := mergeInputFiles(commonDir, specificDir)

		require.NoError(t, err)
		assert.Len(t, merged, 1)
		assert.Equal(t, specificFile, merged["shared.json"])
	})

	t.Run("both directories empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		commonDir := filepath.Join(tmpDir, "common")
		specificDir := filepath.Join(tmpDir, "specific")
		err := os.MkdirAll(commonDir, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(specificDir, 0755)
		require.NoError(t, err)

		merged, err := mergeInputFiles(commonDir, specificDir)

		require.NoError(t, err)
		assert.Empty(t, merged)
	})

	t.Run("neither directory exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		merged, err := mergeInputFiles(
			filepath.Join(tmpDir, "nonexistent1"),
			filepath.Join(tmpDir, "nonexistent2"),
		)

		require.NoError(t, err)
		assert.Empty(t, merged)
	})
}

func TestNormalizeSuiteName(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", "default"},
		{"Simple", "simple"},
		{"Test Suite", "test-suite"},
		{"Test_Suite", "test-suite"},
		{"Test Suite_With_Multiple   Spaces", "test-suite-with-multiple---spaces"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizeSuiteName(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDirExists(t *testing.T) {
	t.Run("existing directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.True(t, dirExists(tmpDir))
	})

	t.Run("non-existing directory", func(t *testing.T) {
		assert.False(t, dirExists("/nonexistent/path"))
	})

	t.Run("empty path", func(t *testing.T) {
		assert.False(t, dirExists(""))
	})

	t.Run("file not directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "file.txt")
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)

		assert.False(t, dirExists(filePath))
	})
}

func TestHasJSONFiles(t *testing.T) {
	t.Run("directory with JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte(`[]`), 0644)
		require.NoError(t, err)

		assert.True(t, hasJSONFiles(tmpDir))
	})

	t.Run("directory without JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
		require.NoError(t, err)

		assert.False(t, hasJSONFiles(tmpDir))
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		assert.False(t, hasJSONFiles(tmpDir))
	})

	t.Run("non-existing directory", func(t *testing.T) {
		assert.False(t, hasJSONFiles("/nonexistent/path"))
	})
}

func TestListJSONFiles(t *testing.T) {
	t.Run("directory with multiple JSON files", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := os.WriteFile(filepath.Join(tmpDir, "test1.json"), []byte(`[]`), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "test2.JSON"), []byte(`[]`), 0644)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)
		require.NoError(t, err)

		files, err := listJSONFiles(tmpDir)

		require.NoError(t, err)
		assert.Len(t, files, 2)
		assert.Contains(t, files, "test1.json")
		assert.Contains(t, files, "test2.JSON")
		assert.NotContains(t, files, "test.txt")
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		files, err := listJSONFiles(tmpDir)

		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("non-existing directory", func(t *testing.T) {
		files, err := listJSONFiles("/nonexistent/path")

		require.NoError(t, err)
		assert.Nil(t, files)
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir.json")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tmpDir, "file.json"), []byte(`[]`), 0644)
		require.NoError(t, err)

		files, err := listJSONFiles(tmpDir)

		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Contains(t, files, "file.json")
	})
}

func TestDefaultTestCases(t *testing.T) {
	t.Run("creates single test case with all events", func(t *testing.T) {
		testCases, err := defaultTestCases()

		require.NoError(t, err)
		require.Len(t, testCases, 1)
		assert.Equal(t, "default/all-events", testCases[0].Name)
		assert.NotEmpty(t, testCases[0].InputEvents)
		assert.Nil(t, testCases[0].ExpectedOutput)

		// Verify it contains multiple events (Track, Identify, Page, Screen)
		assert.GreaterOrEqual(t, len(testCases[0].InputEvents), 4)
	})
}
