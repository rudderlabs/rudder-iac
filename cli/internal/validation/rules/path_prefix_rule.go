package rules

// PathPrefixRule wraps a rule and prepends a configurable prefix to all
// reference paths in validation results. This is used for provider rules
// which validate within a subsection (e.g., spec) but need full document
// paths for position resolution.
type PathPrefixRule struct {
	inner  Rule
	prefix string
}

// NewPathPrefixRule creates a wrapper that delegates to the inner rule
// and prefixes all reference paths with the given prefix.
func NewPathPrefixRule(inner Rule, prefix string) Rule {
	return &PathPrefixRule{
		inner:  inner,
		prefix: prefix,
	}
}

func (w *PathPrefixRule) ID() string           { return w.inner.ID() }
func (w *PathPrefixRule) Severity() Severity   { return w.inner.Severity() }
func (w *PathPrefixRule) Description() string  { return w.inner.Description() }
func (w *PathPrefixRule) AppliesTo() []string  { return w.inner.AppliesTo() }
func (w *PathPrefixRule) Examples() Examples   { return w.inner.Examples() }

func (w *PathPrefixRule) Validate(ctx *ValidationContext) []ValidationResult {
	results := w.inner.Validate(ctx)
	for i := range results {
		if results[i].Reference != "" {
			results[i].Reference = w.prefix + results[i].Reference
		}
	}
	return results
}
