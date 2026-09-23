package variant

import (
	"fmt"
	"slices"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

// validDiscriminatorTypes are the primitive types allowed for discriminator properties.
var validDiscriminatorTypes = []string{"string", "integer", "boolean"}

// ValidateVariantSemanticV0 runs all semantic checks on V0 variants:
//  1. Discriminator type validity — must be string, integer, or boolean.
//  2. Discriminator ownership — must reference a property in the parent's own properties.
//  3. Duplicate property refs — within each case and default property list.
func ValidateVariantSemanticV0(
	variants localcatalog.Variants,
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
				checkDuplicatePropertyRefsV0(vCase.Properties, caseRef)...)
		}

		defaultRef := fmt.Sprintf("%s/variants/%d/default", basePath, i)
		results = append(
			results,
			checkDuplicatePropertyRefsV0(vt.Default, defaultRef)...)
	}

	return results
}

// validateDiscriminatorType looks up the discriminator property in the graph
// and checks that its type is string, integer, or boolean.
func validateDiscriminatorType(discriminator, jsonPointer string, graph *resources.Graph) []rules.ValidationResult {
	resourceType, localID, err := funcs.ParseURNRef(discriminator)
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
		if !ok || isAllowedDiscriminatorType(ctType) {
			return nil
		}

		return []rules.ValidationResult{{
			Reference: jsonPointer,
			Message:   fmt.Sprintf("discriminator references custom type with type '%s' which is invalid, must be one of: string, integer, boolean", ctType),
		}}
	default:
		return nil
	}
}

// checkDuplicatePropertyRefsV0 dedupes a flat list of V0 variant property refs
// and emits one error per occurrence of any ref that appears more than once.
func checkDuplicatePropertyRefsV0(props []localcatalog.PropertyReference, parentRef string) []rules.ValidationResult {
	counts := make(map[string]int)
	for _, prop := range props {
		counts[prop.Ref]++
	}

	var results []rules.ValidationResult
	for i, prop := range props {
		if counts[prop.Ref] > 1 {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("%s/%d/$ref", parentRef, i),
				Message:   "duplicate property reference in tracking plan event rule",
			})
		}
	}
	return results
}

func isAllowedDiscriminatorType(input string) bool {
	rawTypes := lo.Map(strings.Split(input, ","), func(item string, _ int) string {
		return strings.TrimSpace(item)
	})
	for _, allowedType := range validDiscriminatorTypes {
		if lo.Contains(rawTypes, allowedType) {
			return true
		}
	}
	return false
}
