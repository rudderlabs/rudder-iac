package testorchestrator

import (
	"context"
	"fmt"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
)

// TestDefinition represents a single test case in the batch test request
type TestDefinition struct {
	Name          string `json:"name"`
	Input         []any  `json:"input"`         // Array of event payloads
	ExpectedOutput []any  `json:"output,omitempty"` // Expected transformation output
}

// MultiTransformationTestInput represents the transformation to test with inline code
type MultiTransformationTestInput struct {
	Code        string           `json:"code"`
	Language    string           `json:"language"`
	Tests       []TestDefinition `json:"tests"`
	LibraryTags []string         `json:"libraryTags"` // Array of library version IDs
}

// TransformationLibraryInput represents a library referenced by versionID
type TransformationLibraryInput struct {
	VersionID string `json:"versionId"`
}

// BatchTestRequest is the request payload for the batch test API
type BatchTestRequest struct {
	Transformation MultiTransformationTestInput  `json:"transformation"`
	Libraries      []TransformationLibraryInput `json:"libraries"`
}

// TestRunStatus represents the status of a test execution
type TestRunStatus string

const (
	TestRunStatusPass  TestRunStatus = "pass"
	TestRunStatusFail  TestRunStatus = "fail"
	TestRunStatusError TestRunStatus = "error"
)

// TestError represents an error that occurred during test execution
type TestError struct {
	Message string `json:"message"`
}

// TestResult represents the result of a single test case execution
type TestResult struct {
	Name   string        `json:"name"`
	Status TestRunStatus `json:"status"`
	Output []any         `json:"output,omitempty"` // Actual transformation output
	Error  *TestError    `json:"error,omitempty"`  // Error details if status is error
}

// TestSuiteRunResult represents the results for all tests in a suite
type TestSuiteRunResult struct {
	Tests []TestResult `json:"tests"`
}

// TransformationTestResult represents the complete test results for a transformation
type TransformationTestResult struct {
	TransformationID string             `json:"transformationId"`
	Result           TestSuiteRunResult `json:"result"`
}

// APIClient handles communication with the batch test API
type APIClient struct {
	store transformations.TransformationStore
}

// NewAPIClient creates a new API client for running tests
func NewAPIClient(store transformations.TransformationStore) *APIClient {
	return &APIClient{
		store: store,
	}
}

// RunTests executes a batch test request and returns the results
// This method will be integrated with the actual BatchTest API in a later PR
func (c *APIClient) RunTests(ctx context.Context, req *BatchTestRequest) (*TransformationTestResult, error) {
	// TODO: Call actual BatchTest API once it's available in transformations.TransformationStore
	// For now, return a placeholder indicating this will be implemented
	return nil, fmt.Errorf("batch test API integration pending - will be completed when BatchTest is available")
}
