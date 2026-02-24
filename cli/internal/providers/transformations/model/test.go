package model

import (
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

// TransformationTestWithDefinitions combines test results with their original definitions
type TransformationTestWithDefinitions struct {
	Result      *transformations.TransformationTestResult
	Definitions []*transformations.TestDefinition
}

// TestResults contains the results of all test executions with their definitions
type TestResults struct {
	Pass            bool
	Message         string
	Libraries       []transformations.LibraryTestResult
	Transformations []*TransformationTestWithDefinitions
}
