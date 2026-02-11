package funcs

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

// validDiscriminatorTypes are the primitive types allowed for discriminator properties.
var validDiscriminatorTypes = []string{"string", "integer", "boolean"}

// ValidateVariantDiscriminators checks all variants' discriminators for:
//  1. Type validity — discriminator property must be string, integer, or boolean.
//     For custom type refs, the referenced custom type's own type is validated.
//  2. Ownership — discriminator must reference a property in the parent's own properties.
func ValidateVariantDiscriminators(
	variants localcatalog.Variants,
	ownPropertyRefs []string,
	basePath string,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, variant := range variants {
		discriminatorPath := fmt.Sprintf("%s/variants/%d/discriminator", basePath, i)

		// discriminator must be in the parent's own properties
		if !slices.Contains(ownPropertyRefs, variant.Discriminator) {
			results = append(results, rules.ValidationResult{
				Reference: discriminatorPath,
				Message:   "discriminator must reference a property defined in the parent's own properties",
			})
		}

		results = append(
			results,
			validateDiscriminatorType(
				variant.Discriminator,
				discriminatorPath,
				graph,
			)...)
	}

	return results
}

// validateDiscriminatorType looks up the discriminator property in the graph
// and checks that its type is string, integer, or boolean.
func validateDiscriminatorType(discriminator, jsonPointer string, graph *resources.Graph) []rules.ValidationResult {
	resourceType, localID, err := ParseURNRef(discriminator)
	if err != nil {
		// Parse errors are already reported by ValidateReferences
		return nil
	}

	urn := resources.URN(localID, resourceType)
	resource, exists := graph.GetResource(urn)
	if !exists {
		// the next part of the check
		// can only by performed if we
		// have valid reference of the resource
		return nil
	}

	propType := resource.Data()["type"]

	switch t := propType.(type) {
	case string:
		inputTypes := lo.Map(strings.Split(t, ","), func(item string, _ int) string {
			return strings.TrimSpace(item)
		})

		for _, validType := range validDiscriminatorTypes {
			if lo.Contains(inputTypes, validType) {
				return nil
			}
		}
		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("discriminator property type '%s' must contain one of: string, integer, boolean", t),
		}}

	case resources.PropertyRef:
		// Resolve the custom type from the graph and validate its type
		ctResource, exists := graph.GetResource(t.URN)
		if !exists {
			// we return nil here because this part
			// of the validation is handled by the validate
			// references and we only do the checking
			// if the customtype is found.
			return nil
		}

		ctType, ok := ctResource.Data()["type"].(string)
		if !ok {
			return nil
		}
		ctTypes := lo.Map(strings.Split(ctType, ","), func(item string, _ int) string {
			return strings.TrimSpace(item)
		})

		for _, vt := range validDiscriminatorTypes {
			if lo.Contains(ctTypes, vt) {
				return nil
			}
		}

		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("discriminator references custom type with type '%s' which is invalid, must be one of: string, integer, boolean", ctType),
		}}
	default:
		return nil
	}
}
