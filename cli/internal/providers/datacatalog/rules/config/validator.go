package config

import (
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// TypeConfigValidator validates config keywords for a specific data type.
type TypeConfigValidator interface {
	// ConfigAllowed returns whether config is allowed for this type.
	ConfigAllowed() bool

	// ValidateField validates a single config field.
	// rawKey is the original user-supplied key; keyword is the resolved ConfigKeyword.
	// Use keyword for matching logic and rawKey for Reference/Message output.
	// Returns:
	//   - (nil, nil) if field is valid
	//   - (nil, ErrFieldNotSupported) if field is not applicable to this type
	//   - ([]ValidationResult, nil) if field is applicable but has validation errors
	ValidateField(rawKey string, keyword ConfigKeyword, fieldval any) ([]rules.ValidationResult, error)

	// ValidateCrossFields validates relationships between config fields.
	// config contains only the resolved (keyword -> value) pairs.
	ValidateCrossFields(config map[ConfigKeyword]any) []rules.ValidationResult
}

// Sentinel errors for field validation.
var (
	ErrFieldNotSupported = errors.New("field not supported for this type")
)

// resolvedField holds a single config entry after alias resolution.
// Fields with no alias mapping are assigned KeywordUnknownField as a sentinel.
type resolvedField struct {
	RawKey  string
	Keyword ConfigKeyword
	Value   any
}

// resolveAliases maps every raw config key to a resolvedField.
// Keys with no alias entry are assigned KeywordUnknownField; they are excluded from
// crossFieldMap so cross-field validators never see them.
func resolveAliases(
	config map[string]any,
	aliases map[string]ConfigKeyword,
) (fields []resolvedField, crossFieldMap map[ConfigKeyword]any) {
	crossFieldMap = make(map[ConfigKeyword]any, len(config))

	for rawKey, val := range config {
		kw, ok := aliases[rawKey]
		if !ok {
			kw = KeywordUnknownField
		}

		fields = append(fields, resolvedField{
			RawKey:  rawKey,
			Keyword: kw,
			Value:   val,
		})

		if kw != KeywordUnknownField {
			crossFieldMap[kw] = val
		}
	}

	return fields, crossFieldMap
}

// ValidateConfig validates config map for given type(s) using V0 camelCase field names.
// This is the legacy entrypoint; it delegates to ValidateConfigWithOptions with an
// explicit V0 alias preset so backward compatibility is part of the API contract.
// validatorOverrides allows callers to inject context-specific validators for specific types;
// pass nil to use the default validators for all types.
func ValidateConfig(types []string, config map[string]any, reference string, validatorOverrides map[string]TypeConfigValidator) []rules.ValidationResult {
	return ValidateConfigWithOptions(
		types,
		config,
		reference,
		WithFieldAliases(V0FieldAliases),
		WithCustomTypeRefMatcher(legacyCustomTypeRefMatcher),
		WithValidatorOverrides(validatorOverrides),
	)
}

// ValidateConfigWithOptions validates config map for given type(s) using caller-supplied options.
// Use WithFieldAliases to supply a V0 or V1 alias preset, WithCustomTypeRefMatcher to configure
// custom type reference recognition, and WithValidatorOverrides for context-specific overrides.
func ValidateConfigWithOptions(types []string, config map[string]any, reference string, opts ...ValidateConfigOption) []rules.ValidationResult {
	if len(config) == 0 {
		return nil
	}

	o := newValidateConfigOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	var validators []TypeConfigValidator
	for _, typeName := range types {
		validator := getDefaultValidatorForType(typeName, *o)

		if v, ok := o.validatorOverrides[typeName]; ok {
			validator = v
		}

		if validator == nil {
			// If no validator is found for a type, defer to surrounding syntax validators.
			continue
		}
		validators = append(validators, validator)
	}

	if len(validators) == 0 {
		return nil
	}

	// Verify that config is allowed for at least one of the resolved types.
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

	fields, crossFieldMap := resolveAliases(config, o.fieldAliases)

	var results []rules.ValidationResult

	for _, rf := range fields {
		fieldResults := validateFieldUnion(
			validators,
			rf.RawKey,
			rf.Keyword,
			rf.Value,
			reference,
		)
		results = append(results, fieldResults...)
	}

	crossResults := validateCrossFieldsWithDedup(validators, crossFieldMap, reference)
	results = append(results, crossResults...)

	return results
}

// validateFieldUnion implements union semantics: a field is valid if ANY validator accepts it.
func validateFieldUnion(
	validators []TypeConfigValidator,
	rawKey string,
	keyword ConfigKeyword,
	fieldVal any,
	baseRef string,
) []rules.ValidationResult {
	var (
		allNotSupported  = true
		collectedResults []rules.ValidationResult
	)

	for _, validator := range validators {
		results, err := validator.ValidateField(rawKey, keyword, fieldVal)

		if err == nil && len(results) == 0 {
			return nil
		}

		if errors.Is(err, ErrFieldNotSupported) {
			continue
		}

		allNotSupported = false

		for i := range results {
			results[i].Reference = joinReference(baseRef, results[i].Reference)
		}

		collectedResults = append(collectedResults, results...)
	}

	if allNotSupported {
		return []rules.ValidationResult{{
			Reference: joinReference(baseRef, rawKey),
			Message:   fmt.Sprintf("'%s' is not applicable for type(s)", rawKey),
		}}
	}

	return dedup(collectedResults)
}

// validateCrossFieldsWithDedup implements strict cross-field semantics with deduplication.
func validateCrossFieldsWithDedup(
	validators []TypeConfigValidator,
	config map[ConfigKeyword]any,
	baseRef string,
) []rules.ValidationResult {
	var collectedResults []rules.ValidationResult

	for _, validator := range validators {
		crossResults := validator.ValidateCrossFields(config)

		for i := range crossResults {
			crossResults[i].Reference = joinReference(
				baseRef,
				crossResults[i].Reference,
			)
		}
		collectedResults = append(collectedResults, crossResults...)
	}

	return dedup(collectedResults)
}

// dedup removes duplicate validation results based on reference and message.
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

// getDefaultValidatorForType returns the default validator for a given type name.
// Built-in types are checked first; the custom type ref matcher is only consulted for unknowns.
func getDefaultValidatorForType(typeName string, opts validateConfigOptions) TypeConfigValidator {
	switch typeName {
	case "string":
		return &StringTypeConfig{}
	case "integer":
		return &IntegerTypeConfig{}
	case "number":
		return &NumberTypeConfig{}
	case "array":
		return &ArrayTypeConfig{isCustomTypeRef: opts.customTypeRefMatcher}
	case "object":
		return &ObjectTypeConfig{}
	case "boolean":
		return &BooleanTypeConfig{}
	case "null":
		return &NullTypeConfig{}
	default:
		if opts.customTypeRefMatcher != nil && opts.customTypeRefMatcher(typeName) {
			return &CustomTypeConfig{}
		}
		return nil
	}
}
