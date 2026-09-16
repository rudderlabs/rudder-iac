package rules

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// PatternValidator encapsulates a typed validation handler for specific (Kind, Version)
// match patterns. Each validator owns its own spec type through a closure created
// by the generic constructors NewPatternValidator or NewSemanticPatternValidator.
type PatternValidator struct {
	patterns []rules.MatchPattern
	validate func(ctx *rules.ValidationContext) []rules.ValidationResult
}

// Patterns returns the match patterns this validator handles.
func (v PatternValidator) Patterns() []rules.MatchPattern {
	return v.patterns
}

// NewPatternValidator creates a syntactic validation handler (no graph access).
// The generic type T determines the spec struct the raw map[string]any is
// unmarshaled into before being passed to fn.
func NewPatternValidator[T any](
	patterns []rules.MatchPattern,
	fn func(Kind string, Version string, Metadata map[string]any, Spec T) []rules.ValidationResult,
) PatternValidator {
	return PatternValidator{
		patterns: patterns,
		validate: func(ctx *rules.ValidationContext) []rules.ValidationResult {
			spec, err := unmarshalSpec[T](ctx.Spec)
			if err != nil {
				return err
			}

			results := fn(ctx.Kind, ctx.Version, ctx.Metadata, spec)
			prefixReferences(results)
			return results
		},
	}
}

// NewPathAwarePatternValidator creates a syntactic validation handler that also
// receives the absolute spec file path from ValidationContext.
func NewPathAwarePatternValidator[T any](
	patterns []rules.MatchPattern,
	fn func(Kind string, Version string, FilePath string, Metadata map[string]any, Spec T) []rules.ValidationResult,
) PatternValidator {
	return PatternValidator{
		patterns: patterns,
		validate: func(ctx *rules.ValidationContext) []rules.ValidationResult {
			spec, err := unmarshalSpec[T](ctx.Spec)
			if err != nil {
				return err
			}

			results := fn(ctx.Kind, ctx.Version, ctx.FilePath, ctx.Metadata, spec)
			prefixReferences(results)
			return results
		},
	}
}

// NewRawPatternValidator creates a syntactic validation handler that receives
// the raw spec map instead of a decoded struct, for a rule that has to own its
// decoding. The json round-trip the typed constructors use cannot express a
// strict spec shape: it drops keys the spec type does not carry without a word,
// and collapses every wrong-typed value into a single result at the spec root
// rather than pointing at the field. A rule that decodes the map itself reports
// both against the entry that carries them.
func NewRawPatternValidator(
	patterns []rules.MatchPattern,
	fn func(Kind string, Version string, Metadata map[string]any, Spec map[string]any) []rules.ValidationResult,
) PatternValidator {
	return PatternValidator{
		patterns: patterns,
		validate: func(ctx *rules.ValidationContext) []rules.ValidationResult {
			results := fn(ctx.Kind, ctx.Version, ctx.Metadata, ctx.Spec)
			prefixReferences(results)
			return results
		},
	}
}

// NewSemanticPatternValidator creates a semantic validation handler (with graph access).
// The generic type T determines the spec struct the raw map[string]any is
// unmarshaled into before being passed to fn along with the resource graph.
func NewSemanticPatternValidator[T any](
	patterns []rules.MatchPattern,
	fn func(Kind string, Version string, Metadata map[string]any, Spec T, Graph *resources.Graph) []rules.ValidationResult,
) PatternValidator {
	return PatternValidator{
		patterns: patterns,
		validate: func(ctx *rules.ValidationContext) []rules.ValidationResult {
			spec, err := unmarshalSpec[T](ctx.Spec)
			if err != nil {
				return err
			}

			results := fn(ctx.Kind, ctx.Version, ctx.Metadata, spec, ctx.Graph)
			prefixReferences(results)
			return results
		},
	}
}

// unmarshalSpec round-trips the raw spec through JSON to produce a typed value.
// Returns the typed spec on success, or a single-element error slice on failure.
func unmarshalSpec[T any](raw map[string]any) (T, []rules.ValidationResult) {
	var spec T

	jsonByt, err := json.Marshal(raw)
	if err != nil {
		return spec, []rules.ValidationResult{{
			Reference: "/spec",
			Message:   fmt.Sprintf("failed to marshal spec: %v", err),
		}}
	}

	if err := json.Unmarshal(jsonByt, &spec); err != nil {
		return spec, []rules.ValidationResult{{
			Reference: "/spec",
			Message:   fmt.Sprintf("failed to unmarshal spec: %v", err),
		}}
	}

	return spec, nil
}

// prefixReferences prepends "/spec" to every result's Reference field
// so that JSON pointers are rooted at the spec level in the YAML document.
func prefixReferences(results []rules.ValidationResult) {
	for i := range results {
		results[i].Reference = "/spec" + results[i].Reference
	}
}
