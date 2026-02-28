package rules

import (
	"fmt"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateCategorySemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.CategorySpec, graph *resources.Graph) []rules.ValidationResult {
	return validateCategoryNameUniqueness(spec, graph)
}

// validateCategoryNameUniqueness checks that each category's name is unique
// across the entire resource graph. Category names must be globally unique
// in the catalog.
func validateCategoryNameUniqueness(spec localcatalog.CategorySpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.CategoryResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}

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

func NewCategorySemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/categories/semantic-valid",
		rules.Error,
		"category names must be unique across the catalog",
		rules.Examples{},
		prules.LegacyVersionPatterns(localcatalog.KindCategories),
		validateCategorySemantic,
	)
}