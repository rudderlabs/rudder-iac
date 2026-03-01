package rules

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Variant encapsulates a typed validation handler for specific (Kind, Version)
// match patterns. Each variant owns its own spec type through a closure created
// by the generic constructors NewVariant or NewSemanticVariant.
type Variant struct {
	patterns []rules.MatchPattern
	validate func(ctx *rules.ValidationContext) []rules.ValidationResult
}

// Patterns returns the match patterns this variant handles.
func (v Variant) Patterns() []rules.MatchPattern {
	return v.patterns
}

// NewVariant creates a syntactic validation variant (no graph access).
// The generic type T determines the spec struct the raw map[string]any is
// unmarshaled into before being passed to fn.
func NewVariant[T any](
	patterns []rules.MatchPattern,
	fn func(Kind string, Version string, Metadata map[string]any, Spec T) []rules.ValidationResult,
) Variant {
	return Variant{
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

// NewSemanticVariant creates a semantic validation variant (with graph access).
// The generic type T determines the spec struct the raw map[string]any is
// unmarshaled into before being passed to fn along with the resource graph.
func NewSemanticVariant[T any](
	patterns []rules.MatchPattern,
	fn func(Kind string, Version string, Metadata map[string]any, Spec T, Graph *resources.Graph) []rules.ValidationResult,
) Variant {
	return Variant{
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
