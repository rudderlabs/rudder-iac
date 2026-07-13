package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateCategorySemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.CategorySpec, graph *resources.Graph) []rules.ValidationResult {
	return validateCategoryNameUniqueness(spec, graph)
}

var validateCategorySemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.CategorySpecV1, graph *resources.Graph) []rules.ValidationResult {
	return validateCategoryNameUniquenessV1(spec, graph)
}

// validateCategoryNameUniqueness checks that each category's name is unique
// across the entire resource graph. Category names must be globally unique
// in the catalog.
func validateCategoryNameUniqueness(spec localcatalog.CategorySpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := buildCategoryNameCountMap(graph)

	var results []rules.ValidationResult
	for i, category := range spec.Categories {
		if countMap[category.Name] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/categories/%d/name", i),
				Message:   fmt.Sprintf("duplicate name '%s' within kind 'categories'", category.Name),
			})
		}
	}

	return results
}

func validateCategoryNameUniquenessV1(spec localcatalog.CategorySpecV1, graph *resources.Graph) []rules.ValidationResult {
	countMap := buildCategoryNameCountMap(graph)

	var results []rules.ValidationResult
	for i, category := range spec.Categories {
		if countMap[category.Name] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/categories/%d/name", i),
				Message:   fmt.Sprintf("duplicate name '%s' within kind 'categories'", category.Name),
			})
		}
	}

	return results
}

func buildCategoryNameCountMap(graph *resources.Graph) map[string]int {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.CategoryResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}
	return countMap
}

func NewCategorySemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/categories/semantic-valid",
		rules.Error,
		"category names must be unique across the catalog",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindCategories),
			validateCategorySemantic,
		),
		prules.NewSemanticPatternValidator(
			[]rules.MatchPattern{rules.MatchKindVersion(
				localcatalog.KindCategories,
				specs.SpecVersionV1,
			)},
			validateCategorySemanticV1,
		),
	)
}
