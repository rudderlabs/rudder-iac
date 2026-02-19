package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStringValidator tests all aspects of StringTypeConfig
func TestStringValidator(t *testing.T) {
	t.Parallel()

	validator := &StringTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name      string
			fieldname string
			fieldval  any
		}{
			{"enum with strings", "enum", []any{"active", "inactive"}},
			{"minLength integer", "minLength", 5},
			{"minLength float64", "minLength", float64(10)},
			{"maxLength integer", "maxLength", 100},
			{"pattern string", "pattern", "^[a-z]+$"},
			{"format string", "format", "email"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("invalid field names", func(t *testing.T) {
		results, err := validator.ValidateField("minimum", 10)
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("invalid field values", func(t *testing.T) {
		testCases := []struct {
			name            string
			fieldname       string
			fieldval        any
			expectedMessage string
		}{
			{"enum not array", "enum", "not-array", "'enum' must be an array"},
			{"minLength not integer", "minLength", "five", "'minLength' must be an integer"},
			{"minLength negative", "minLength", -5, "'minLength' must be >= 0"},
			{"maxLength not integer", "maxLength", 3.5, "'maxLength' must be an integer"},
			{"pattern not string", "pattern", 123, "'pattern' must be a string"},
			{"format not string", "format", 123, "'format' must be a string"},
			{"format invalid value", "format", "invalid", "'format' must be one of:"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("enum with duplicates", func(t *testing.T) {
		results, err := validator.ValidateField("enum", []any{"active", "inactive", "active"})
		assert.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "enum/2", results[0].Reference)
		assert.Contains(t, results[0].Message, "'active' is a duplicate value")
	})

	t.Run("enum with multiple duplicates", func(t *testing.T) {
		results, err := validator.ValidateField("enum", []any{"a", "b", "a", "c", "b"})
		assert.NoError(t, err)
		require.Len(t, results, 2)
		// Should report both duplicate occurrences
		assert.Contains(t, results[0].Message, "is a duplicate value")
		assert.Contains(t, results[1].Message, "is a duplicate value")
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: minLength < maxLength", func(t *testing.T) {
			config := map[string]any{"minLength": 5, "maxLength": 10}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("invalid: minLength > maxLength", func(t *testing.T) {
			config := map[string]any{"minLength": 10, "maxLength": 5}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "minLength cannot be greater than maxLength", results[0].Message)
		})

		t.Run("skips if values are invalid", func(t *testing.T) {
			config := map[string]any{"minLength": "invalid", "maxLength": 5}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})
	})
}

// TestIntegerValidator tests all aspects of IntegerTypeConfig
func TestIntegerValidator(t *testing.T) {
	t.Parallel()

	validator := &IntegerTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name      string
			fieldname string
			fieldval  any
		}{
			{"enum with integers", "enum", []any{1, 2, 3}},
			{"minimum", "minimum", 0},
			{"maximum", "maximum", 100},
			{"exclusiveMinimum", "exclusiveMinimum", 0},
			{"exclusiveMaximum", "exclusiveMaximum", 100},
			{"multipleOf", "multipleOf", 5},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("invalid field values", func(t *testing.T) {
		testCases := []struct {
			name            string
			fieldname       string
			fieldval        any
			expectedMessage string
		}{
			{"minimum not integer", "minimum", 1.5, "'minimum' must be an integer"},
			{"multipleOf not positive", "multipleOf", 0, "'multipleOf' must be > 0"},
			{"multipleOf negative", "multipleOf", -5, "'multipleOf' must be > 0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: minimum <= maximum", func(t *testing.T) {
			config := map[string]any{"minimum": 0, "maximum": 100}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("invalid: minimum > maximum", func(t *testing.T) {
			config := map[string]any{"minimum": 100, "maximum": 0}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "minimum cannot be greater than maximum", results[0].Message)
		})

		t.Run("valid: exclusiveMinimum < exclusiveMaximum", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": 0, "exclusiveMaximum": 100}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("invalid: exclusiveMinimum >= exclusiveMaximum", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": 100, "exclusiveMaximum": 100}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "exclusiveMinimum must be less than exclusiveMaximum", results[0].Message)
		})
	})
}

// TestNumberValidator tests all aspects of NumberTypeConfig
func TestNumberValidator(t *testing.T) {
	t.Parallel()

	validator := &NumberTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name      string
			fieldname string
			fieldval  any
		}{
			{"enum with numbers", "enum", []any{1, 2, 3}},
			{"enum with decimals", "enum", []any{1.5, 2.5, 3.5}},
			{"minimum integer", "minimum", 0},
			{"minimum decimal", "minimum", 0.5},
			{"maximum integer", "maximum", 100},
			{"maximum decimal", "maximum", 99.9},
			{"exclusiveMinimum integer", "exclusiveMinimum", 0},
			{"exclusiveMinimum decimal", "exclusiveMinimum", 0.5},
			{"exclusiveMaximum integer", "exclusiveMaximum", 100},
			{"exclusiveMaximum decimal", "exclusiveMaximum", 99.9},
			{"multipleOf integer", "multipleOf", 5},
			{"multipleOf decimal", "multipleOf", 0.1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("invalid field names", func(t *testing.T) {
		results, err := validator.ValidateField("minLength", 10)
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("invalid field values", func(t *testing.T) {
		testCases := []struct {
			name            string
			fieldname       string
			fieldval        any
			expectedMessage string
		}{
			{"enum not array", "enum", "not-array", "'enum' must be an array"},
			{"minimum not number", "minimum", "five", "'minimum' must be a number"},
			{"maximum not number", "maximum", "hundred", "'maximum' must be a number"},
			{"exclusiveMinimum not number", "exclusiveMinimum", "zero", "'exclusiveMinimum' must be a number"},
			{"exclusiveMaximum not number", "exclusiveMaximum", true, "'exclusiveMaximum' must be a number"},
			{"multipleOf not number", "multipleOf", "five", "'multipleOf' must be a number"},
			{"multipleOf zero", "multipleOf", 0, "'multipleOf' must be > 0"},
			{"multipleOf negative", "multipleOf", -5, "'multipleOf' must be > 0"},
			{"multipleOf negative decimal", "multipleOf", -0.5, "'multipleOf' must be > 0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("enum with duplicates", func(t *testing.T) {
		t.Run("single duplicate", func(t *testing.T) {
			results, err := validator.ValidateField("enum", []any{1.5, 2.5, 1.5})
			assert.NoError(t, err)
			require.Len(t, results, 1)
			assert.Equal(t, "enum/2", results[0].Reference)
			assert.Contains(t, results[0].Message, "1.5")
		})

		t.Run("multiple duplicates", func(t *testing.T) {
			results, err := validator.ValidateField("enum", []any{1, 2, 1, 3, 2})
			assert.NoError(t, err)
			require.Len(t, results, 2)
			assert.Contains(t, results[0].Message, "is a duplicate value")
			assert.Contains(t, results[1].Message, "is a duplicate value")
		})
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: minimum <= maximum", func(t *testing.T) {
			config := map[string]any{"minimum": 0, "maximum": 100}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("valid: minimum == maximum", func(t *testing.T) {
			config := map[string]any{"minimum": 50, "maximum": 50}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("valid: minimum <= maximum with decimals", func(t *testing.T) {
			config := map[string]any{"minimum": 0.5, "maximum": 99.9}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("invalid: minimum > maximum", func(t *testing.T) {
			config := map[string]any{"minimum": 100, "maximum": 0}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "minimum cannot be greater than maximum", results[0].Message)
		})

		t.Run("invalid: minimum > maximum with decimals", func(t *testing.T) {
			config := map[string]any{"minimum": 99.9, "maximum": 0.5}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "minimum cannot be greater than maximum", results[0].Message)
		})

		t.Run("valid: exclusiveMinimum < exclusiveMaximum", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": 0, "exclusiveMaximum": 100}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("valid: exclusiveMinimum < exclusiveMaximum with decimals", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": 0.5, "exclusiveMaximum": 99.9}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("invalid: exclusiveMinimum == exclusiveMaximum", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": 50, "exclusiveMaximum": 50}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "exclusiveMinimum must be less than exclusiveMaximum", results[0].Message)
		})

		t.Run("invalid: exclusiveMinimum > exclusiveMaximum", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": 100, "exclusiveMaximum": 0}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "exclusiveMinimum must be less than exclusiveMaximum", results[0].Message)
		})

		t.Run("skips validation if values are invalid", func(t *testing.T) {
			config := map[string]any{"minimum": "invalid", "maximum": 100}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("skips validation if exclusive values are invalid", func(t *testing.T) {
			config := map[string]any{"exclusiveMinimum": "invalid", "exclusiveMaximum": 100}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("validates only fields present", func(t *testing.T) {
			config := map[string]any{"minimum": 50}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})
	})
}

// TestArrayValidator tests all aspects of ArrayTypeConfig
func TestArrayValidator(t *testing.T) {
	t.Parallel()

	validator := &ArrayTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid fields", func(t *testing.T) {
		testCases := []struct {
			name      string
			fieldname string
			fieldval  any
		}{
			{"itemTypes with primitives", "itemTypes", []any{"string", "number"}},
			{"itemTypes with single custom type", "itemTypes", []any{"#/custom-types/foo/bar"}},
			{"minItems", "minItems", 1},
			{"maxItems", "maxItems", 10},
			{"uniqueItems true", "uniqueItems", true},
			{"uniqueItems false", "uniqueItems", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				assert.Empty(t, results)
			})
		}
	})

	t.Run("invalid fields", func(t *testing.T) {
		testCases := []struct {
			name            string
			fieldname       string
			fieldval        any
			expectedMessage string
		}{
			{"itemTypes not array", "itemTypes", "string", "'itemTypes' must be an array"},
			{"itemTypes empty", "itemTypes", []any{}, "'itemTypes' must contain at least one type"},
			{"itemTypes element not string", "itemTypes", []any{123}, "must be a string value"},
			{"itemTypes invalid type", "itemTypes", []any{"invalid"}, "invalid type 'invalid' in itemTypes"},
			{"itemTypes custom type with others", "itemTypes", []any{"#/custom-types/foo/bar", "string"}, "custom type reference cannot be paired with other types"},
			{"minItems not integer", "minItems", "one", "'minItems' must be an integer"},
			{"minItems negative", "minItems", -1, "'minItems' must be >= 0"},
			{"uniqueItems not boolean", "uniqueItems", "yes", "'uniqueItems' must be a boolean"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				results, err := validator.ValidateField(tc.fieldname, tc.fieldval)
				assert.NoError(t, err)
				require.Len(t, results, 1)
				assert.Contains(t, results[0].Message, tc.expectedMessage)
			})
		}
	})

	t.Run("cross-field validation", func(t *testing.T) {
		t.Run("valid: minItems <= maxItems", func(t *testing.T) {
			config := map[string]any{"minItems": 1, "maxItems": 10}
			results := validator.ValidateCrossFields(config)
			assert.Empty(t, results)
		})

		t.Run("invalid: minItems > maxItems", func(t *testing.T) {
			config := map[string]any{"minItems": 10, "maxItems": 1}
			results := validator.ValidateCrossFields(config)
			require.Len(t, results, 1)
			assert.Equal(t, "minItems cannot be greater than maxItems", results[0].Message)
		})
	})
}

// TestBooleanValidator tests all aspects of BooleanTypeConfig
func TestBooleanValidator(t *testing.T) {
	t.Parallel()

	validator := &BooleanTypeConfig{}

	t.Run("ConfigAllowed returns true", func(t *testing.T) {
		assert.True(t, validator.ConfigAllowed())
	})

	t.Run("valid enum", func(t *testing.T) {
		results, err := validator.ValidateField("enum", []any{true, false})
		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("enum with duplicates", func(t *testing.T) {
		results, err := validator.ValidateField("enum", []any{true, false, true})
		assert.NoError(t, err)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "true")
	})

	t.Run("non-enum field not supported", func(t *testing.T) {
		results, err := validator.ValidateField("minLength", 5)
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[string]any{})
		assert.Empty(t, results)
	})
}

// TestObjectValidator tests all aspects of ObjectTypeConfig
func TestObjectValidator(t *testing.T) {
	t.Parallel()

	validator := &ObjectTypeConfig{}

	t.Run("ConfigAllowed returns false", func(t *testing.T) {
		assert.False(t, validator.ConfigAllowed())
	})

	t.Run("all fields not supported", func(t *testing.T) {
		results, err := validator.ValidateField("additionalProperties", true)
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[string]any{})
		assert.Empty(t, results)
	})
}

// TestNullValidator tests all aspects of NullTypeConfig
func TestNullValidator(t *testing.T) {
	t.Parallel()

	validator := &NullTypeConfig{}

	t.Run("ConfigAllowed returns false", func(t *testing.T) {
		assert.False(t, validator.ConfigAllowed())
	})

	t.Run("all fields not supported", func(t *testing.T) {
		results, err := validator.ValidateField("enum", []any{})
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[string]any{})
		assert.Empty(t, results)
	})
}

// TestCustomTypeValidator tests all aspects of CustomTypeConfig
func TestCustomTypeValidator(t *testing.T) {
	t.Parallel()

	validator := &CustomTypeConfig{}

	t.Run("ConfigAllowed returns false", func(t *testing.T) {
		assert.False(t, validator.ConfigAllowed())
	})

	t.Run("all fields not supported", func(t *testing.T) {
		results, err := validator.ValidateField("$ref", "#/custom-types/foo")
		assert.ErrorIs(t, err, ErrFieldNotSupported)
		assert.Nil(t, results)
	})

	t.Run("no cross-field validation", func(t *testing.T) {
		results := validator.ValidateCrossFields(map[string]any{})
		assert.Empty(t, results)
	})
}

// TestValidateConfig tests the main ValidateConfig function
func TestValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("single type validation", func(t *testing.T) {
		config := map[string]any{
			"minLength": 5,
			"maxLength": 10,
		}
		results := ValidateConfig([]string{"string"}, config, "/test", nil)
		assert.Empty(t, results)
	})

	t.Run("multi-type union semantics", func(t *testing.T) {
		t.Run("field valid for any type is accepted", func(t *testing.T) {
			config := map[string]any{
				"minimum":   10, // valid for integer
				"minLength": 5,  // valid for string
			}
			// Both fields should be valid because union semantics
			results := ValidateConfig([]string{"string", "integer"}, config, "/test", nil)
			assert.Empty(t, results)
		})

		t.Run("field invalid for all types is rejected", func(t *testing.T) {
			config := map[string]any{
				"invalidKey": "value",
			}
			results := ValidateConfig([]string{"string", "integer"}, config, "/test", nil)
			require.Len(t, results, 1)
			assert.Contains(t, results[0].Message, "'invalidKey' is not applicable")
		})
	})

	t.Run("config not allowed for type", func(t *testing.T) {
		config := map[string]any{
			"someKey": "value",
		}
		results := ValidateConfig([]string{"object"}, config, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed")
	})

	t.Run("config not allowed for multi-type where all disallow", func(t *testing.T) {
		config := map[string]any{
			"someKey": "value",
		}
		results := ValidateConfig([]string{"object", "null"}, config, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed")
	})

	t.Run("empty config is valid", func(t *testing.T) {
		results := ValidateConfig([]string{"object"}, map[string]any{}, "/test", nil)
		assert.Empty(t, results)
	})

	t.Run("reference prefixing", func(t *testing.T) {
		config := map[string]any{
			"minLength": "invalid",
		}
		results := ValidateConfig([]string{"string"}, config, "/types/0/config", nil)
		require.Len(t, results, 1)
		assert.Equal(t, "/types/0/config/minLength", results[0].Reference)
	})

	t.Run("multi-type union deduplicates enum errors", func(t *testing.T) {
		config := map[string]any{
			"enum": []any{1, 2, 1},
		}
		// Both string and integer validators recognize "enum" and report the same duplicate error.
		// Without dedup, we'd get duplicate results.
		results := ValidateConfig([]string{"string", "integer"}, config, "/test", nil)
		require.Len(t, results, 1)
		assert.Equal(t, "/test/enum/2", results[0].Reference)
		assert.Contains(t, results[0].Message, "'1' is a duplicate value")
	})

	t.Run("multi-type union deduplicates enum not-array error", func(t *testing.T) {
		config := map[string]any{
			"enum": "not-array",
		}
		results := ValidateConfig([]string{"string", "integer", "boolean"}, config, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "'enum' must be an array")
	})

	t.Run("cross-field validation with deduplication", func(t *testing.T) {
		config := map[string]any{
			"minLength": 10,
			"maxLength": 5,
		}
		results := ValidateConfig([]string{"string"}, config, "/test", nil)
		require.Len(t, results, 1)
		assert.Equal(t, "/test", results[0].Reference)
		assert.Contains(t, results[0].Message, "minLength cannot be greater than maxLength")
	})

	t.Run("validator override replaces default for specified type", func(t *testing.T) {
		// Override the object type with a string type validator that allows the 'format' field
		overrides := map[string]TypeConfigValidator{
			"object": &StringTypeConfig{},
		}
		config := map[string]any{
			"format": "date",
		}
		results := ValidateConfig([]string{"object"}, config, "/test", overrides)
		assert.Empty(t, results)
	})

	t.Run("validator override does not affect other types", func(t *testing.T) {
		// add an override for object type but validate a number type
		overrides := map[string]TypeConfigValidator{
			"object": &StringTypeConfig{},
		}
		config := map[string]any{
			"multipleOf": 5,
		}
		results := ValidateConfig([]string{"number"}, config, "/test", overrides)
		assert.Empty(t, results)
	})

	t.Run("nil override falls back to default validators", func(t *testing.T) {
		config := map[string]any{
			"someKey": "value",
		}
		// object with nil override uses default ObjectTypeConfig — config not allowed
		results := ValidateConfig([]string{"object"}, config, "/test", nil)
		require.Len(t, results, 1)
		assert.Contains(t, results[0].Message, "config is not allowed")
	})
}
