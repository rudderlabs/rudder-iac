package rules

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type typedRuleImpl[T any] struct {
	id                string
	severity          rules.Severity
	description       string
	examples          rules.Examples
	appliesToKinds    []string
	appliesToVersions []string
	validateFunc      func(Kind string, Version string, Metadata map[string]any, Spec T) []rules.ValidationResult
}

var _ rules.Rule = &typedRule[any]{}

type typedRule[T any] struct {
	rule typedRuleImpl[T]
}

func NewTypedRule[T any](
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
	) []rules.ValidationResult,
) rules.Rule {
	return &typedRule[T]{rule: typedRuleImpl[T]{
		id:                id,
		severity:          severity,
		description:       description,
		examples:          examples,
		appliesToKinds:    appliesToKinds,
		appliesToVersions: []string{"*"},
		validateFunc:      validateFunc,
	}}
}

func (w *typedRule[T]) ID() string {
	return w.rule.id
}

func (w *typedRule[T]) Severity() rules.Severity {
	return w.rule.severity
}

func (w *typedRule[T]) Description() string {
	return w.rule.description
}

func (w *typedRule[T]) Examples() rules.Examples {
	return w.rule.examples
}

func (w *typedRule[T]) AppliesToKinds() []string {
	return w.rule.appliesToKinds
}

func (w *typedRule[T]) AppliesToVersions() []string {
	return w.rule.appliesToVersions
}

func (w *typedRule[T]) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
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
	)

	// attach the spec prefix to the reference
	// generated from results through the typed rule.
	for i := range results {
		results[i].Reference = "/spec" + results[i].Reference
	}

	return results
}
