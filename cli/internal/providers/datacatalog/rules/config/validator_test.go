package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// TestStringValidator tests all aspects of StringTypeConfig.
func TestStringValidator(t *testing.T) {
	t.Parallel()

	validator := &StringTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name    string
			rawKey  string
			keyword ConfigKeyword
			val     any
		}{
			{"enum with strings", "enum", KeywordEnum, []any{"active", "inactive"}},
			{"minLength integer", "minLength", KeywordMinLength, 5},
			{"minLength float64", "minLength", KeywordMinLength, float64(10)},
			{"maxLength integer", "maxLength", KeywordMaxLength, 100},
			{"pattern string", "pattern", KeywordPattern, "^[a-z]+$"},
			{"format string", "format", KeywordFormat, "email"},
			// V1 raw keys resolve to same keywords
			{"min_length integer", "min_length", KeywordMinLength, 5},
			{"max_length integer", "max_length", KeywordMaxLength, 100},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("unsupported keyword returns ErrFieldNotSupported", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "minimum", Keyword: KeywordMinimum, Value: 10})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("invalid field values", func(t *testing.T) {
		testCases := []struct {
			name            string
			rawKey          string
			keyword         ConfigKeyword
			val             any
			expectedMessage string
		}{
			{"enum not array", "enum", KeywordEnum, "not-array", "'enum' must be an array"},
			{"minLength not integer (V0)", "minLength", KeywordMinLength, "five", "'minLength' must be an integer"},
			{"minLength not integer (V1)", "min_length", KeywordMinLength, "five", "'min_length' must be an integer"},
			{"minLength negative", "minLength", KeywordMinLength, -5, "'minLength' must be >= 0"},
			{"maxLength not integer", "maxLength", KeywordMaxLength, 3.5, "'maxLength' must be an integer"},
			{"pattern not string", "pattern", KeywordPattern, 123, "'pattern' must be a string"},
			{"format not string", "format", KeywordFormat, 123, "'format' must be a string"},
			{"format invalid value", "format", KeywordFormat, "invalid", "'format' must be one of:"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("enum with duplicates", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{"active", "inactive", "active"}})
		assert.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "enum/2", results[0].Reference)
		assert.Contains(t, results[0].Message, "'active' is a duplicate value")
	})

	t.Run("enum with multiple duplicates", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{"a", "b", "a", "c", "b"}})
		assert.NoError(t, err)
		require.Len(t, results, 2)
		assert.Contains(t, results[0].Message, "is a duplicate value")
		assert.Contains(t, results[1].Message, "is a duplicate value")
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: min_length < max_length", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinLength: {RawKey: "min_length", Keyword: KeywordMinLength, Value: 5},
				KeywordMaxLength: {RawKey: "max_length", Keyword: KeywordMaxLength, Value: 10},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("invalid: minLength > maxLength", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinLength: {RawKey: "minLength", Keyword: KeywordMinLength, Value: 10},
				KeywordMaxLength: {RawKey: "maxLength", Keyword: KeywordMaxLength, Value: 5},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "minLength cannot be greater than maxLength", results[0].Message)
		})

		t.Run("invalid V1: min_length > max_length preserves snake_case in message", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinLength: {RawKey: "min_length", Keyword: KeywordMinLength, Value: 10},
				KeywordMaxLength: {RawKey: "max_length", Keyword: KeywordMaxLength, Value: 5},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "min_length cannot be greater than max_length", results[0].Message)
		})

		t.Run("skips if values are invalid", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinLength: {RawKey: "minLength", Keyword: KeywordMinLength, Value: "invalid"},
				KeywordMaxLength: {RawKey: "maxLength", Keyword: KeywordMaxLength, Value: 5},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})
	})
}

// TestIntegerValidator tests all aspects of IntegerTypeConfig.
func TestIntegerValidator(t *testing.T) {
	t.Parallel()

	validator := &IntegerTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name    string
			rawKey  string
			keyword ConfigKeyword
			val     any
		}{
			{"enum with integers", "enum", KeywordEnum, []any{1, 2, 3}},
			{"minimum", "minimum", KeywordMinimum, 0},
			{"maximum", "maximum", KeywordMaximum, 100},
			{"exclusiveMinimum", "exclusiveMinimum", KeywordExclusiveMinimum, 0},
			{"exclusiveMaximum", "exclusiveMaximum", KeywordExclusiveMaximum, 100},
			{"multipleOf", "multipleOf", KeywordMultipleOf, 5},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("invalid field values", func(t *testing.T) {
		testCases := []struct {
			name            string
			rawKey          string
			keyword         ConfigKeyword
			val             any
			expectedMessage string
		}{
			{"minimum not integer (V0)", "minimum", KeywordMinimum, 1.5, "'minimum' must be an integer"},
			{"multipleOf not positive", "multipleOf", KeywordMultipleOf, 0, "'multipleOf' must be > 0"},
			{"multipleOf negative", "multipleOf", KeywordMultipleOf, -5, "'multipleOf' must be > 0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: minimum <= maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 0},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("invalid: minimum > maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 100},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 0},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "minimum cannot be greater than maximum", results[0].Message)
		})

		t.Run("valid: exclusive_minimum < exclusive_maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: 0},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("invalid V0: exclusiveMinimum >= exclusiveMaximum preserves camelCase in message", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: 100},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "exclusiveMinimum must be less than exclusiveMaximum", results[0].Message)
		})
	})
}

// TestNumberValidator tests all aspects of NumberTypeConfig.
func TestNumberValidator(t *testing.T) {
	t.Parallel()

	validator := &NumberTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name    string
			rawKey  string
			keyword ConfigKeyword
			val     any
		}{
			{"enum with numbers", "enum", KeywordEnum, []any{1, 2, 3}},
			{"enum with decimals", "enum", KeywordEnum, []any{1.5, 2.5, 3.5}},
			{"minimum integer", "minimum", KeywordMinimum, 0},
			{"minimum decimal", "minimum", KeywordMinimum, 0.5},
			{"maximum integer", "maximum", KeywordMaximum, 100},
			{"maximum decimal", "maximum", KeywordMaximum, 99.9},
			{"exclusiveMinimum integer", "exclusiveMinimum", KeywordExclusiveMinimum, 0},
			{"exclusiveMinimum decimal", "exclusiveMinimum", KeywordExclusiveMinimum, 0.5},
			{"exclusiveMaximum integer", "exclusiveMaximum", KeywordExclusiveMaximum, 100},
			{"exclusiveMaximum decimal", "exclusiveMaximum", KeywordExclusiveMaximum, 99.9},
			{"multipleOf integer", "multipleOf", KeywordMultipleOf, 5},
			{"multipleOf decimal", "multipleOf", KeywordMultipleOf, 0.1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("unsupported keyword returns ErrFieldNotSupported", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "minLength", Keyword: KeywordMinLength, Value: 10})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("invalid field values", func(t *testing.T) {
		testCases := []struct {
			name            string
			rawKey          string
			keyword         ConfigKeyword
			val             any
			expectedMessage string
		}{
			{"enum not array", "enum", KeywordEnum, "not-array", "'enum' must be an array"},
			{"minimum not number", "minimum", KeywordMinimum, "five", "'minimum' must be a number"},
			{"maximum not number", "maximum", KeywordMaximum, "hundred", "'maximum' must be a number"},
			{"exclusiveMinimum not number", "exclusiveMinimum", KeywordExclusiveMinimum, "zero", "'exclusiveMinimum' must be a number"},
			{"exclusiveMaximum not number", "exclusiveMaximum", KeywordExclusiveMaximum, true, "'exclusiveMaximum' must be a number"},
			{"multipleOf not number", "multipleOf", KeywordMultipleOf, "five", "'multipleOf' must be a number"},
			{"multipleOf zero", "multipleOf", KeywordMultipleOf, 0, "'multipleOf' must be > 0"},
			{"multipleOf negative", "multipleOf", KeywordMultipleOf, -5, "'multipleOf' must be > 0"},
			{"multipleOf negative decimal", "multipleOf", KeywordMultipleOf, -0.5, "'multipleOf' must be > 0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("enum with duplicates", func(t *testing.T) {
		t.Run("single duplicate", func(t *testing.T) {
			results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{1.5, 2.5, 1.5}})
			assert.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, "enum/2", results[0].Reference)
			assert.Contains(t, results[0].Message, "1.5")
		})

		t.Run("multiple duplicates", func(t *testing.T) {
			results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{1, 2, 1, 3, 2}})
			assert.NoError(t, err)
			require.Len(t, results, 2)
			assert.Contains(t, results[0].Message, "is a duplicate value")
			assert.Contains(t, results[1].Message, "is a duplicate value")
		})
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: minimum <= maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 0},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("valid: minimum == maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 50},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 50},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("valid: minimum <= maximum with decimals", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 0.5},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 99.9},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("invalid: minimum > maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 100},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 0},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "minimum cannot be greater than maximum", results[0].Message)
		})

		t.Run("invalid: minimum > maximum with decimals", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 99.9},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 0.5},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "minimum cannot be greater than maximum", results[0].Message)
		})

		t.Run("valid: exclusive_minimum < exclusive_maximum", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: 0},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("valid: exclusive_minimum < exclusive_maximum with decimals", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: 0.5},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 99.9},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("invalid V0: exclusiveMinimum == exclusiveMaximum preserves camelCase in message", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: 50},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 50},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "exclusiveMinimum must be less than exclusiveMaximum", results[0].Message)
		})

		t.Run("invalid V0: exclusiveMinimum > exclusiveMaximum preserves camelCase in message", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: 100},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 0},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "exclusiveMinimum must be less than exclusiveMaximum", results[0].Message)
		})

		t.Run("skips validation if values are invalid", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: "invalid"},
				KeywordMaximum: {RawKey: "maximum", Keyword: KeywordMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("skips validation if exclusive values are invalid", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordExclusiveMinimum: {RawKey: "exclusiveMinimum", Keyword: KeywordExclusiveMinimum, Value: "invalid"},
				KeywordExclusiveMaximum: {RawKey: "exclusiveMaximum", Keyword: KeywordExclusiveMaximum, Value: 100},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("validates only fields present", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinimum: {RawKey: "minimum", Keyword: KeywordMinimum, Value: 50},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})
	})
}

// TestArrayValidator tests all aspects of ArrayTypeConfig.
func TestArrayValidator(t *testing.T) {
	t.Parallel()

	legacyMatcher := func(s string) bool { return customTypeLegacyRefRegex.MatchString(s) }
	currentMatcher := func(s string) bool { return customTypeCurrentRefRegex.MatchString(s) }

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, (&ArrayTypeConfig{}).ConfigAllowed())
	})

	t.Run("valid fields with legacy matcher", func(t *testing.T) {
		validator := &ArrayTypeConfig{isCustomTypeRef: legacyMatcher}
		testCases := []struct {
			name    string
			rawKey  string
			keyword ConfigKeyword
			val     any
		}{
			{"itemTypes with primitives", "itemTypes", KeywordItemTypes, []any{"string", "number"}},
			{"itemTypes with single legacy custom type", "itemTypes", KeywordItemTypes, []any{"#/custom-types/foo/bar"}},
			{"minItems", "minItems", KeywordMinItems, 1},
			{"maxItems", "maxItems", KeywordMaxItems, 10},
			{"uniqueItems true", "uniqueItems", KeywordUniqueItems, true},
			{"uniqueItems false", "uniqueItems", KeywordUniqueItems, false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("valid fields with current matcher", func(t *testing.T) {
		validator := &ArrayTypeConfig{isCustomTypeRef: currentMatcher}
		testCases := []struct {
			name    string
			rawKey  string
			keyword ConfigKeyword
			val     any
		}{
			{"item_types with primitives", "item_types", KeywordItemTypes, []any{"string", "number"}},
			{"item_types with current custom type ref", "item_types", KeywordItemTypes, []any{"#custom-type:Address"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("invalid fields", func(t *testing.T) {
		validator := &ArrayTypeConfig{isCustomTypeRef: legacyMatcher}
		testCases := []struct {
			name            string
			rawKey          string
			keyword         ConfigKeyword
			val             any
			expectedMessage string
		}{
			{"itemTypes not array", "itemTypes", KeywordItemTypes, "string", "'itemTypes' must be an array"},
			{"itemTypes empty", "itemTypes", KeywordItemTypes, []any{}, "'itemTypes' must contain at least one type"},
			{"itemTypes element not string", "itemTypes", KeywordItemTypes, []any{123}, "must be a string value"},
			{"itemTypes invalid type", "itemTypes", KeywordItemTypes, []any{"invalid"}, "invalid type 'invalid' in itemTypes"},
			{"itemTypes custom type with others", "itemTypes", KeywordItemTypes, []any{"#/custom-types/foo/bar", "string"}, "custom type reference cannot be paired with other types"},
			{"minItems not integer", "minItems", KeywordMinItems, "one", "'minItems' must be an integer"},
			{"minItems negative", "minItems", KeywordMinItems, -1, "'minItems' must be >= 0"},
			{"uniqueItems not boolean", "uniqueItems", KeywordUniqueItems, "yes", "'uniqueItems' must be a boolean"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(ResolvedField{RawKey: tc.rawKey, Keyword: tc.keyword, Value: tc.val})
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("invalid: item_types with current custom type ref paired with others is rejected", func(t *testing.T) {
		validator := &ArrayTypeConfig{isCustomTypeRef: currentMatcher}
		results, err := validator.ValidateField(ResolvedField{RawKey: "item_types", Keyword: KeywordItemTypes, Value: []any{"#custom-type:Address", "string"}})
		assert.NoError(t, err)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "custom type reference cannot be paired with other types")
	})

	t.Run("cross-field validation", func(t *testing.T) {
		validator := &ArrayTypeConfig{}

		t.Run("valid: min_items <= max_items", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinItems: {RawKey: "minItems", Keyword: KeywordMinItems, Value: 1},
				KeywordMaxItems: {RawKey: "maxItems", Keyword: KeywordMaxItems, Value: 10},
			}
			results := validator.ValidateCrossFields(cfg)
			assert.Empty(t, results)
		})

		t.Run("invalid: minItems > maxItems preserves raw keys in message", func(t *testing.T) {
			cfg := map[ConfigKeyword]ResolvedField{
				KeywordMinItems: {RawKey: "minItems", Keyword: KeywordMinItems, Value: 10},
				KeywordMaxItems: {RawKey: "maxItems", Keyword: KeywordMaxItems, Value: 1},
			}
			results := validator.ValidateCrossFields(cfg)
			require.Len(t, results, 1)
			assert.Equal(t, "minItems cannot be greater than maxItems", results[0].Message)
		})
	})
}

// TestBooleanValidator tests all aspects of BooleanTypeConfig.
func TestBooleanValidator(t *testing.T) {
	t.Parallel()

	validator := &BooleanTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid enum", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{true, false}})
		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("enum with duplicates", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{true, false, true}})
		assert.NoError(t, err)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "true")
	})

	t.Run("non-enum keyword not supported", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "minLength", Keyword: KeywordMinLength, Value: 5})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[ConfigKeyword]ResolvedField{})
		assert.Empty(t, results)
	})
}

// TestObjectValidator tests all aspects of ObjectTypeConfig.
func TestObjectValidator(t *testing.T) {
	t.Parallel()

	validator := &ObjectTypeConfig{}

	t.Run("ConfigAllowed returns false", func(t *testing.T) {
		assert.False(t, validator.ConfigAllowed())
	})

	t.Run("all fields not supported", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "additionalProperties", Keyword: KeywordAdditionalProperties, Value: true})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[ConfigKeyword]ResolvedField{})
		assert.Empty(t, results)
	})
}

// TestNullValidator tests all aspects of NullTypeConfig.
func TestNullValidator(t *testing.T) {
	t.Parallel()

	validator := &NullTypeConfig{}

	t.Run("ConfigAllowed returns false", func(t *testing.T) {
		assert.False(t, validator.ConfigAllowed())
	})

	t.Run("all fields not supported", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "enum", Keyword: KeywordEnum, Value: []any{}})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[ConfigKeyword]ResolvedField{})
		assert.Empty(t, results)
	})
}

// TestCustomTypeValidator tests all aspects of CustomTypeConfig.
func TestCustomTypeValidator(t *testing.T) {
	t.Parallel()

	validator := &CustomTypeConfig{}

	t.Run("ConfigAllowed returns false", func(t *testing.T) {
		assert.False(t, validator.ConfigAllowed())
	})

	t.Run("all fields not supported", func(t *testing.T) {
		results, err := validator.ValidateField(ResolvedField{RawKey: "$ref", Keyword: ConfigKeyword(""), Value: "#/custom-types/foo"})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[ConfigKeyword]ResolvedField{})
		assert.Empty(t, results)
	})
}

// TestValidateConfig tests the legacy V0 entrypoint.
func TestValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("single type validation", func(t *testing.T) {
		cfg := map[string]any{
			"minLength": 5,
			"maxLength": 10,
		}
		results := ValidateConfig([]string{"string"}, cfg, "/test", nil)
		assert.Empty(t, results)
	})

	t.Run("multi-type union semantics", func(t *testing.T) {
		t.Run("field valid for any type is accepted", func(t *testing.T) {
			cfg := map[string]any{
				"minimum":   10,
				"minLength": 5,
			}
			results := ValidateConfig([]string{"string", "integer"}, cfg, "/test", nil)
			assert.Empty(t, results)
		})

		t.Run("field invalid for all types is rejected", func(t *testing.T) {
			cfg := map[string]any{
				"invalidKey": "value",
			}
			results := ValidateConfig([]string{"string", "integer"}, cfg, "/test", nil)
			require.Len(t, results, 1)
			assert.Contains(t, results[0].Message, "'invalidKey' is not applicable")
		})
	})

	t.Run("config not allowed for type", func(t *testing.T) {
		cfg := map[string]any{
			"someKey": "value",
		}
		results := ValidateConfig([]string{"object"}, cfg, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed")
	})

	t.Run("config not allowed for multi-type where all disallow", func(t *testing.T) {
		cfg := map[string]any{
			"someKey": "value",
		}
		results := ValidateConfig([]string{"object", "null"}, cfg, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed")
	})

	t.Run("empty config is valid", func(t *testing.T) {
		results := ValidateConfig([]string{"object"}, map[string]any{}, "/test", nil)
		assert.Empty(t, results)
	})

	t.Run("reference prefixing", func(t *testing.T) {
		cfg := map[string]any{
			"minLength": "invalid",
		}
		results := ValidateConfig([]string{"string"}, cfg, "/types/0/config", nil)
		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/config/minLength", results[0].Reference)
	})

	t.Run("multi-type union deduplicates enum errors", func(t *testing.T) {
		cfg := map[string]any{
			"enum": []any{1, 2, 1},
		}
		results := ValidateConfig([]string{"string", "integer"}, cfg, "/test", nil)
		require.Len(t, results, 1)
		assert.Equal(t, "/test/enum/2", results[0].Reference)
		assert.Contains(t, results[0].Message, "'1' is a duplicate value")
	})

	t.Run("multi-type union deduplicates enum not-array error", func(t *testing.T) {
		cfg := map[string]any{
			"enum": "not-array",
		}
		results := ValidateConfig([]string{"string", "integer", "boolean"}, cfg, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'enum' must be an array")
	})

	t.Run("cross-field validation with deduplication", func(t *testing.T) {
		cfg := map[string]any{
			"minLength": 10,
			"maxLength": 5,
		}
		results := ValidateConfig([]string{"string"}, cfg, "/test", nil)
		require.Len(t, results, 1)
		assert.Equal(t, "/test", results[0].Reference)
		// V0 cross-field messages use the original camelCase raw keys for backward compatibility.
		assert.Contains(t, results[0].Message, "minLength cannot be greater than maxLength")
	})

	t.Run("validator override replaces default for specified type", func(t *testing.T) {
		overrides := map[string]TypeConfigValidator{
			"object": &StringTypeConfig{},
		}
		cfg := map[string]any{
			"format": "date",
		}
		results := ValidateConfig([]string{"object"}, cfg, "/test", overrides)
		assert.Empty(t, results)
	})

	t.Run("validator override does not affect other types", func(t *testing.T) {
		overrides := map[string]TypeConfigValidator{
			"object": &StringTypeConfig{},
		}
		cfg := map[string]any{
			"multipleOf": 5,
		}
		results := ValidateConfig([]string{"number"}, cfg, "/test", overrides)
		assert.Empty(t, results)
	})

	t.Run("nil override falls back to default validators", func(t *testing.T) {
		cfg := map[string]any{
			"someKey": "value",
		}
		results := ValidateConfig([]string{"object"}, cfg, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed")
	})

	t.Run("V0 legacy custom type ref in itemTypes is accepted", func(t *testing.T) {
		cfg := map[string]any{
			"itemTypes": []any{"#/custom-types/foo/bar"},
		}
		results := ValidateConfig([]string{"array"}, cfg, "/test", nil)
		assert.Empty(t, results)
	})
}

// TestValidateConfigWithOptions covers all V1 and options-aware behavior.
func TestValidateConfigWithOptions(t *testing.T) {
	t.Parallel()

	v1Matcher := CurrentCustomTypeRefMatcher

	t.Run("zero options does not panic", func(t *testing.T) {
		cfg := map[string]any{"min_length": 5}
		// Must not panic; all keys are unresolved → "not applicable" behavior
		assert.NotPanics(t, func() {
			ValidateConfigWithOptions([]string{"string"}, cfg, "/test")
		})
	})

	t.Run("empty alias map treats all fields as unresolved", func(t *testing.T) {
		cfg := map[string]any{"min_length": 5}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(map[string]ConfigKeyword{}),
		)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'min_length' is not applicable for type(s)")
	})

	t.Run("empty config is valid", func(t *testing.T) {
		results := ValidateConfigWithOptions(
			[]string{"string"}, map[string]any{}, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		assert.Empty(t, results)
	})

	t.Run("V0 wrapper behavior unchanged", func(t *testing.T) {
		testCases := []struct {
			name      string
			types     []string
			cfg       map[string]any
			wantEmpty bool
			wantMsg   string
		}{
			{
				name:      "V0 minLength valid",
				types:     []string{"string"},
				cfg:       map[string]any{"minLength": 5},
				wantEmpty: true,
			},
			{
				name:      "V0 itemTypes valid",
				types:     []string{"array"},
				cfg:       map[string]any{"itemTypes": []any{"string"}},
				wantEmpty: true,
			},
			{
				name:      "V0 exclusiveMinimum valid",
				types:     []string{"integer"},
				cfg:       map[string]any{"exclusiveMinimum": 0},
				wantEmpty: true,
			},
			{
				name:    "V0 invalid minLength type",
				types:   []string{"string"},
				cfg:     map[string]any{"minLength": "bad"},
				wantMsg: "'minLength' must be an integer",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results := ValidateConfig(tc.types, tc.cfg, "/test", nil)
				if tc.wantEmpty {
					assert.Empty(t, results)
				} else {
					require.NotEmpty(t, results)
					assert.Contains(t, results[0].Message, tc.wantMsg)
				}
			})
		}
	})

	t.Run("V1 snake_case keys are accepted", func(t *testing.T) {
		testCases := []struct {
			name  string
			types []string
			cfg   map[string]any
		}{
			{"min_length", []string{"string"}, map[string]any{"min_length": 5}},
			{"max_length", []string{"string"}, map[string]any{"max_length": 100}},
			{"exclusive_minimum", []string{"integer"}, map[string]any{"exclusive_minimum": 0}},
			{"exclusive_maximum", []string{"integer"}, map[string]any{"exclusive_maximum": 100}},
			{"multiple_of", []string{"integer"}, map[string]any{"multiple_of": 5}},
			{"min_items", []string{"array"}, map[string]any{"min_items": 1}},
			{"max_items", []string{"array"}, map[string]any{"max_items": 10}},
			{"unique_items", []string{"array"}, map[string]any{"unique_items": true}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results := ValidateConfigWithOptions(
					tc.types, tc.cfg, "/test",
					WithFieldAliases(V1FieldAliases),
					WithCustomTypeRefMatcher(v1Matcher),
				)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("V1 unchanged keys accepted: enum minimum maximum pattern format", func(t *testing.T) {
		testCases := []struct {
			name  string
			types []string
			cfg   map[string]any
		}{
			{"enum", []string{"string"}, map[string]any{"enum": []any{"a", "b"}}},
			{"minimum", []string{"integer"}, map[string]any{"minimum": 0}},
			{"maximum", []string{"integer"}, map[string]any{"maximum": 100}},
			{"pattern", []string{"string"}, map[string]any{"pattern": "^[a-z]+$"}},
			{"format", []string{"string"}, map[string]any{"format": "email"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results := ValidateConfigWithOptions(
					tc.types, tc.cfg, "/test",
					WithFieldAliases(V1FieldAliases),
				)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("V1 unaliased camelCase keys fall through to not-applicable", func(t *testing.T) {
		testCases := []struct {
			name   string
			types  []string
			cfg    map[string]any
			rawKey string
		}{
			{"minLength not in V1 aliases", []string{"string"}, map[string]any{"minLength": 5}, "minLength"},
			{"itemTypes not in V1 aliases", []string{"array"}, map[string]any{"itemTypes": []any{"string"}}, "itemTypes"},
			{"exclusiveMinimum not in V1 aliases", []string{"integer"}, map[string]any{"exclusiveMinimum": 0}, "exclusiveMinimum"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results := ValidateConfigWithOptions(
					tc.types, tc.cfg, "/test",
					WithFieldAliases(V1FieldAliases),
				)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, fmt.Sprintf("'%s' is not applicable for type(s)", tc.rawKey))
			})
		}
	})

	t.Run("V1 item_types in config is not applicable (spec-level field)", func(t *testing.T) {
		cfg := map[string]any{
			"item_types": []any{"string"},
		}
		results := ValidateConfigWithOptions(
			[]string{"array"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
			WithCustomTypeRefMatcher(v1Matcher),
		)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'item_types' is not applicable for type(s)")
	})

	t.Run("V1 custom type object config accepts additional_properties", func(t *testing.T) {
		// Simulate the customTypeObjectConfig override using keyword-aware interface
		type kwValidator struct {
			TypeConfigValidator
		}
		override := &customTypeObjectConfigForTest{}
		cfg := map[string]any{
			"additional_properties": true,
		}
		results := ValidateConfigWithOptions(
			[]string{"object"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
			WithValidatorOverrides(map[string]TypeConfigValidator{"object": override}),
		)
		assert.Empty(t, results)
	})

	t.Run("V1 additionalProperties (unaliased) falls through to not-applicable", func(t *testing.T) {
		override := &customTypeObjectConfigForTest{}
		cfg := map[string]any{
			"additionalProperties": true,
		}
		results := ValidateConfigWithOptions(
			[]string{"object"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
			WithValidatorOverrides(map[string]TypeConfigValidator{"object": override}),
		)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'additionalProperties' is not applicable for type(s)")
	})

	t.Run("mixed valid snake_case and unaliased camelCase in same config", func(t *testing.T) {
		cfg := map[string]any{
			"min_length": 5,
			"minLength":  10,
		}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		// min_length validates normally; minLength is not applicable
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'minLength' is not applicable for type(s)")
	})

	t.Run("cross-field validation works after V1 normalization", func(t *testing.T) {
		t.Run("min_length > max_length", func(t *testing.T) {
			cfg := map[string]any{"min_length": 10, "max_length": 5}
			results := ValidateConfigWithOptions(
				[]string{"string"}, cfg, "/test",
				WithFieldAliases(V1FieldAliases),
			)
			require.Len(t, results, 1)
			assert.Contains(t, results[0].Message, "min_length cannot be greater than max_length")
		})

		t.Run("min_items > max_items", func(t *testing.T) {
			cfg := map[string]any{"min_items": 10, "max_items": 5}
			results := ValidateConfigWithOptions(
				[]string{"array"}, cfg, "/test",
				WithFieldAliases(V1FieldAliases),
			)
			require.Len(t, results, 1)
			assert.Contains(t, results[0].Message, "min_items cannot be greater than max_items")
		})

		t.Run("exclusive_minimum >= exclusive_maximum", func(t *testing.T) {
			cfg := map[string]any{"exclusive_minimum": 10, "exclusive_maximum": 5}
			results := ValidateConfigWithOptions(
				[]string{"integer"}, cfg, "/test",
				WithFieldAliases(V1FieldAliases),
			)
			require.Len(t, results, 1)
			assert.Contains(t, results[0].Message, "exclusive_minimum must be less than exclusive_maximum")
		})
	})

	t.Run("validator override behavior works with options-aware entrypoint", func(t *testing.T) {
		overrides := map[string]TypeConfigValidator{
			"object": &StringTypeConfig{},
		}
		cfg := map[string]any{"format": "date"}
		results := ValidateConfigWithOptions(
			[]string{"object"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
			WithValidatorOverrides(overrides),
		)
		assert.Empty(t, results)
	})

	t.Run("V1 custom type top-level type detection rejects config", func(t *testing.T) {
		cfg := map[string]any{"min_length": 5}
		results := ValidateConfigWithOptions(
			[]string{"#custom-type:Address"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
			WithCustomTypeRefMatcher(v1Matcher),
		)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed for the specified type(s)")
	})

	t.Run("enum mixed element types remain valid", func(t *testing.T) {
		cfg := map[string]any{"enum": []any{1, "two", true}}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		assert.Empty(t, results)
	})

	t.Run("pattern invalid regex syntax is not rejected", func(t *testing.T) {
		cfg := map[string]any{"pattern": "[invalid regex"}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		assert.Empty(t, results)
	})

	t.Run("unknown type names cause validator to defer", func(t *testing.T) {
		cfg := map[string]any{"min_length": 5}
		results := ValidateConfigWithOptions(
			[]string{"unknownType"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		// No validator found → defers, returns nil
		assert.Empty(t, results)
	})

	t.Run("custom type ref matcher returning true for string does not override built-in", func(t *testing.T) {
		alwaysTrue := func(string) bool { return true }
		cfg := map[string]any{"min_length": 5}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
			WithCustomTypeRefMatcher(alwaysTrue),
		)
		// Built-in string validator takes precedence; min_length is valid
		assert.Empty(t, results)
	})

	t.Run("error Reference uses raw key", func(t *testing.T) {
		cfg := map[string]any{"min_length": "bad"}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		require.Len(t, results, 1)
		assert.Equal(t, "/test/min_length", results[0].Reference)
	})

	t.Run("error Message uses raw key", func(t *testing.T) {
		cfg := map[string]any{"min_length": "bad"}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'min_length' must be an integer")
	})

	t.Run("cross-field with V1 input uses raw snake_case keys in message", func(t *testing.T) {
		cfg := map[string]any{"min_length": 10, "max_length": 5}
		results := ValidateConfigWithOptions(
			[]string{"string"}, cfg, "/test",
			WithFieldAliases(V1FieldAliases),
		)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "min_length cannot be greater than max_length")
	})
}

// customTypeObjectConfigForTest is a test-local replica of the customtype package's
// customTypeObjectConfig, used to test the override path without importing that package.
type customTypeObjectConfigForTest struct{}

var allowedCustomTypeObjectKeysForTest = map[ConfigKeyword]bool{
	KeywordAdditionalProperties: true,
}

func (c *customTypeObjectConfigForTest) ConfigAllowed() bool { return true }

func (c *customTypeObjectConfigForTest) ValidateField(field ResolvedField) ([]rules.ValidationResult, error) {
	if !allowedCustomTypeObjectKeysForTest[field.Keyword] {
		return nil, ErrFieldNotSupported
	}
	if field.Keyword == KeywordAdditionalProperties {
		if _, ok := field.Value.(bool); !ok {
			return []rules.ValidationResult{{
				Reference: field.RawKey,
				Message:   fmt.Sprintf("'%s' must be a boolean", field.RawKey),
			}}, nil
		}
	}
	return nil, nil
}

func (c *customTypeObjectConfigForTest) ValidateCrossFields(_ map[ConfigKeyword]ResolvedField) []rules.ValidationResult {
	return nil
}
