package config

import (
	"errors"
	"fmt"

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
)

// ValidateConfig validates config map for given type(s).
// Implements union semantics for field validation and strict semantics for cross-field validation.
// validatorOverrides allows callers to inject context-specific validators for specific types;
// pass nil to use the default validators for all types.
func ValidateConfig(types []string, config map[string]any, reference string, validatorOverrides map[string]TypeConfigValidator) []rules.ValidationResult {
	return ValidateConfigWithOptions(
		types,
		config,
		reference,
		WithFieldAliases(V0FieldAliases()),
		WithValidatorOverrides(validatorOverrides),
		WithCustomTypeRefMatcher(LegacyCustomTypeRefMatcher),
	)
}

type normalizedField struct {
	Raw      string
	Keyword  ConfigKeyword
	Value    any
	Resolved bool
}

// ValidateConfigWithOptions validates config using explicit field aliasing and custom type matching rules.
func ValidateConfigWithOptions(
	types []string,
	config map[string]any,
	reference string,
	opts ...ValidateConfigOption,
) []rules.ValidationResult {

	if len(config) == 0 {
		return nil
	}

	options := newValidateConfigOptions(opts...)

	var validators []TypeConfigValidator
	for _, typeName := range types {
		validator := getDefaultValidatorForType(typeName, options)
		if v, ok := options.validatorOverrides[typeName]; ok {
			validator = v
		}

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

	var results []rules.ValidationResult

	fields, normalizedConfig, normalizedFieldNames := normalizeConfig(config, options)

	for _, field := range fields {
		fieldResults := validateFieldUnion(
			validators,
			field,
			reference,
		)
		results = append(results, fieldResults...)
	}

	crossResults := validateCrossFieldsWithDedup(
		validators,
		normalizedConfig,
		normalizedFieldNames,
		reference,
	)
	results = append(results, crossResults...)

	return results
}

func normalizeConfig(
	config map[string]any,
	options validateConfigOptions,
) ([]normalizedField, map[string]any, map[ConfigKeyword]string) {
	fields := make([]normalizedField, 0, len(config))
	normalized := make(map[string]any, len(config))
	fieldNames := make(map[ConfigKeyword]string, len(config))

	for rawField, value := range config {
		keyword, ok := options.fieldAliases[rawField]
		field := normalizedField{
			Raw:      rawField,
			Keyword:  keyword,
			Value:    value,
			Resolved: ok,
		}
		fields = append(fields, field)

		if !ok {
			continue
		}

		normalized[validatorFieldName(keyword)] = value
		fieldNames[keyword] = rawField
	}

	return fields, normalized, fieldNames
}

// validateFieldUnion implements union semantics for field validation
// where in field is valid if ANY validator accepts it
func validateFieldUnion(
	validators []TypeConfigValidator,
	field normalizedField,
	baseRef string,
) []rules.ValidationResult {
	var (
		allNotSupported  = true
		collectedResults []rules.ValidationResult
	)

	if !field.Resolved {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("%s/%s", baseRef, field.Raw),
			Message:   fmt.Sprintf("'%s' is not applicable for type(s)", field.Raw),
		}}
	}

	canonicalField := validatorFieldName(field.Keyword)

	for _, validator := range validators {
		results, err := validator.ValidateField(canonicalField, field.Value)

		if err == nil && len(results) == 0 {
			return nil
		}

		if errors.Is(err, ErrFieldNotSupported) {
			continue
		}

		allNotSupported = false

		results = rewriteResultsToRawKey(results, canonicalField, field.Raw)
		collectedResults = append(collectedResults, results...)
	}

	if allNotSupported {
		return []rules.ValidationResult{{
			Reference: fmt.Sprintf("%s/%s", baseRef, field.Raw),
			Message:   fmt.Sprintf("'%s' is not applicable for type(s)", field.Raw),
		}}
	}

	for i := range collectedResults {
		collectedResults[i].Reference = joinReference(baseRef, collectedResults[i].Reference)
	}

	return dedup(collectedResults)
}

// validateCrossFieldsWithDedup implements strict semantics with deduplication
func validateCrossFieldsWithDedup(
	validators []TypeConfigValidator,
	config map[string]any,
	fieldNames map[ConfigKeyword]string,
	baseRef string,
) []rules.ValidationResult {
	var collectedResults []rules.ValidationResult

	for _, validator := range validators {
		crossResults := validator.ValidateCrossFields(config)
		crossResults = rewriteCrossFieldResults(crossResults, fieldNames)

		for i := range crossResults {
			crossResults[i].Reference = joinReference(baseRef, crossResults[i].Reference)
		}

		collectedResults = append(collectedResults, crossResults...)
	}

	return dedup(collectedResults)
}

// dedup removes duplicate validation results based on reference and message
func dedup(results []rules.ValidationResult) []rules.ValidationResult {
	type errorKey struct {
		Reference string
		Message   string
	}

	seen := make(map[errorKey]struct{}, len(results))
	deduplicated := make([]rules.ValidationResult, 0, len(results))

	for _, result := range results {
		key := errorKey{Reference: result.Reference, Message: result.Message}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduplicated = append(deduplicated, result)
	}

	return deduplicated
}

// getDefaultValidatorForType returns validator for given type name
func getDefaultValidatorForType(typeName string, options validateConfigOptions) TypeConfigValidator {
	switch typeName {
	case "string":
		return &StringTypeConfig{}
	case "integer":
		return &IntegerTypeConfig{}
	case "number":
		return &NumberTypeConfig{}
	case "array":
		return &ArrayTypeConfig{isCustomTypeRef: options.customTypeRefMatcher}
	case "object":
		return &ObjectTypeConfig{}
	case "boolean":
		return &BooleanTypeConfig{}
	case "null":
		return &NullTypeConfig{}
	default:
		// It could also be a custom type reference
		if options.customTypeRefMatcher != nil && options.customTypeRefMatcher(typeName) {
			return &CustomTypeConfig{}
		}
		return nil
	}
}
