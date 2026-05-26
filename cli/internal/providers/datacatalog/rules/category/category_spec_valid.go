package rules

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
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
var validateCategorySpec = func(_ string, _ string, _ map[string]any, spec localcatalog.CategorySpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(
		spec,
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

	return funcs.ParseValidationErrors(validationErrors, nil)
}

func NewCategorySpecSyntaxValidRule() rules.Rule {
	base := prules.NewTypedRule(
		"datacatalog/categories/spec-syntax-valid",
		rules.Error,
		"category spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindCategories),
			validateCategorySpec,
		),
		prules.NewPatternValidator(
			[]rules.MatchPattern{rules.MatchKindVersion(
				localcatalog.KindCategories,
				specs.SpecVersionV1,
			)},
			validateCategorySpecV1,
		),
	)
	return docsCategorySpecRule{Rule: base}
}

// docsCategorySpecRule wraps the typed rule with documentation data.
// The typed rule is returned by prules.NewTypedRule and isn't under this
// package's control; embedding lets us add DocExamples() without
// re-implementing the rules.Rule surface.
type docsCategorySpecRule struct {
	rules.Rule
}

func (docsCategorySpecRule) DocExamples() []docs.MatchBehaviorEntry {
	return []docs.MatchBehaviorEntry{
		{
			AppliesTo: []docs.MatchPatternDoc{
				{Kind: "categories", Version: "rudder/0.1"},
				{Kind: "categories", Version: "rudder/v0.1"},
			},
			Valid: []docs.ValidExample{
				{
					ExampleID: "categories-v0.1-valid-minimal",
					Title:     "Two categories with id and name",
					Files: map[string]string{
						"categories.yaml": heredoc.Doc(`
							kind: categories
							version: rudder/v0.1
							metadata:
							  name: example-categories
							spec:
							  categories:
							    - id: user_actions
							      name: User Actions
							    - id: system_events
							      name: System Events
						`),
					},
				},
			},
			Invalid: []docs.InvalidExample{
				{
					ExampleID: "categories-v0.1-missing-id",
					Title:     "Category missing id",
					Files: map[string]string{
						"categories.yaml": heredoc.Doc(`
							kind: categories
							version: rudder/v0.1
							metadata:
							  name: example-categories
							spec:
							  categories:
							    - name: No ID Here
						`),
					},
					ExpectedDiagnostics: []docs.ExpectedDiagnostic{
						{
							File:            "categories.yaml",
							Reference:       "datacatalog/categories/spec-syntax-valid",
							Severity:        "error",
							MessageContains: "id",
						},
					},
				},
			},
		},
		{
			AppliesTo: []docs.MatchPatternDoc{{Kind: "categories", Version: "rudder/v1"}},
			Valid: []docs.ValidExample{
				{
					ExampleID: "categories-v1-valid-minimal",
					Title:     "Two categories with id and name (v1)",
					Files: map[string]string{
						"categories.yaml": heredoc.Doc(`
							kind: categories
							version: rudder/v1
							metadata:
							  name: example-categories
							spec:
							  categories:
							    - id: user_actions
							      name: User Actions
							    - id: system_events
							      name: System Events
						`),
					},
				},
			},
			Invalid: []docs.InvalidExample{
				{
					ExampleID: "categories-v1-missing-name",
					Title:     "Category missing name (v1)",
					Files: map[string]string{
						"categories.yaml": heredoc.Doc(`
							kind: categories
							version: rudder/v1
							metadata:
							  name: example-categories
							spec:
							  categories:
							    - id: user_actions
						`),
					},
					ExpectedDiagnostics: []docs.ExpectedDiagnostic{
						{
							File:            "categories.yaml",
							Reference:       "datacatalog/categories/spec-syntax-valid",
							Severity:        "error",
							MessageContains: "name",
						},
					},
				},
			},
		},
	}
}
