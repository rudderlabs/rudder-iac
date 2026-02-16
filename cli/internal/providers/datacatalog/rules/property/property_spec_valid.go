package property

import (
	"fmt"
	"regexp"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	drules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

var customTypeLegacyRefRegex = regexp.MustCompile(drules.CustomTypeLegacyReferenceRegex)

var examples = rules.Examples{
	Valid: []string{
		`properties:
  - id: user_id
    name: User ID
    description: Unique identifier for the user
    type: string
  - id: email
    name: Email
    type: string`,
	},
	Invalid: []string{
		`properties:
  - name: Missing ID
    type: string`,
		`properties:
  - id: user_id
    # Missing required name field
    type: string`,
		`properties:
  - id: user_id
    name: ""
    # Name cannot be empty
    type: string`,
		`properties:
  - id: user_id
    name: This is a very long name that exceeds the maximum allowed length of sixty five characters for a property name
    # Name exceeds 65 characters
    type: string`,
	},
}

var validatePropertySpec = func(_ string, _ string, _ map[string]any, Spec localcatalog.PropertySpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(Spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/properties",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, nil)

	// validate the type field on the property
	// which can be a custom type reference or a comma-separated
	// list of primitive types
	for i, prop := range Spec.Properties {
		if prop.Type == "" {
			continue
		}

		if strings.HasPrefix(prop.Type, "#") {
			if !customTypeLegacyRefRegex.MatchString(prop.Type) {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/properties/%d/type", i),
					Message:   fmt.Sprintf("'%s' is invalid: must be of pattern #/custom-types/<group>/<id>", prop.Type),
				})
			}
			continue
		}

		typs := lo.Map(strings.Split(prop.Type, ","), func(item string, _ int) string {
			return strings.TrimSpace(item)
		})

		if !lo.Every(drules.ValidPrimitiveTypes, typs) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/type", i),
				Message:   fmt.Sprintf("'%s' is not a valid primitive type: must be unique one of [%s]", prop.Type, strings.Join(drules.ValidPrimitiveTypes, ", ")),
			})
			continue
		}

		if len(lo.Uniq(typs)) != len(typs) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/type", i),
				Message:   fmt.Sprintf("'%s' is not a valid primitive type: must be unique one of [%s]", prop.Type, strings.Join(drules.ValidPrimitiveTypes, ", ")),
			})
		}
	}

	return results
}

func NewPropertySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/properties/spec-syntax-valid",
		rules.Error,
		"property spec syntax must be valid",
		examples,
		[]string{"properties"},
		validatePropertySpec,
	)
}
