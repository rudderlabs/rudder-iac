package customtype

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules/variant"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validateCustomTypeSemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	// Generic ref validation for pattern=legacy_* tagged fields
	results := funcs.ValidateReferences(spec, graph)

	// Legacy config itemTypes reference validation remains in-place.
	results = append(results, validateConfigItemTypesV0(spec, graph)...)

	// Variant discriminator type + ownership checks (shared module)
	results = append(results, validateCustomTypeVariants(spec, graph)...)

	// Name uniqueness across the entire resource graph
	results = append(results, validateCustomTypeNameUniqueness(spec, graph)...)

	return results
}

var validateCustomTypeSemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.CustomTypeSpecV1, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)
	results = append(results, validateItemTypeRefsV1(spec, graph)...)
	results = append(results, validateCustomTypeVariantsV1(spec, graph)...)
	results = append(results, validateCustomTypeNameUniquenessV1(spec, graph)...)
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
		results = append(results, variant.ValidateVariantSemanticV0(
			ct.Variants, ownRefs, fmt.Sprintf("/types/%d", i), graph,
		)...)
	}

	return results
}

func validateCustomTypeVariantsV1(spec localcatalog.CustomTypeSpecV1, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, ct := range spec.Types {
		ownRefs := make([]string, len(ct.Properties))
		for j, prop := range ct.Properties {
			ownRefs[j] = prop.Property
		}

		results = append(results, variant.ValidateVariantSemanticV1(
			ct.Variants,
			ownRefs,
			fmt.Sprintf("/types/%d", i),
			graph,
		)...)
	}

	return results
}

// buildCustomTypeNameCountMap builds a map of custom type
// names to their count in the resource graph.
func buildCustomTypeNameCountMap(graph *resources.Graph) map[string]int {
	countMap := make(map[string]int)
	for _, resource := range graph.ResourcesByType(types.CustomTypeResourceType) {
		data := resource.Data()
		name, _ := data["name"].(string)
		countMap[name]++
	}
	return countMap
}

// validateCustomTypeNameUniqueness checks that each custom type's name is unique
// across the entire resource graph.
func validateCustomTypeNameUniqueness(spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	countMap := buildCustomTypeNameCountMap(graph)

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

func validateCustomTypeNameUniquenessV1(spec localcatalog.CustomTypeSpecV1, graph *resources.Graph) []rules.ValidationResult {
	countMap := buildCustomTypeNameCountMap(graph)

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

// validateItemTypeRefsV1 checks custom type references in the top-level ItemType field.
func validateItemTypeRefsV1(spec localcatalog.CustomTypeSpecV1, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, ct := range spec.Types {
		if strings.HasPrefix(ct.ItemType, "#") {
			results = append(
				results,
				checkRef(ct.ItemType, fmt.Sprintf("/types/%d/item_type", i), graph)...,
			)
		}
	}

	return results
}

func checkRef(ref, jsonPointer string, graph *resources.Graph) []rules.ValidationResult {
	resourceType, localID, err := funcs.ParseURNRef(ref)
	if err != nil {
		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("'%s' is invalid: must be of pattern #custom-type:<id>", ref),
		}}
	}

	urn := resources.URN(localID, resourceType)
	if _, exists := graph.GetResource(urn); !exists {
		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("referenced %s '%s' not found in resource graph", resourceType, localID),
		}}
	}

	return nil
}

// validateConfigItemTypesV0 checks custom type references in Config["itemTypes"].
func validateConfigItemTypesV0(spec localcatalog.CustomTypeSpec, graph *resources.Graph) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, ct := range spec.Types {
		results = append(
			results,
			validateConfigItemTypes(ct.Config, "itemTypes", fmt.Sprintf("/types/%d/config/itemTypes", i), graph)...,
		)
	}

	return results
}

// validateConfigItemTypes checks custom type references in the provided config key.
// These may contain URN-format refs (e.g., "#custom-type:Address") that must
// exist in the resource graph.
func validateConfigItemTypes(config map[string]any, configKey, refBase string, graph *resources.Graph) []rules.ValidationResult {
	if config == nil {
		return nil
	}

	itemTypes, ok := config[configKey]
	if !ok {
		return nil
	}

	items, ok := itemTypes.([]any)
	if !ok {
		return nil
	}

	var results []rules.ValidationResult
	for i, item := range items {
		ref, ok := item.(string)
		if !ok || !strings.HasPrefix(ref, "#") {
			continue
		}

		resourceType, localID, err := funcs.ParseURNRef(ref)
		if err != nil {
			continue
		}

		urn := resources.URN(localID, resourceType)
		if _, exists := graph.GetResource(urn); !exists {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d", refBase, i),
				Message:   fmt.Sprintf("referenced %s '%s' not found in resource graph", resourceType, localID),
			})
		}
	}

	return results
}

func NewCustomTypeSemanticValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/custom-types/semantic-valid",
		rules.Error,
		"custom type references must resolve to existing resources",
		rules.Examples{},
		prules.NewSemanticPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindCustomTypes),
			validateCustomTypeSemantic,
		),
		prules.NewSemanticPatternValidator(
			prules.V1VersionPatterns(localcatalog.KindCustomTypes),
			validateCustomTypeSemanticV1,
		),
	)
}
