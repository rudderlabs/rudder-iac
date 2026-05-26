package docs

// Authored types — providers attach these via the Documented interface.

type MatchBehaviorEntry struct {
	AppliesTo []MatchPatternDoc `yaml:"applies_to"        json:"applies_to"        validate:"required,min=1,dive"`
	Valid     []ValidExample    `yaml:"valid,omitempty"   json:"valid,omitempty"   validate:"dive"`
	Invalid   []InvalidExample  `yaml:"invalid,omitempty" json:"invalid,omitempty" validate:"dive"`
}

type MatchPatternDoc struct {
	Kind    string `yaml:"kind"    json:"kind"    validate:"required"`
	Version string `yaml:"version" json:"version" validate:"required"`
}

type ValidExample struct {
	ExampleID string            `yaml:"example_id" json:"example_id" validate:"required"`
	Title     string            `yaml:"title"      json:"title"      validate:"required"`
	Files     map[string]string `yaml:"files"      json:"files"      validate:"required,min=1"`
}

type InvalidExample struct {
	ExampleID           string               `yaml:"example_id"           json:"example_id"           validate:"required"`
	Title               string               `yaml:"title"                json:"title"                validate:"required"`
	Files               map[string]string    `yaml:"files"                json:"files"                validate:"required,min=1"`
	ExpectedDiagnostics []ExpectedDiagnostic `yaml:"expected_diagnostics" json:"expected_diagnostics" validate:"required,min=1,dive"`
}

type ExpectedDiagnostic struct {
	File            string `yaml:"file"                      json:"file"                      validate:"required"`
	Reference       string `yaml:"reference"                 json:"reference"                 validate:"required"`
	Severity        string `yaml:"severity"                  json:"severity"                  validate:"required,oneof=error warning info"`
	MessageContains string `yaml:"message_contains,omitempty" json:"message_contains,omitempty"`
}

// Resolved types — what the generator emits in the YAML catalog.

type RulesDoc struct {
	SchemaVersion int            `yaml:"schema_version" json:"schema_version"`
	ToolMetadata  ToolMetadata   `yaml:"tool_metadata"  json:"tool_metadata"`
	Rules         []ResolvedRule `yaml:"rules"          json:"rules"`
}

type ToolMetadata struct {
	CLIVersion string `yaml:"cli_version" json:"cli_version"`
}

type ResolvedRule struct {
	RuleID        string               `yaml:"rule_id"        json:"rule_id"        validate:"required"`
	Phase         string               `yaml:"phase"          json:"phase"          validate:"required,oneof=syntactic semantic"`
	Severity      string               `yaml:"severity"       json:"severity"       validate:"required,oneof=error warning info"`
	Description   string               `yaml:"description"    json:"description"    validate:"required"`
	AppliesTo     []MatchPatternDoc    `yaml:"applies_to"     json:"applies_to"     validate:"required,min=1,dive"`
	MatchBehavior []MatchBehaviorEntry `yaml:"match_behavior" json:"match_behavior" validate:"required,min=1,dive"`
}
