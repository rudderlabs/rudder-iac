package rules

// Examples provides valid and invalid usage examples for a rule.
// These examples help users understand what the rule checks for
// and how to write correct specs.
type Examples struct {
	// Valid contains examples of correct spec configurations
	// that would pass the rule's validation.
	Valid []string

	// Invalid contains examples of incorrect spec configurations
	// that would fail the rule's validation.
	Invalid []string
}

// HasExamples returns true if at least one example (valid or invalid) is provided.
// This is useful for checking if a rule has documentation examples available.
func (e Examples) HasExamples() bool {
	return len(e.Valid) > 0 || len(e.Invalid) > 0
}
