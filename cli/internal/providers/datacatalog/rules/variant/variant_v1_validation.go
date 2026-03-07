package variant

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ValidateVariantDiscriminatorsV1 checks all V1 variants' discriminators for:
//  1. Type validity — discriminator property must be string, integer, or boolean.
//     For custom type refs, the referenced custom type's own type is validated.
//  2. Ownership — discriminator must reference a property in the parent's own properties.
func ValidateVariantDiscriminatorsV1(
	variants localcatalog.VariantsV1,
	ownPropertyRefs []string,
	basePath string,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, v := range variants {
		discriminatorPath := fmt.Sprintf("%s/variants/%d/discriminator", basePath, i)

		if !slices.Contains(ownPropertyRefs, v.Discriminator) {
			results = append(results, rules.ValidationResult{
				Reference: discriminatorPath,
				Message:   "discriminator must reference a property defined in the parent's own properties",
			})
		}

		results = append(
			results,
			validateDiscriminatorType(
				v.Discriminator,
				discriminatorPath,
				graph,
			)...)
	}

	return results
}
