package config

import (
	"regexp"

	catalogRules "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/rules"
)

// ConfigKeyword is the version-independent logical keyword for a config field.
type ConfigKeyword string

const (
	KeywordEnum                 ConfigKeyword = "enum"
	KeywordMinimum              ConfigKeyword = "minimum"
	KeywordMaximum              ConfigKeyword = "maximum"
	KeywordPattern              ConfigKeyword = "pattern"
	KeywordFormat               ConfigKeyword = "format"
	KeywordMinLength            ConfigKeyword = "min_length"
	KeywordMaxLength            ConfigKeyword = "max_length"
	KeywordExclusiveMinimum     ConfigKeyword = "exclusive_minimum"
	KeywordExclusiveMaximum     ConfigKeyword = "exclusive_maximum"
	KeywordMultipleOf           ConfigKeyword = "multiple_of"
	KeywordItemTypes            ConfigKeyword = "item_types"
	KeywordMinItems             ConfigKeyword = "min_items"
	KeywordMaxItems             ConfigKeyword = "max_items"
	KeywordUniqueItems          ConfigKeyword = "unique_items"
	KeywordAdditionalProperties ConfigKeyword = "additional_properties"

	// KeywordUnknownField is a sentinel assigned to config keys that have no alias mapping.
	// Validators receiving this keyword must return ErrFieldNotSupported — they cannot
	// meaningfully validate a field they don't recognise. It is never added to crossFieldMap.
	KeywordUnknownField ConfigKeyword = "__unknown__"
)

// V0FieldAliases maps V0 camelCase raw keys to canonical ConfigKeyword values.
// Exposed so callers can compose or inspect the V0 preset.
var V0FieldAliases = map[string]ConfigKeyword{
	"enum":                 KeywordEnum,
	"minimum":              KeywordMinimum,
	"maximum":              KeywordMaximum,
	"pattern":              KeywordPattern,
	"format":               KeywordFormat,
	"minLength":            KeywordMinLength,
	"maxLength":            KeywordMaxLength,
	"exclusiveMinimum":     KeywordExclusiveMinimum,
	"exclusiveMaximum":     KeywordExclusiveMaximum,
	"multipleOf":           KeywordMultipleOf,
	"itemTypes":            KeywordItemTypes,
	"minItems":             KeywordMinItems,
	"maxItems":             KeywordMaxItems,
	"uniqueItems":          KeywordUniqueItems,
	"additionalProperties": KeywordAdditionalProperties,
}

// V1FieldAliases maps V1 snake_case raw keys to canonical ConfigKeyword values.
// Exposed so callers can compose or inspect the V1 preset.
var V1FieldAliases = map[string]ConfigKeyword{
	"enum":                  KeywordEnum,
	"minimum":               KeywordMinimum,
	"maximum":               KeywordMaximum,
	"pattern":               KeywordPattern,
	"format":                KeywordFormat,
	"min_length":            KeywordMinLength,
	"max_length":            KeywordMaxLength,
	"exclusive_minimum":     KeywordExclusiveMinimum,
	"exclusive_maximum":     KeywordExclusiveMaximum,
	"multiple_of":           KeywordMultipleOf,
	"min_items":             KeywordMinItems,
	"max_items":             KeywordMaxItems,
	"unique_items":          KeywordUniqueItems,
	"additional_properties": KeywordAdditionalProperties,
}

// validateConfigOptions holds the resolved options for a single ValidateConfigWithOptions call.
type validateConfigOptions struct {
	fieldAliases         map[string]ConfigKeyword
	customTypeRefMatcher func(string) bool
	validatorOverrides   map[string]TypeConfigValidator
}

func newValidateConfigOptions() *validateConfigOptions {
	return &validateConfigOptions{
		fieldAliases:         map[string]ConfigKeyword{},
		customTypeRefMatcher: nil,
		validatorOverrides:   map[string]TypeConfigValidator{},
	}
}

// ValidateConfigOption is a functional option for ValidateConfigWithOptions.
type ValidateConfigOption func(*validateConfigOptions)

// WithFieldAliases configures the raw-key-to-keyword mapping used during normalization.
// Pass V0FieldAliases for V0 input or V1FieldAliases for V1 input.
func WithFieldAliases(aliases map[string]ConfigKeyword) ValidateConfigOption {
	return func(o *validateConfigOptions) {
		o.fieldAliases = aliases
	}
}

// WithCustomTypeRefMatcher configures the function used to recognize custom type references
// in type names and array itemTypes entries.
func WithCustomTypeRefMatcher(fn func(string) bool) ValidateConfigOption {
	return func(o *validateConfigOptions) {
		o.customTypeRefMatcher = fn
	}
}

// WithValidatorOverrides injects context-specific validators for specific type names,
// replacing the default validator for those types.
func WithValidatorOverrides(overrides map[string]TypeConfigValidator) ValidateConfigOption {
	return func(o *validateConfigOptions) {
		o.validatorOverrides = overrides
	}
}

var (
	customTypeLegacyRefRegex  = regexp.MustCompile(catalogRules.CustomTypeLegacyReferenceRegex)
	customTypeCurrentRefRegex = regexp.MustCompile(catalogRules.CustomTypeReferenceRegex)
)

// legacyCustomTypeRefMatcher matches the V0 legacy custom type reference format:
// #/custom-types/<group>/<id>
func legacyCustomTypeRefMatcher(typeName string) bool {
	return customTypeLegacyRefRegex.MatchString(typeName)
}

// CurrentCustomTypeRefMatcher matches the current V1 custom type reference format:
// #custom-type:<id>
// Exposed so V1 callers can pass it via WithCustomTypeRefMatcher.
func CurrentCustomTypeRefMatcher(typeName string) bool {
	return customTypeCurrentRefRegex.MatchString(typeName)
}
