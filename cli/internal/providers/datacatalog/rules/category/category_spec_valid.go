package rules

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

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

	return funcs.ParseValidationErrors(validationErrors, nil)
}

var validateCategorySpecV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.CategorySpecV1) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/categories",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)
	results = append(results, validateCategoryNameWhitespace(spec)...)

	return results
}

func validateCategoryNameWhitespace(spec localcatalog.CategorySpecV1) []rules.ValidationResult {
	var results []rules.ValidationResult
	for i, category := range spec.Categories {
		if category.Name != strings.TrimSpace(category.Name) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/categories/%d/name", i),
				Message:   "'name' must not have leading or trailing whitespace",
			})
		}
	}
	return results
}

func NewCategorySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/categories/spec-syntax-valid",
		rules.Error,
		"category spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindCategories),
			validateCategorySpec,
		),
		prules.NewPatternValidator(
			[]rules.MatchPattern{rules.MatchKindVersion(localcatalog.KindCategories, specs.SpecVersionV1)},
			validateCategorySpecV1,
		),
	)
}
