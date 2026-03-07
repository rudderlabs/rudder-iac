package customtype

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
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

	// Legacy config itemTypes reference validation remains in-place.
	results = append(results, validateConfigItemTypes(spec, graph)...)

	// Variant discriminator type + ownership checks (shared module)
	results = append(results, validateCustomTypeVariants(spec, graph)...)

	// Name uniqueness across the entire resource graph
	results = append(results, validateCustomTypeNameUniqueness(spec, graph)...)

	return results
}

var validateCustomTypeSemanticV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.CustomTypeSpecV1, graph *resources.Graph) []rules.ValidationResult {
	results := funcs.ValidateReferences(spec, graph)
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
		results = append(results, funcs.ValidateVariantDiscriminators(
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

		results = append(results, validateVariantDiscriminatorsV1(
			ct.Variants,
			ownRefs,
			fmt.Sprintf("/types/%d", i),
			graph,
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

func validateCustomTypeNameUniquenessV1(spec localcatalog.CustomTypeSpecV1, graph *resources.Graph) []rules.ValidationResult {
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

func validateVariantDiscriminatorsV1(
	variants localcatalog.VariantsV1,
	ownPropertyRefs []string,
	basePath string,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, variant := range variants {
		discriminatorPath := fmt.Sprintf("%s/variants/%d/discriminator", basePath, i)
		if !slices.Contains(ownPropertyRefs, variant.Discriminator) {
			results = append(results, rules.ValidationResult{
				Reference: discriminatorPath,
				Message:   "discriminator must reference a property defined in the parent's own properties",
			})
		}

		results = append(results, validateDiscriminatorTypeV1(
			variant.Discriminator,
			discriminatorPath,
			graph,
		)...)
	}

	return results
}

func validateDiscriminatorTypeV1(discriminator, jsonPointer string, graph *resources.Graph) []rules.ValidationResult {
	resourceType, localID, err := funcs.ParseURNRef(discriminator)
	if err != nil {
		return nil
	}

	urn := resources.URN(localID, resourceType)
	resource, exists := graph.GetResource(urn)
	if !exists {
		return nil
	}

	switch propType := resource.Data()["type"].(type) {
	case string:
		if isAllowedDiscriminatorType(propType) {
			return nil
		}

		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("discriminator property type '%s' must contain one of: string, integer, boolean", propType),
		}}

	case resources.PropertyRef:
		customTypeResource, exists := graph.GetResource(propType.URN)
		if !exists {
			return nil
		}

		customType, ok := customTypeResource.Data()["type"].(string)
		if !ok || isAllowedDiscriminatorType(customType) {
			return nil
		}

		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("discriminator references custom type with type '%s' which is invalid, must be one of: string, integer, boolean", customType),
		}}
	default:
		return nil
	}
}

func isAllowedDiscriminatorType(input string) bool {
	for _, allowedType := range []string{"string", "integer", "boolean"} {
		if slices.Contains(splitTypes(input), allowedType) {
			return true
		}
	}

	return false
}

func splitTypes(input string) []string {
	rawTypes := strings.Split(input, ",")
	for i := range rawTypes {
		rawTypes[i] = strings.TrimSpace(rawTypes[i])
	}
	return rawTypes
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
			[]rules.MatchPattern{
				rules.MatchKindVersion(localcatalog.KindCustomTypes, specs.SpecVersionV1),
			},
			validateCustomTypeSemanticV1,
		),
	)
}
