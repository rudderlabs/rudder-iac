package config

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
)

type ValidateConfigOption func(*validateConfigOptions)

type validateConfigOptions struct {
	fieldAliases         map[string]ConfigKeyword
	customTypeRefMatcher func(string) bool
	validatorOverrides   map[string]TypeConfigValidator
}

var (
	v0FieldAliases = map[string]ConfigKeyword{
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
	v1FieldAliases = map[string]ConfigKeyword{
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
		"item_types":            KeywordItemTypes,
		"min_items":             KeywordMinItems,
		"max_items":             KeywordMaxItems,
		"unique_items":          KeywordUniqueItems,
		"additional_properties": KeywordAdditionalProperties,
	}
	configKeywordToValidatorField = map[ConfigKeyword]string{
		KeywordEnum:                 "enum",
		KeywordMinimum:              "minimum",
		KeywordMaximum:              "maximum",
		KeywordPattern:              "pattern",
		KeywordFormat:               "format",
		KeywordMinLength:            "minLength",
		KeywordMaxLength:            "maxLength",
		KeywordExclusiveMinimum:     "exclusiveMinimum",
		KeywordExclusiveMaximum:     "exclusiveMaximum",
		KeywordMultipleOf:           "multipleOf",
		KeywordItemTypes:            "itemTypes",
		KeywordMinItems:             "minItems",
		KeywordMaxItems:             "maxItems",
		KeywordUniqueItems:          "uniqueItems",
		KeywordAdditionalProperties: "additionalProperties",
	}
)

func WithFieldAliases(aliases map[string]ConfigKeyword) ValidateConfigOption {
	return func(opts *validateConfigOptions) {
		opts.fieldAliases = cloneFieldAliases(aliases)
	}
}

func WithCustomTypeRefMatcher(fn func(string) bool) ValidateConfigOption {
	return func(opts *validateConfigOptions) {
		opts.customTypeRefMatcher = fn
	}
}

func WithValidatorOverrides(overrides map[string]TypeConfigValidator) ValidateConfigOption {
	return func(opts *validateConfigOptions) {
		opts.validatorOverrides = cloneValidatorOverrides(overrides)
	}
}

func V0FieldAliases() map[string]ConfigKeyword {
	return cloneFieldAliases(v0FieldAliases)
}

func V1FieldAliases() map[string]ConfigKeyword {
	return cloneFieldAliases(v1FieldAliases)
}

func newValidateConfigOptions(opts ...ValidateConfigOption) validateConfigOptions {
	options := validateConfigOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&options)
	}
	return options
}

func cloneFieldAliases(aliases map[string]ConfigKeyword) map[string]ConfigKeyword {
	if aliases == nil {
		return nil
	}

	cloned := make(map[string]ConfigKeyword, len(aliases))
	for raw, keyword := range aliases {
		cloned[raw] = keyword
	}
	return cloned
}

func cloneValidatorOverrides(overrides map[string]TypeConfigValidator) map[string]TypeConfigValidator {
	if overrides == nil {
		return nil
	}

	cloned := make(map[string]TypeConfigValidator, len(overrides))
	for typeName, validator := range overrides {
		cloned[typeName] = validator
	}
	return cloned
}

func validatorFieldName(keyword ConfigKeyword) string {
	return configKeywordToValidatorField[keyword]
}
