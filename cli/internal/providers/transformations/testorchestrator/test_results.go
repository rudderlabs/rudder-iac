package testorchestrator

import (
	"github.com/samber/lo"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

type RunStatus string

const (
	RunStatusExecuted    RunStatus = "executed"
	RunStatusNoResources RunStatus = "no_resources"
)

// TransformationTestWithDefinitions combines test results with their original definitions
type TransformationTestWithDefinitions struct {
	Result      *transformations.TransformationTestResult `json:"result"`
	Definitions []*transformations.TestDefinition         `json:"-"`
}

// TestResults contains the results of all test executions with their definitions
type TestResults struct {
	Status          RunStatus                            `json:"status"`
	Libraries       []transformations.LibraryTestResult  `json:"libraries,omitempty"`
	Transformations []*TransformationTestWithDefinitions `json:"transformations,omitempty"`
}

// HasFailures computes whether any tests failed or errored
func (r *TestResults) HasFailures() bool {
	if lo.ContainsBy(r.Libraries, func(lib transformations.LibraryTestResult) bool {
		return !lib.Pass
	}) {
		return true
	}

	return lo.ContainsBy(r.Transformations, func(tr *TransformationTestWithDefinitions) bool {
		return lo.ContainsBy(tr.Result.TestSuiteResult.Results, func(res transformations.TestResult) bool {
			return res.Status == transformations.TestRunStatusFail || res.Status == transformations.TestRunStatusError
		})
	})
}

// DefaultSuiteTransformationNames returns the names of transformations that used the default test suite
func (r *TestResults) DefaultSuiteTransformationNames() []string {
	const defaultTestSuiteName = "default-events"

	return lo.FilterMap(r.Transformations, func(tr *TransformationTestWithDefinitions, _ int) (string, bool) {
		if len(tr.Definitions) == 1 && tr.Definitions[0].Name == defaultTestSuiteName {
			return tr.Result.Name, true
		}
		return "", false
	})
}
