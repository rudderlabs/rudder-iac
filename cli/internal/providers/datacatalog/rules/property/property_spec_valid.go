package property

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	drules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

var (
	customTypeLegacyRefRegex = regexp.MustCompile(drules.CustomTypeLegacyReferenceRegex)
	customTypeRefRegex       = regexp.MustCompile(drules.CustomTypeReferenceRegex)
)

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

var validatePropertySpecV1 = func(_ string, _ string, _ map[string]any, spec localcatalog.PropertySpecV1) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(spec, "")
	if err != nil {
		return []rules.ValidationResult{{
			Reference: "/properties",
			Message:   err.Error(),
		}}
	}

	results := funcs.ParseValidationErrors(validationErrors, reflect.TypeOf(spec))

	for i, property := range spec.Properties {
		if strings.TrimSpace(property.Name) != property.Name {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/name", i),
				Message:   "'name' must not have leading or trailing whitespace",
			})
		}

		if hasDuplicateValues(property.Types) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/types", i),
				Message:   "'types' must not contain duplicate values",
			})
		}

		if property.Type != "" && !isValidV1TypeOrCustomTypeRef(property.Type) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/type", i),
				Message: fmt.Sprintf("'%s' is invalid: must be one of [%s] or of pattern #custom-type:<id>",
					property.Type,
					strings.Join(drules.ValidPrimitiveTypes, ", "),
				),
			})
		}

		if property.ItemType != "" && !isValidV1TypeOrCustomTypeRef(property.ItemType) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/item_type", i),
				Message: fmt.Sprintf("'%s' is invalid: must be one of [%s] or of pattern #custom-type:<id>",
					property.ItemType,
					strings.Join(drules.ValidPrimitiveTypes, ", "),
				),
			})
		}

		if hasDuplicateValues(property.ItemTypes) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/properties/%d/item_types", i),
				Message:   "'item_types' must not contain duplicate values",
			})
		}
	}

	return results
}

func hasDuplicateValues(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, found := seen[value]; found {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func isValidV1TypeOrCustomTypeRef(typeValue string) bool {
	if lo.Contains(drules.ValidPrimitiveTypes, typeValue) {
		return true
	}

	return customTypeRefRegex.MatchString(typeValue)
}

func NewPropertySpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/properties/spec-syntax-valid",
		rules.Error,
		"property spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindProperties),
			validatePropertySpec,
		),
		prules.NewPatternValidator(
			[]rules.MatchPattern{
				rules.MatchKindVersion(localcatalog.KindProperties, specs.SpecVersionV1),
			},
			validatePropertySpecV1,
		),
	)
}
