package specs

// TransformationSpec represents the user-defined specification for a transformation
type TransformationSpec struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Language    string                 `json:"language"` // "javascript" or "python"
	Code        string                 `json:"code,omitempty"`
	File        string                 `json:"file,omitempty"`
	Tests       []TransformationTest   `json:"tests,omitempty"`
}

// TransformationTest represents a test case for a transformation
type TransformationTest struct {
	Name   string `json:"name"`
	Input  string `json:"input"`  // Path to directory with input JSON files
	Output string `json:"output"` // Path to directory with expected output JSON files
}

// TransformationLibrarySpec represents the user-defined specification for a transformation library
type TransformationLibrarySpec struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Language    string `json:"language"` // "javascript" or "python"
	Code        string `json:"code,omitempty"`
	File        string `json:"file,omitempty"`
	ImportName  string `json:"import_name"` // Handle name for importing in transformations
}
