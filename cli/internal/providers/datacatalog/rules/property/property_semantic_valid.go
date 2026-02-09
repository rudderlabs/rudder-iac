package property

import (
	"fmt"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var validatePropertySemantic = func(_ string, _ string, _ map[string]any, spec localcatalog.PropertySpec, graph *resources.Graph) []rules.ValidationResult {
	// Generic ref validation for pattern=legacy_* tagged fields
	results := funcs.ValidateReferences(spec, graph)

	// Property-specific: Type field may contain a custom type reference
	// that needs graph lookup. The generic walker skips it because
	// it has no pattern=legacy_* tag (format validated in syntactic rule).
	for i, prop := range spec.Properties {
		if !strings.HasPrefix(prop.Type, "#") {
			continue
		}

		resourceType, localID, err := funcs.ParseURNRef(prop.Type)
		if err != nil {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/type", i),
				Message:   fmt.Sprintf("failed to parse reference '%s': %v", prop.Type, err),
			})
			continue
		}

		urn := resources.URN(localID, resourceType)
		if _, exists := graph.GetResource(urn); !exists {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/type", i),
				Message:   fmt.Sprintf("referenced %s '%s' not found in resource graph", resourceType, localID),
			})
		}
	}

	return results
}

func NewPropertySemanticValidRule() rules.Rule {
	return prules.NewSemanticTypedRule(
		"datacatalog/properties/semantic-valid",
		rules.Error,
		"property references must resolve to existing resources",
		rules.Examples{},
		[]string{localcatalog.KindProperties},
		validatePropertySemantic,
	)
}
