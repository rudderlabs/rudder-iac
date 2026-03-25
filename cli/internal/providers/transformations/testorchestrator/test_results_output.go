package testorchestrator

import (
	"path/filepath"
	"strings"

	"github.com/samber/lo"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

const defaultEventsOutputFilename = "default_events.json"

// TestResultsOutput is the serialization struct for results.json.
type TestResultsOutput struct {
	Status          RunStatus                           `json:"status"`
	Libraries       []transformations.LibraryTestResult `json:"libraries,omitempty"`
	Transformations []TransformationTestOutput          `json:"transformations,omitempty"`
}

type TransformationTestOutput struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	VersionID       string                `json:"versionId"`
	Imports         []string              `json:"imports,omitempty"`
	Pass            bool                  `json:"pass"`
	TestSuiteResult TestSuiteResultOutput `json:"testResult"`
	Message         string                `json:"message,omitempty"`
}

type TestSuiteResultOutput struct {
	Status  transformations.TestRunStatus `json:"status"`
	Results []TestResultOutput            `json:"results"`
}

// TestResultOutput replaces TestResult for JSON output.
// Adds inputLocation/outputLocation, replaces actualOutput with actualOutputFile.
type TestResultOutput struct {
	ID               string                        `json:"id"`
	Name             string                        `json:"name"`
	Description      string                        `json:"description,omitempty"`
	Status           transformations.TestRunStatus `json:"status"`
	InputLocation    string                        `json:"inputLocation,omitempty"`
	OutputLocation   string                        `json:"outputLocation,omitempty"`
	ActualOutputFile string                        `json:"actualOutputFile,omitempty"`
	Errors           []transformations.TestError   `json:"errors,omitempty"`
}

// ActualOutputEntry pairs a relative file path (under test-results/) with actual output data
// to be written as a separate JSON file.
type ActualOutputEntry struct {
	RelPath      string
	ActualOutput []any
}

// ToOutput converts TestResults into the serialization-ready TestResultsOutput,
// also returning actual output entries to be written as separate files.
func (r *TestResults) ToOutput() (*TestResultsOutput, []ActualOutputEntry) {
	var (
		trOutputs     []TransformationTestOutput
		outputEntries []ActualOutputEntry
	)

	for _, tr := range r.Transformations {
		trOut, entries := buildTransformationOutput(tr)
		trOutputs = append(trOutputs, trOut)
		outputEntries = append(outputEntries, entries...)
	}

	return &TestResultsOutput{
		Status:          r.Status,
		Libraries:       r.Libraries,
		Transformations: trOutputs,
	}, outputEntries
}

func buildTransformationOutput(tr *TransformationTestWithDefinitions) (TransformationTestOutput, []ActualOutputEntry) {
	result := tr.Result
	defsByID := lo.SliceToMap(tr.Definitions, func(def *transformations.TestDefinition) (string, *transformations.TestDefinition) {
		return def.ID, def
	})

	var (
		resultOutputs []TestResultOutput
		entries       []ActualOutputEntry
	)

	for _, testResult := range result.TestSuiteResult.Results {
		def := defsByID[testResult.ID]

		var inputLocation, outputLocation, actualOutputFile string
		if def != nil {
			inputLocation = def.InputFile
			outputLocation = def.OutputFile

			if len(testResult.ActualOutput) > 0 {
				actualOutputFile = filepath.Join(
					sanitizePath(result.Name),
					sanitizePath(testResult.Name),
					def.Filename,
				)
				entries = append(entries, ActualOutputEntry{
					RelPath:      actualOutputFile,
					ActualOutput: testResult.ActualOutput,
				})
			}
		}

		resultOutputs = append(resultOutputs, TestResultOutput{
			ID:               testResult.ID,
			Name:             testResult.Name,
			Description:      testResult.Description,
			Status:           testResult.Status,
			InputLocation:    inputLocation,
			OutputLocation:   outputLocation,
			ActualOutputFile: actualOutputFile,
			Errors:           testResult.Errors,
		})
	}

	return TransformationTestOutput{
		ID:        result.ID,
		Name:      result.Name,
		VersionID: result.VersionID,
		Imports:   result.Imports,
		Pass:      result.Pass,
		TestSuiteResult: TestSuiteResultOutput{
			Status:  result.TestSuiteResult.Status,
			Results: resultOutputs,
		},
		Message: result.Message,
	}, entries
}

// sanitizePath replaces filesystem-unsafe characters with underscores.
func sanitizePath(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_",
		"*", "_", "?", "_", "\"", "_",
		"<", "_", ">", "_", "|", "_",
	)
	return strings.TrimSpace(replacer.Replace(s))
}
