package rules

// MatchPattern defines a (Kind, Version) pair that a rule applies to.
// Use "*" for either field as a wildcard to match all values.
type MatchPattern struct {
	Kind    string // specific kind or "*" for wildcard
	Version string // specific version or "*" for wildcard
}

// Matches returns true if this pattern matches the given kind and version.
// A "*" in either field acts as a wildcard matching any value.
func (mp MatchPattern) Matches(kind, version string) bool {
	return (mp.Kind == "*" || mp.Kind == kind) &&
		(mp.Version == "*" || mp.Version == version)
}

// MatchAll returns a pattern that matches all kinds and all versions.
func MatchAll() MatchPattern {
	return MatchPattern{Kind: "*", Version: "*"}
}

// MatchKind returns a pattern that matches a specific kind across all versions.
func MatchKind(kind string) MatchPattern {
	return MatchPattern{Kind: kind, Version: "*"}
}

// MatchKindVersion returns a pattern that matches a specific kind and version.
func MatchKindVersion(kind, version string) MatchPattern {
	return MatchPattern{Kind: kind, Version: version}
}

// Rule defines the interface that all validation rules must implement.
// Rules can be either syntactic (pre-graph, validating spec structure and format)
// or semantic (post-graph, validating cross-resource relationships and business logic).
//
// Rules should be stateless and idempotent - they should not modify any state
// and should always return the same results for the same input.
type Rule interface {
	// ID returns a unique identifier for this rule.
	// Convention: use kebab-case (e.g., "required-name-field", "no-circular-deps")
	ID() string

	// Severity returns the default severity level for violations of this rule.
	// Individual ValidationResults can override this if needed.
	Severity() Severity

	// Description returns a human-readable description of what this rule checks.
	// This should be a complete sentence explaining the rule's purpose.
	Description() string

	// AppliesTo returns the list of (Kind, Version) patterns this rule validates.
	// Use []MatchPattern{MatchAll()} for rules that apply universally.
	// Use []MatchPattern{MatchKind("properties")} for kind-specific rules across all versions.
	// Use []MatchPattern{MatchKindVersion("properties", "rudder/v1")} for specific combinations.
	// A rule is selected if ANY of its patterns match the spec's (kind, version).
	AppliesTo() []MatchPattern

	// Examples returns usage examples for this rule.
	// Can return nil if no examples are provided.
	// Examples help users understand what the rule checks for.
	Examples() Examples

	// Validate performs the actual validation logic.
	// It receives a ValidationContext with all necessary information
	// and returns a slice of ValidationResults (empty slice if validation passes).
	//
	// For syntactic rules, ctx.Graph will be nil.
	// For semantic rules, ctx.Graph will be populated with the resource graph.
	Validate(ctx *ValidationContext) []ValidationResult
}
