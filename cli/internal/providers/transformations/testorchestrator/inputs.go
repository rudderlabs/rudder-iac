package testorchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
)

var inputLog = logger.New("testorchestrator", logger.Attr{
	Key:   "component",
	Value: "inputs",
})

// TestCase represents a single test case with input events and expected output
type TestCase struct {
	Name           string // Relative path to filename excluding extension
	InputEvents    []any  // Array of input event payloads
	ExpectedOutput []any  // Expected output (nil if no output file)
}

// InputResolver resolves test cases from file system or falls back to defaults
type InputResolver struct{}

// NewInputResolver creates a new input resolver
func NewInputResolver() *InputResolver {
	return &InputResolver{}
}

// ResolveTestCases resolves test cases for a transformation.
// It merges common and transformation-specific inputs, falling back to defaults if needed.
func (r *InputResolver) ResolveTestCases(transformation *model.TransformationResource) ([]TestCase, error) {
	// If no tests defined, use defaults with warning
	if len(transformation.Tests) == 0 {
		inputLog.Warn("No test suites defined for transformation, using default events", "transformationID", transformation.ID)
		return r.buildDefaultTestCases()
	}

	// Check if any suite has input files
	hasInputFiles := false
	for _, suite := range transformation.Tests {
		inputDir := suite.Input
		if inputDir != "" && dirExists(inputDir) && hasJSONFiles(inputDir) {
			hasInputFiles = true
			break
		}
	}

	// If no input files found, use defaults with warning
	if !hasInputFiles {
		inputLog.Warn("No test input files found for transformation, using default events", "transformationID", transformation.ID)
		return r.buildDefaultTestCases()
	}

	// Build test cases from all suites
	var allTestCases []TestCase
	for _, suite := range transformation.Tests {
		testCases, err := r.buildTestCasesForSuite(suite, transformation.ID)
		if err != nil {
			return nil, fmt.Errorf("building test cases for suite %s: %w", suite.Name, err)
		}
		allTestCases = append(allTestCases, testCases...)
	}

	if len(allTestCases) == 0 {
		inputLog.Warn("No test cases found, using default events", "transformationID", transformation.ID)
		return r.buildDefaultTestCases()
	}

	return allTestCases, nil
}

// buildTestCasesForSuite builds test cases for a single suite
func (r *InputResolver) buildTestCasesForSuite(suite specs.TransformationTest, transformationID string) ([]TestCase, error) {
	// SpecDir is populated by handler during spec resolution (Phase 1)
	if suite.SpecDir == "" {
		return nil, fmt.Errorf("SpecDir not populated for test suite %s", suite.Name)
	}

	inputDir := suite.Input
	outputDir := suite.Output

	// If input directory doesn't exist, skip this suite
	if !dirExists(inputDir) {
		inputLog.Debug("Input directory does not exist", "suite", suite.Name, "dir", inputDir)
		return nil, nil
	}

	// Build merged input file map (common + specific)
	commonInputDir := filepath.Join(suite.SpecDir, "tests", "input")
	inputFiles := mergeInputFiles(commonInputDir, inputDir)

	if len(inputFiles) == 0 {
		inputLog.Debug("No input files found for suite", "suite", suite.Name)
		return nil, nil
	}

	// Load output files if directory exists
	outputFiles := make(map[string]string)
	if dirExists(outputDir) {
		files, err := listJSONFiles(outputDir)
		if err != nil {
			return nil, fmt.Errorf("listing output files: %w", err)
		}
		outputFiles = files
	}

	// Build test cases
	var testCases []TestCase
	suiteName := normalizeSuiteName(suite.Name)

	for filename, filepath := range inputFiles {
		// Parse input events
		inputEvents, err := parseJSONFile(filepath)
		if err != nil {
			return nil, fmt.Errorf("parsing input file %s: %w", filename, err)
		}

		// Parse expected output if exists
		var expectedOutput []any
		if outputPath, exists := outputFiles[filename]; exists {
			expectedOutput, err = parseJSONFile(outputPath)
			if err != nil {
				return nil, fmt.Errorf("parsing output file %s: %w", filename, err)
			}
		}

		// Create test case with relative path as name
		testName := strings.TrimSuffix(filename, ".json")
		if suiteName != "" && suiteName != "default" {
			testName = fmt.Sprintf("%s/%s", suiteName, testName)
		}

		testCase := TestCase{
			Name:           testName,
			InputEvents:    inputEvents,
			ExpectedOutput: expectedOutput,
		}
		testCases = append(testCases, testCase)
	}

	return testCases, nil
}

// mergeInputFiles merges common and specific input files.
// Specific files override common files with the same name.
// Returns a map of filename (base name) to full path.
func mergeInputFiles(commonDir, specificDir string) map[string]string {
	merged := make(map[string]string)

	// Load common files first
	if dirExists(commonDir) {
		commonFiles, err := listJSONFiles(commonDir)
		if err == nil {
			for name, path := range commonFiles {
				merged[name] = path
			}
		}
	}

	// Load specific files (overrides common)
	if dirExists(specificDir) {
		specificFiles, err := listJSONFiles(specificDir)
		if err == nil {
			for name, path := range specificFiles {
				merged[name] = path
			}
		}
	}

	return merged
}

// buildDefaultTestCases creates test cases from embedded default events
// All default events are included in a single test case's InputEvents array
func (r *InputResolver) buildDefaultTestCases() ([]TestCase, error) {
	defaultEvents := GetDefaultEvents()

	// Collect all events into a single array
	var allEvents []any
	for _, eventData := range defaultEvents {
		allEvents = append(allEvents, eventData)
	}

	// Create a single test case containing all default events
	testCase := TestCase{
		Name:           "default/all-events",
		InputEvents:    allEvents,
		ExpectedOutput: nil,
	}

	return []TestCase{testCase}, nil
}

// parseJSONFile reads and parses a JSON file as an array of events
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
			// Invalid JSON - fail immediately as per requirements
			return nil, fmt.Errorf("invalid JSON in file %s: %w", filepath.Base(path), err)
		}
		// Wrap single event in array
		events = []any{singleEvent}
	}

	return events, nil
}

// normalizeSuiteName converts a suite name to lowercase-with-dashes format
func normalizeSuiteName(name string) string {
	if name == "" {
		return "default"
	}
	// Replace spaces and underscores with dashes, convert to lowercase
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

// dirExists checks if a directory exists
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

// hasJSONFiles checks if a directory contains any .json files
func hasJSONFiles(dir string) bool {
	files, err := listJSONFiles(dir)
	if err != nil {
		return false
	}
	return len(files) > 0
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
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			files[name] = filepath.Join(dir, name)
		}
	}

	return files, nil
}
