package variant

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ValidateVariantSemanticV1 runs all semantic checks on V1 variants:
//  1. Discriminator type validity — must be string, integer, or boolean.
//  2. Discriminator ownership — must reference a property in the parent's own properties.
//  3. Duplicate property refs — within each case and default property list.
func ValidateVariantSemanticV1(
	variants localcatalog.VariantsV1,
	ownPropertyRefs []string,
	basePath string,
	graph *resources.Graph,
) []rules.ValidationResult {
	var results []rules.ValidationResult

	for i, vt := range variants {
		discriminatorPath := fmt.Sprintf("%s/variants/%d/discriminator", basePath, i)

		if !slices.Contains(ownPropertyRefs, vt.Discriminator) {
			results = append(results, rules.ValidationResult{
				Reference: discriminatorPath,
				Message:   "discriminator must reference a property defined in the parent's own properties",
			})
		}

		results = append(
			results,
			validateDiscriminatorType(
				vt.Discriminator,
				discriminatorPath,
				graph,
			)...)

		for c, vCase := range vt.Cases {
			caseRef := fmt.Sprintf("%s/variants/%d/cases/%d/properties", basePath, i, c)
			results = append(
				results,
				checkDuplicatePropertyRefsV1(vCase.Properties, caseRef)...)
		}

		defaultRef := fmt.Sprintf("%s/variants/%d/default/properties", basePath, i)
		results = append(
			results,
			checkDuplicatePropertyRefsV1(vt.Default.Properties, defaultRef)...)
	}

	return results
}

// checkDuplicatePropertyRefsV1 dedupes a flat list of V1 variant property refs
// and emits one error per occurrence of any ref that appears more than once.
func checkDuplicatePropertyRefsV1(props []localcatalog.PropertyReferenceV1, parentRef string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, prop := range props {
		counts[prop.Property]++
	}

	var results []rules.ValidationResult
	for i, prop := range props {
		if counts[prop.Property] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d/property", parentRef, i),
				Message:   "duplicate property reference in tracking plan event rule",
			})
		}
	}
	return results
}
