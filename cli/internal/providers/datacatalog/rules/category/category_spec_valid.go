package rules

import (
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const (
	categoryNameRegexPattern = `^[A-Z_a-z][\s\w,.-]{2,64}$`
	categoryNameRegexTag     = "category_name"
	categoryNameErrorMessage = "must start with a letter or underscore, followed by 2-64 alphanumeric, space, comma, period, or hyphen characters"
)

func init() {
	funcs.NewPattern(
		categoryNameRegexTag,
		categoryNameRegexPattern,
		categoryNameErrorMessage,
	)
}

var examples = rules.Examples{
	Valid: []string{
		`categories:
  - id: user_actions
    name: User Actions
  - id: system_events
    name: System Events`,
	},
	Invalid: []string{
		`categories:
  - name: Missing ID`,
		`categories:
  - id: missing_name
    # Missing required name field`,
	},
}

// Main validation function for category spec
// which delegates the validation to the go-validator through struct tags.
var validateCategorySpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CategorySpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		Spec,
		"",
	)

	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/categories",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(validationErrors)
}

func NewCategorySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/categories/spec-syntax-valid",
		rules.Error,
		"category spec syntax must be valid",
		examples,
		[]string{"categories"},
		validateCategorySpec,
	)
}
