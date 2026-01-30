package rules

import (
	"encoding/json"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var _ rules.Rule = &TypedRuleWrapper[any]{}

type TypedRuleWrapper[T any] struct {
	rule TypedRule[T]
}

func NewTypedRuleWrapper[T any](rule TypedRule[T]) rules.Rule {
	return &TypedRuleWrapper[T]{rule: rule}
}

func (w *TypedRuleWrapper[T]) ID() string {
	return w.rule.ID()
}

func (w *TypedRuleWrapper[T]) Severity() rules.Severity {
	return w.rule.Severity()
}

func (w *TypedRuleWrapper[T]) Description() string {
	return w.rule.Description()
}

func (w *TypedRuleWrapper[T]) Examples() rules.Examples {
	return w.rule.Examples()
}

func (w *TypedRuleWrapper[T]) AppliesTo() []string {
	return w.rule.AppliesTo()
}

func (w *TypedRuleWrapper[T]) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	var typedSpec T

	jsonByt, err := json.Marshal(ctx.Spec)
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "",
				Message:   fmt.Sprintf("failed to marshal spec: %v", err),
			},
		}
	}

	if err := json.Unmarshal(jsonByt, &typedSpec); err != nil {
		return []rules.ValidationResult{
			{
				Reference: "",
				Message:   fmt.Sprintf("failed to unmarshal spec: %v", err),
			},
		}
	}

	return w.rule.Validate(
		ctx.Kind,
		ctx.Version,
		ctx.Metadata,
		typedSpec,
	)
}
