package specs

// TransformationSpec represents a transformation resource as defined in YAML.
// This is a pure data structure for unmarshaling user-written YAML specs.
// No validation logic or computed fields are included here.
//
// Transformations execute custom code on events as they flow through RudderStack.
// They can reference transformation libraries for shared functionality.
//
// Dependencies (libraries used by transformations) are NOT declared in this spec.
// They are computed later by parsing the transformation code to extract library
// handle names from import statements.
type TransformationSpec struct {
	// ID is the unique identifier for this transformation within the project
	ID string `json:"id"`

	// Name is the human-readable display name for the transformation
	Name string `json:"name"`

	// Description provides additional context about what this transformation does
	Description string `json:"description,omitempty"`

	// Language specifies the programming language of the transformation code.
	// Valid values: "javascript", "python"
	Language string `json:"language"`

	// Code contains the inline transformation code.
	// Mutually exclusive with File. Validation is performed in the handler.
	Code string `json:"code,omitempty"`

	// File specifies the path to an external file containing the transformation code.
	// Relative paths are resolved relative to the spec file's parent directory.
	// Mutually exclusive with Code. Validation is performed in the handler.
	File string `json:"file,omitempty"`

	// Tests defines the test suites for this transformation.
	// Tests are local-only and never synced to the RudderStack workspace.
	Tests []TransformationTest `json:"tests,omitempty"`
}

// TransformationTest represents a test suite configuration for a transformation.
// Tests execute the transformation against input JSON files and optionally
// compare the output to expected results.
//
// Tests are local-only development tools and are never uploaded to RudderStack.
type TransformationTest struct {
	// Name is a human-readable identifier for this test suite
	Name string `json:"name"`

	// Input is the path to a directory containing JSON files with test event payloads.
	// Each JSON file will be passed to the transformEvent function.
	// Relative paths are resolved relative to the spec file's parent directory.
	// Default: "./input" (applied in handler logic, not here)
	Input string `json:"input,omitempty"`

	// Output is the path to a directory containing expected output JSON files.
	// If specified, each file name must match a corresponding input file.
	// The transformation output will be compared against these expected results.
	// Relative paths are resolved relative to the spec file's parent directory.
	// Default: "./output" (applied in handler logic, not here)
	Output string `json:"output,omitempty"`
}

// TransformationLibrarySpec represents a transformation library resource as defined in YAML.
// This is a pure data structure for unmarshaling user-written YAML specs.
//
// Libraries provide reusable code that can be imported by transformations.
// Each library has a unique ImportName (handle name) used in transformation imports.
type TransformationLibrarySpec struct {
	// ID is the unique identifier for this library within the project
	ID string `json:"id"`

	// Name is the human-readable display name for the library
	Name string `json:"name"`

	// Description provides additional context about what this library provides
	Description string `json:"description,omitempty"`

	// Language specifies the programming language of the library code.
	// Valid values: "javascript", "python"
	Language string `json:"language"`

	// Code contains the inline library code.
	// Mutually exclusive with File. Validation is performed in the handler.
	Code string `json:"code,omitempty"`

	// File specifies the path to an external file containing the library code.
	// Relative paths are resolved relative to the spec file's parent directory.
	// Mutually exclusive with Code. Validation is performed in the handler.
	File string `json:"file,omitempty"`

	// ImportName is the handle name used when importing this library in transformations.
	// This is typically a camelCase identifier (e.g., "myMathLibrary").
	// Must be unique within the workspace.
	ImportName string `json:"import_name"`
}
