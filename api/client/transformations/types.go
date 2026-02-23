package transformations

// Transformation represents a transformation resource from the API
type Transformation struct {
	ID          string   `json:"id"`
	VersionID   string   `json:"versionId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Code        string   `json:"code"`
	Language    string   `json:"language"`
	Imports     []string `json:"imports"`
	WorkspaceID string   `json:"workspaceId"`
	ExternalID  string   `json:"externalId,omitempty"`
}

// TransformationLibrary represents a transformation library resource from the API
type TransformationLibrary struct {
	ID          string `json:"id"`
	VersionID   string `json:"versionId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ImportName  string `json:"importName"`
	WorkspaceID string `json:"workspaceId"`
	ExternalID  string `json:"externalId,omitempty"`
}

// CreateTransformationRequest is the request body for creating a transformation
type CreateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ExternalID  string `json:"externalId"`
}

// UpdateTransformationRequest is the request body for updating a transformation
type UpdateTransformationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
}

// CreateLibraryRequest is the request body for creating a library
type CreateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
	ExternalID  string `json:"externalId"`
}

// UpdateLibraryRequest is the request body for updating a library
type UpdateLibraryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Code        string `json:"code"`
	Language    string `json:"language"`
}

// BatchPublishRequest is the request body for batch publishing transformations and libraries
type BatchPublishRequest struct {
	Transformations []BatchPublishTransformation `json:"transformations,omitempty"`
	Libraries       []BatchPublishLibrary        `json:"libraries,omitempty"`
}

// BatchPublishTransformation represents a transformation to publish
type BatchPublishTransformation struct {
	VersionID string           `json:"versionId"`
	TestSuite []TestDefinition `json:"testSuite,omitempty"`
}

// BatchPublishLibrary represents a library to publish
type BatchPublishLibrary struct {
	VersionID string `json:"versionId"`
}

type SetExternalIDRequest struct {
	ExternalID string `json:"externalId"`
}

// Batch Test API Types

// TestDefinition defines the structure of a test case
type TestDefinition struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Input          []any  `json:"input"`
	ExpectedOutput []any  `json:"expectedOutput,omitempty"`
}

// TransformationTestInputs represents transformation test input
type TransformationTestInput struct {
	VersionID string           `json:"versionId"`
	TestSuite []TestDefinition `json:"testSuite"`
}

// TransformationLibraryInput represents library input for transformation testing
type LibraryTestInput struct {
	VersionID string `json:"versionId"`
}

// BatchTestRequest is the request body for batch testing
type BatchTestRequest struct {
	Transformations []TransformationTestInput `json:"transformations,omitempty"`
	Libraries       []LibraryTestInput        `json:"libraries,omitempty"`
}

// TestRunStatus represents possible test run statuses
type TestRunStatus string

const (
	TestRunStatusPass  TestRunStatus = "pass"
	TestRunStatusFail  TestRunStatus = "fail"
	TestRunStatusError TestRunStatus = "error"
)

// TestError represents an error that occurred during test execution
type TestError struct {
	Message string `json:"message"`
	Event   any    `json:"event,omitempty"`
}

// TestResult represents the result of a single test run
type TestResult struct {
	Name         string        `json:"name"`
	Description  string        `json:"description,omitempty"`
	Status       TestRunStatus `json:"status"`
	ActualOutput []any         `json:"actualOutput,omitempty"`
	Errors       []TestError   `json:"errors,omitempty"`
}

// TestSuiteRunResult represents the aggregate result of running a test suite
type TestSuiteRunResult struct {
	Status  TestRunStatus `json:"status"`
	Results []TestResult  `json:"results"`
}

// TransformationTestResult represents result for a single transformation's test suite
type TransformationTestResult struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	VersionID  string             `json:"versionId"`
	Imports    []string           `json:"imports,omitempty"`
	Pass       bool               `json:"pass"`
	TestSuiteResult TestSuiteRunResult `json:"testResult"`
	Message    string             `json:"message,omitempty"`
}

// LibraryTestResult represents validation result for a library in batch test response
type LibraryTestResult struct {
	HandleName string `json:"handleName"`
	VersionID  string `json:"versionId"`
	Pass       bool   `json:"pass"`
	Message    string `json:"message,omitempty"`
}

// ValidationOutput contains validation details for libraries and transformations
type ValidationOutput struct {
	Libraries       []LibraryTestResult        `json:"libraries,omitempty"`
	Transformations []TransformationTestResult `json:"transformations,omitempty"`
}

type BatchTestResponse struct {
	Pass             bool             `json:"pass"`
	Message          string           `json:"message,omitempty"`
	ValidationOutput ValidationOutput `json:"validationOutput"`
}

type BatchPublishResponse struct {
	Published        bool             `json:"published"`
	Message          string           `json:"message,omitempty"`
	ValidationOutput ValidationOutput `json:"validationOutput"`
}
