package testorchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

var JsonExt = ".json"

// ResolveTestDefinitions resolves test cases for a transformation.
func ResolveTestDefinitions(transformation *model.TransformationResource) ([]*transformations.TestDefinition, error) {
	// If no tests defined, use defaults with warning
	if len(transformation.Tests) == 0 {
		testLogger.Warn("No test suites defined for transformation, using default events", "transformationID", transformation.ID)
		return defaultTestDefinitions()
	}

	// Build test definitions from all suites
	var allTestDefs []*transformations.TestDefinition
	for _, suite := range transformation.Tests {
		testDefs, err := buildTestDefinitionsForSuite(suite)
		if err != nil {
			return nil, fmt.Errorf("building test cases for suite %s: %w", suite.Name, err)
		}
		allTestDefs = append(allTestDefs, testDefs...)
	}

	// If no test definitions found, use defaults with warning
	if len(allTestDefs) == 0 {
		testLogger.Warn("No test cases found, using default events", "transformationID", transformation.ID)
		return defaultTestDefinitions()
	}

	return allTestDefs, nil
}

// buildTestDefinitionsForSuite builds test cases for a single suite by reading
// input and output files directly from the configured directories.
func buildTestDefinitionsForSuite(suite specs.TransformationTest) ([]*transformations.TestDefinition, error) {
	// Resolve relative paths against SpecDir (set during spec loading)
	inputDir := resolveDir(suite.SpecDir, suite.Input)
	outputDir := resolveDir(suite.SpecDir, suite.Output)

	inputFiles, err := listJSONFiles(inputDir)
	if err != nil {
		return nil, fmt.Errorf("listing input files: %w", err)
	}

	if len(inputFiles) == 0 {
		testLogger.Debug("No input files found for suite", "suite", suite.Name)
		return nil, nil
	}

	outputFiles, err := listJSONFiles(outputDir)
	if err != nil {
		return nil, fmt.Errorf("listing output files: %w", err)
	}

	var testDefs []*transformations.TestDefinition

	for filename, fullPath := range inputFiles {
		inputEvents, err := parseJSONFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("parsing input file %s: %w", filename, err)
		}

		var expectedOutput []any
		if outputPath, exists := outputFiles[filename]; exists {
			expectedOutput, err = parseJSONFile(outputPath)
			if err != nil {
				return nil, fmt.Errorf("parsing output file %s: %w", filename, err)
			}
		}

		testDef := &transformations.TestDefinition{
			Name:           fmt.Sprintf("%s (%s/%s)", suite.Name, suite.Input, filename),
			Input:          inputEvents,
			ExpectedOutput: expectedOutput,
		}
		testDefs = append(testDefs, testDef)
	}

	return testDefs, nil
}

func defaultTestDefinitions() ([]*transformations.TestDefinition, error) {
	defaultEvents := GetDefaultEvents()

	var allEvents []any
	for _, eventData := range defaultEvents {
		allEvents = append(allEvents, eventData)
	}

	testDef := &transformations.TestDefinition{
		Name:           "default-events",
		Input:          allEvents,
		ExpectedOutput: nil,
	}

	return []*transformations.TestDefinition{testDef}, nil
}

func parseJSONFile(path string) ([]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var events []any
	if err := json.Unmarshal(data, &events); err != nil {
		// Try parsing as single event (not an array)
		var singleEvent any
		if err2 := json.Unmarshal(data, &singleEvent); err2 != nil {
			return nil, fmt.Errorf("invalid JSON in file %s: %w", filepath.Base(path), err)
		}
		events = []any{singleEvent}
	}

	return events, nil
}

// listJSONFiles lists all .json files in a directory
// Returns a map of filename (base name) to full path
func listJSONFiles(dir string) (map[string]string, error) {
	if !dirExists(dir) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), JsonExt) {
			files[name] = filepath.Join(dir, name)
		}
	}

	return files, nil
}

func resolveDir(baseDir, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}

func dirExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
