package specs

// TransformationSpec represents a transformation YAML spec
type TransformationSpec struct {
	ID          string               `yaml:"id"`
	Name        string               `yaml:"name"`
	Description string               `yaml:"description,omitempty"`
	Language    string               `yaml:"language"` // "javascript" | "python"
	Code        string               `yaml:"code,omitempty"`
	File        string               `yaml:"file,omitempty"`
	Tests       []TransformationTest `yaml:"tests,omitempty"`
}

type TransformationTest struct {
	Name   string `yaml:"name"`
	Input  string `yaml:"input,omitempty"`  // Default: "./input"
	Output string `yaml:"output,omitempty"` // Default: "./output"
}

// TransformationLibrarySpec represents a library YAML spec
type TransformationLibrarySpec struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Language    string `yaml:"language"` // "javascript" | "python"
	Code        string `yaml:"code,omitempty"`
	File        string `yaml:"file,omitempty"`
	ImportName  string `yaml:"import_name"` // Handle name for imports
}
