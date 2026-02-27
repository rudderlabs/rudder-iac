package rules

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type semanticTypedRuleImpl[T any] struct {
	id           string
	severity     rules.Severity
	description  string
	examples     rules.Examples
	appliesTo    []rules.MatchPattern
	validateFunc func(Kind string, Version string, Metadata map[string]any, Spec T, Graph *resources.Graph) []rules.ValidationResult
}

var _ rules.Rule = &semanticTypedRule[any]{}

type semanticTypedRule[T any] struct {
	rule semanticTypedRuleImpl[T]
}

// NewSemanticTypedRule creates a semantic validation rule that receives a typed spec
// and the resource graph. Parallel to NewTypedRule but for post-graph validation
// where cross-resource lookups are needed.
func NewSemanticTypedRule[T any](
	id string,
	severity rules.Severity,
	description string,
	examples rules.Examples,
	appliesToKinds []string,
	validateFunc func(
		Kind string,
		Version string,
		Metadata map[string]any,
		Spec T,
		Graph *resources.Graph,
	) []rules.ValidationResult,
) rules.Rule {
	patterns := make([]rules.MatchPattern, len(appliesToKinds))
	for i, kind := range appliesToKinds {
		patterns[i] = rules.MatchKind(kind)
	}
	return &semanticTypedRule[T]{rule: semanticTypedRuleImpl[T]{
		id:           id,
		severity:     severity,
		description:  description,
		examples:     examples,
		appliesTo:    patterns,
		validateFunc: validateFunc,
	}}
}

func (w *semanticTypedRule[T]) ID() string {
	return w.rule.id
}

func (w *semanticTypedRule[T]) Severity() rules.Severity {
	return w.rule.severity
}

func (w *semanticTypedRule[T]) Description() string {
	return w.rule.description
}

func (w *semanticTypedRule[T]) Examples() rules.Examples {
	return w.rule.examples
}

func (w *semanticTypedRule[T]) AppliesTo() []rules.MatchPattern {
	return w.rule.appliesTo
}

func (w *semanticTypedRule[T]) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	var typedSpec T

	jsonByt, err := json.Marshal(ctx.Spec)
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/spec",
				Message:   fmt.Sprintf("failed to marshal spec: %v", err),
			},
		}
	}

	if err := json.Unmarshal(jsonByt, &typedSpec); err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/spec",
				Message:   fmt.Sprintf("failed to unmarshal spec: %v", err),
			},
		}
	}

	results := w.rule.validateFunc(
		ctx.Kind,
		ctx.Version,
		ctx.Metadata,
		typedSpec,
		ctx.Graph,
	)

	for i := range results {
		results[i].Reference = "/spec" + results[i].Reference
	}

	return results
}
