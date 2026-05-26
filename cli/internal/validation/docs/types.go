package docs

// Authored types — what providers write in YAML fragments.

type RuleDocEntry struct {
	RuleID        string               `json:"rule_id"        yaml:"rule_id"`
	MatchBehavior []MatchBehaviorEntry `json:"match_behavior" yaml:"match_behavior"`
}

type MatchBehaviorEntry struct {
	AppliesTo []MatchPatternDoc `json:"applies_to"        yaml:"applies_to"        validate:"required,min=1,dive"`
	Valid     []ValidExample    `json:"valid,omitempty"   yaml:"valid,omitempty"   validate:"dive"`
	Invalid   []InvalidExample  `json:"invalid,omitempty" yaml:"invalid,omitempty" validate:"dive"`
}

type MatchPatternDoc struct {
	Kind    string `json:"kind"    yaml:"kind"    validate:"required"`
	Version string `json:"version" yaml:"version" validate:"required"`
}

type ValidExample struct {
	ExampleID string            `json:"example_id" yaml:"example_id" validate:"required"`
	Title     string            `json:"title"      yaml:"title"      validate:"required"`
	Files     map[string]string `json:"files"      yaml:"files"      validate:"required,min=1"`
}

type InvalidExample struct {
	ExampleID           string               `json:"example_id"           yaml:"example_id"           validate:"required"`
	Title               string               `json:"title"                yaml:"title"                validate:"required"`
	Files               map[string]string    `json:"files"                yaml:"files"                validate:"required,min=1"`
	ExpectedDiagnostics []ExpectedDiagnostic `json:"expected_diagnostics" yaml:"expected_diagnostics" validate:"required,min=1,dive"`
}

type ExpectedDiagnostic struct {
	File            string `json:"file"                      yaml:"file"                      validate:"required"`
	Reference       string `json:"reference"                 yaml:"reference"                 validate:"required"`
	Severity        string `json:"severity"                  yaml:"severity"                  validate:"required,oneof=error warning info"`
	MessageContains string `json:"message_contains,omitempty" yaml:"message_contains,omitempty"`
}

// Resolved types — what the generator emits in the YAML catalog.

type RulesDoc struct {
	SchemaVersion int            `json:"schema_version" yaml:"schema_version"`
	ToolMetadata  ToolMetadata   `json:"tool_metadata"  yaml:"tool_metadata"`
	Rules         []ResolvedRule `json:"rules"          yaml:"rules"`
}

type ToolMetadata struct {
	CLIVersion string `json:"cli_version" yaml:"cli_version"`
}

type ResolvedRule struct {
	RuleID        string               `json:"rule_id"        yaml:"rule_id"        validate:"required"`
	Phase         string               `json:"phase"          yaml:"phase"          validate:"required,oneof=syntactic semantic"`
	Severity      string               `json:"severity"       yaml:"severity"       validate:"required,oneof=error warning info"`
	Description   string               `json:"description"    yaml:"description"    validate:"required"`
	AppliesTo     []MatchPatternDoc    `json:"applies_to"     yaml:"applies_to"     validate:"required,min=1,dive"`
	MatchBehavior []MatchBehaviorEntry `json:"match_behavior" yaml:"match_behavior" validate:"required,min=1,dive"`
}
