package rules

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

	// AppliesTo returns the list of spec kinds this rule validates.
	// Use ["*"] to indicate the rule applies to all kinds.
	// Use specific kinds (e.g., ["properties", "events"]) for kind-specific rules.
	AppliesTo() []string

	// Examples returns usage examples for this rule.
	// Can return nil if no examples are provided.
	// Examples help users understand what the rule checks for.
	Examples() *Examples

	// Validate performs the actual validation logic.
	// It receives a ValidationContext with all necessary information
	// and returns a slice of ValidationResults (empty slice if validation passes).
	//
	// For syntactic rules, ctx.Graph will be nil.
	// For semantic rules, ctx.Graph will be populated with the resource graph.
	Validate(ctx *ValidationContext) []ValidationResult
}
