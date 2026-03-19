package customtype

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

const (
	customTypeNameRegexPattern = "^[A-Z][A-Za-z0-9_-]*$"
	customTypeNameRegexTag     = "custom_type_name"
	customTypeNameErrorMessage = "must start with uppercase and contain only alphanumeric, underscores, or hyphens"
	customTypeTypeRegexTag     = "primitive_type"
)

var (
	customTypeTypeRegexPattern = fmt.Sprintf("^(%s)$", strings.Join(catalogRules.ValidPrimitiveTypes, "|"))
	customTypeTypeErrorMessage = fmt.Sprintf("must be one of the following: %s", strings.Join(catalogRules.ValidPrimitiveTypes, ", "))
	customTypeRefRegex         = regexp.MustCompile(catalogRules.CustomTypeReferenceRegex)
)

func init() {
	// Register the custom type name pattern for use with validate:"pattern=custom_type_name"
	funcs.NewPattern(
		customTypeNameRegexTag,
		customTypeNameRegexPattern,
		customTypeNameErrorMessage,
	)

	// Register the custom type type pattern for use with validate:"pattern=primitive_type"
	funcs.NewPattern(
		customTypeTypeRegexTag,
		customTypeTypeRegexPattern,
		customTypeTypeErrorMessage,
	)
}

var examples = rules.Examples{
	Valid: []string{
		`types:
  - id: address
    name: Address
    description: Physical address structure
    type: object
  - id: user_status
    name: User Status
    type: string`,
	},
	Invalid: []string{
		`types:
  - name: Missing ID
    type: string`,
		`types:
  - id: user_status
    # Missing required name field
    type: string`,
		`types:
  - id: status
    name: Status
    # Missing required type field`,
	},
}

// Main validation function for custom type spec
// which delegates the validation to the go-validator
// through struct tags.
var validateCustomTypeSpec = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CustomTypeSpec) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(Spec, "")

	// If any error on running the validate struct command
	// we report that at top `/types` layer
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/types",
				Message:   err.Error(),
			},
		}
	}

	return funcs.ParseValidationErrors(
		validationErrors,
		reflect.TypeOf(Spec),
	)
}

var validateCustomTypeSpecV1 = func(Kind string, Version string, Metadata map[string]any, Spec localcatalog.CustomTypeSpecV1) []rules.ValidationResult {
	validationErrors, err := rules.ValidateStruct(Spec, "")
	if err != nil {
		return []rules.ValidationResult{
			{
				Reference: "/types",
				Message:   err.Error(),
			},
		}
	}

	results := funcs.ParseValidationErrors(
		validationErrors,
		reflect.TypeOf(Spec),
	)

	results = append(results, validateItemTypes(Spec.Types)...)
	return results
}

func validateItemTypes(types []localcatalog.CustomTypeV1) []rules.ValidationResult {
	results := []rules.ValidationResult{}
	for i, ct := range types {
		if ct.ItemType != "" && !isValidV1TypeOrCustomTypeRef(ct.ItemType) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/item_type", i),
				Message: fmt.Sprintf("'item_type' is invalid: must be one of [%s] or of pattern #custom-type:<id>",
					strings.Join(catalogRules.ValidPrimitiveTypes, ", "),
				),
			})
		}

		if hasDuplicateValues(ct.ItemTypes) {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/types/%d/item_types", i),
				Message: fmt.Sprintf("'item_types' is invalid: must be unique one of [%s]",
					strings.Join(catalogRules.ValidPrimitiveTypes, ", "),
				),
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
	if lo.Contains(catalogRules.ValidPrimitiveTypes, typeValue) {
		return true
	}

	return customTypeRefRegex.MatchString(typeValue)
}

func NewCustomTypeSpecSyntaxValidRule() rules.Rule {
	return prules.NewTypedRule(
		"datacatalog/custom-types/spec-syntax-valid",
		rules.Error,
		"custom type spec syntax must be valid",
		examples,
		prules.NewPatternValidator(
			prules.LegacyVersionPatterns(localcatalog.KindCustomTypes),
			validateCustomTypeSpec,
		),
		prules.NewPatternValidator(
			prules.V1VersionPatterns(localcatalog.KindCustomTypes),
			validateCustomTypeSpecV1,
		),
	)
}
