package rules

import "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"

type TypedRule[T any] struct {
	id           string
	severity     rules.Severity
	description  string
	examples     rules.Examples
	appliesTo    []string
	validateFunc func(Kind string, Version string, Metadata map[string]any, Spec T) []rules.ValidationResult
}

func NewTypedRule[T any](
	id string,
	severity rules.Severity,
	description string,
	examples rules.Examples,
	appliesTo []string,
	validateFunc func(Kind string, Version string, Metadata map[string]any, Spec T) []rules.ValidationResult,
) TypedRule[T] {
	return TypedRule[T]{
		id:           id,
		severity:     severity,
		description:  description,
		examples:     examples,
		appliesTo:    appliesTo,
		validateFunc: validateFunc,
	}
}

func (r *TypedRule[T]) ID() string {
	return r.id
}

func (r *TypedRule[T]) Severity() rules.Severity {
	return r.severity
}

func (r *TypedRule[T]) Description() string {
	return r.description
}

func (r *TypedRule[T]) Examples() rules.Examples {
	return r.examples
}

func (r *TypedRule[T]) AppliesTo() []string {
	return r.appliesTo
}

func (r *TypedRule[T]) Validate(
	Kind string,
	Version string,
	Metadata map[string]any,
	Spec T,
) []rules.ValidationResult {
	return r.validateFunc(Kind, Version, Metadata, Spec)
}
