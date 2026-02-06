package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// TypeConfigValidator validates config keywords for a specific data type
type TypeConfigValidator interface {
	// ConfigAllowed returns whether config is allowed for this type
	ConfigAllowed() bool

	// ValidateField validates a single config field
	// Returns:
	//   - (nil, nil) if field is valid
	//   - (nil, ErrFieldNotSupported) if field is not applicable to this type
	//   - ([]ValidationResult, nil) if field is applicable but has validation errors
	ValidateField(fieldname string, fieldval any) ([]rules.ValidationResult, error)

	// ValidateCrossFields validates relationships between config fields
	// Only runs if both field values are valid for this type
	ValidateCrossFields(config map[string]any) []rules.ValidationResult
}

// Sentinel errors for field validation
var (
	ErrFieldNotSupported = errors.New("field not supported for this type")
	ErrFieldInvalid      = errors.New("field value is invalid")
)

// ValidateConfig validates config map for given type(s)
// Implements union semantics for field validation and strict semantics for cross-field
func ValidateConfig(types []string, config map[string]any, reference string) []rules.ValidationResult {

	var validators []TypeConfigValidator
	for _, typeName := range types {
		validator := getValidatorForType(typeName)

		if validator == nil {
			// If we have a type for which a validator
			// is not found, we skip it as the other syntax validations
			// from where the type is being used will raise an error
			continue
		}
		validators = append(validators, validator)
	}

	if len(validators) == 0 {
		return nil
	}

	// Initial check simply verifies that the
	// config object is allowed based on the type
	// or not
	if len(config) > 0 {
		allDisallow := true

		for _, validator := range validators {
			if validator.ConfigAllowed() {
				allDisallow = false
				break
			}
		}

		if allDisallow {
			return []rules.ValidationResult{{
				Reference: reference,
				Message:   "config is not allowed for the specified type(s)",
			}}
		}
	}

	var results []rules.ValidationResult

	for fieldName, fieldVal := range config {
		fieldResults := validateFieldUnion(
			validators,
			fieldName,
			fieldVal,
			reference,
		)
		results = append(results, fieldResults...)
	}

	crossResults := validateCrossFieldsWithDedup(validators, config, reference)
	results = append(results, crossResults...)

	return results
}

// validateFieldUnion implements union semantics for field validation
// where in field is valid if ANY validator accepts it
func validateFieldUnion(
	validators []TypeConfigValidator,
	fieldName string,
	fieldVal any,
	baseRef string,
) []rules.ValidationResult {
	var (
		allNotSupported  = true
		collectedResults []rules.ValidationResult
	)

	for _, validator := range validators {
		results, err := validator.ValidateField(fieldName, fieldVal)

		if err == nil && len(results) == 0 {
			return nil
		}

		if errors.Is(err, ErrFieldNotSupported) {
			continue
		}

		allNotSupported = false

		for i := range results {
			if !strings.HasPrefix(results[i].Reference, baseRef) {
				results[i].Reference = fmt.Sprintf("%s/%s", baseRef, results[i].Reference)
			}
		}

		collectedResults = append(collectedResults, results...)
	}

	if allNotSupported {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("%s/%s", baseRef, fieldName),
			Message:   fmt.Sprintf("'%s' is not applicable for type(s)", fieldName),
		}}
	}

	return collectedResults
}

// validateCrossFieldsWithDedup implements strict semantics with deduplication
func validateCrossFieldsWithDedup(validators []TypeConfigValidator, config map[string]any, baseRef string) []rules.ValidationResult {
	type errorKey struct {
		Reference string
		Message   string
	}

	errorMap := make(map[errorKey]rules.ValidationResult)

	for _, validator := range validators {
		crossResults := validator.ValidateCrossFields(config)

		for _, result := range crossResults {
			result.Reference = fmt.Sprintf("%s/%s", baseRef, result.Reference)

			key := errorKey{
				Reference: result.Reference,
				Message:   result.Message,
			}

			errorMap[key] = result
		}
	}

	var results []rules.ValidationResult

	for _, result := range errorMap {
		results = append(results, result)
	}

	return results
}

// getValidatorForType returns validator for given type name
func getValidatorForType(typeName string) TypeConfigValidator {
	switch typeName {
	case "string":
		return &StringTypeConfig{}
	case "integer":
		return &IntegerTypeConfig{}
	case "number":
		return &NumberTypeConfig{}
	case "array":
		return &ArrayTypeConfig{}
	case "object":
		return &ObjectTypeConfig{}
	case "boolean":
		return &BooleanTypeConfig{}
	case "null":
		return &NullTypeConfig{}
	default:
		// It could also be a custom type reference
		if customTypeLegacyReferenceRegex.MatchString(typeName) {
			return &CustomTypeConfig{}
		}
		return nil
	}
}
