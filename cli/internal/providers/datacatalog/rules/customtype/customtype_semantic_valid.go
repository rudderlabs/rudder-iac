package customtype

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateCustomTypeSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	// Generic ref validation for pattern=legacy_* tagged fields
	results := funcs.ValidateReferences(spec, graph)

	// Config itemTypes custom type ref resolution
	results = append(results, validateConfigItemTypes(spec, graph)...)

	// Variant discriminator type + ownership checks (shared module)
	results = append(results, validateCustomTypeVariants(spec, graph)...)

	// Name uniqueness across the entire resource graph
	results = append(results, validateCustomTypeNameUniqueness(spec, graph)...)

	return results
}

func validateCustomTypeVariants(spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	// Variant discriminator type + ownership checks (shared module)
	for i, ct := range spec.Types {
		ownRefs := make([]string, len(ct.Properties))
		for j, prop := range ct.Properties {
			ownRefs[j] = prop.Ref
		}
		results = append(results, funcs.ValidateVariantDiscriminators(
			ct.Variants, ownRefs, fmt.Sprintf("/types/%d", i), graph,
		)...)
	}	

	return results
}

// validateCustomTypeNameUniqueness checks that each custom type's name is unique
// across the entire resource graph.
func validateCustomTypeNameUniqueness(spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.CustomTypeResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}

	var results []rules.ValidationResult
	for i, ct := range spec.Types {
		if countMap[ct.Name] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/name", i),
				Message:   fmt.Sprintf("duplicate name '%s' within kind 'custom-types'", ct.Name),
			})
		}
	}

	return results
}

// validateConfigItemTypes checks custom type references in Config["itemTypes"].
// These may contain URN-format refs (e.g., "#custom-type:Address") that must
// exist in the resource graph.
func validateConfigItemTypes(spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, ct := range spec.Types {
		if ct.Config == nil {
			continue
		}

		itemTypes, ok := ct.Config["itemTypes"]
		if !ok {
			continue
		}

		items, ok := itemTypes.([]any)
		if !ok {
			continue
		}

		for j, item := range items {
			ref, ok := item.(string)
			if !ok || !strings.HasPrefix(ref, "#") {
				continue
			}

			resourceType, localID, err := funcs.ParseURNRef(ref)
			if err != nil {
				// if the value is not a valid URN ref, it should
				// be reported in the syntactic validation rule.
				continue
			}

			urn := resources.URN(localID, resourceType)
			if _, exists := graph.GetResource(urn); !exists {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/types/%d/config/itemTypes/%d", i, j),
					Message:   fmt.Sprintf("referenced %s '%s' not found in resource graph", resourceType, localID),
				})
			}
		}
	}

	return results
}

func NewCustomTypeSemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/custom-types/semantic-valid",
		rules.Error,
		"custom type references must resolve to existing resources",
		rules.Examples{},
		[]string{localcatalog.KindCustomTypes},
		validateCustomTypeSemantic,
	)
}