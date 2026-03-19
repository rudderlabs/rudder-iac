package docs

// Authored types — what providers write in YAML fragments.

type RuleDocEntry struct {
	RuleID        string               `yaml:"rule_id"`
	MatchBehavior []MatchBehaviorEntry `yaml:"match_behavior"`
}

type MatchBehaviorEntry struct {
	AppliesTo []MatchPatternDoc `yaml:"applies_to" validate:"required,min=1,dive"`
	Valid     []ValidExample    `yaml:"valid,omitempty"   validate:"dive"`
	Invalid   []InvalidExample  `yaml:"invalid,omitempty" validate:"dive"`
}

type MatchPatternDoc struct {
	Kind    string `yaml:"kind"    validate:"required"`
	Version string `yaml:"version" validate:"required"`
}

type ValidExample struct {
	ExampleID string            `yaml:"example_id" validate:"required"`
	Title     string            `yaml:"title"      validate:"required"`
	Files     map[string]string `yaml:"files"      validate:"required,min=1"`
}

type InvalidExample struct {
	ExampleID           string               `yaml:"example_id"           validate:"required"`
	Title               string               `yaml:"title"                validate:"required"`
	Files               map[string]string    `yaml:"files"                validate:"required,min=1"`
	ExpectedDiagnostics []ExpectedDiagnostic `yaml:"expected_diagnostics" validate:"required,min=1,dive"`
}

type ExpectedDiagnostic struct {
	File            string `yaml:"file"                      validate:"required"`
	Reference       string `yaml:"reference"                 validate:"required"`
	Severity        string `yaml:"severity"                  validate:"required,oneof=error warning info"`
	MessageContains string `yaml:"message_contains,omitempty"`
}

// Resolved types — what the generator emits in the YAML catalog.

type RulesDoc struct {
	SchemaVersion int            `yaml:"schema_version"`
	ToolMetadata  ToolMetadata   `yaml:"tool_metadata"`
	Rules         []ResolvedRule `yaml:"rules"`
}

type ToolMetadata struct {
	CLIVersion  string `yaml:"cli_version"`
	GeneratedAt string `yaml:"generated_at"`
}

type ResolvedRule struct {
	RuleID        string               `yaml:"rule_id"        validate:"required"`
	Provider      string               `yaml:"provider"       validate:"required"`
	Phase         string               `yaml:"phase"          validate:"required,oneof=syntactic semantic"`
	Severity      string               `yaml:"severity"       validate:"required,oneof=error warning info"`
	Description   string               `yaml:"description"    validate:"required"`
	AppliesTo     []MatchPatternDoc    `yaml:"applies_to"     validate:"required,min=1,dive"`
	MatchBehavior []MatchBehaviorEntry `yaml:"match_behavior" validate:"required,min=1,dive"`
}
