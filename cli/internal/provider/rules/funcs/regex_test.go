package funcs

import (
	"regexp"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatternRegistry(t *testing.T) {
	t.Parallel()

	t.Run("register and get pattern", func(t *testing.T) {
		t.Parallel()
		reg := &patternRegistry{
			patterns: make(map[string]*regexp.Regexp),
			errors:   make(map[string]string),
		}

		reg.Register("test_pattern", "^[A-Z][a-z]*$", "must start with uppercase")

		regex, errMsg, ok := reg.Get("test_pattern")
		assert.True(t, ok)
		assert.NotNil(t, regex)
		assert.Equal(t, "must start with uppercase", errMsg)
	})

	t.Run("get non-existent pattern", func(t *testing.T) {
		t.Parallel()
		reg := &patternRegistry{
			patterns: make(map[string]*regexp.Regexp),
			errors:   make(map[string]string),
		}

		_, _, ok := reg.Get("nonexistent")
		assert.False(t, ok)
	})

	t.Run("Register pattern with invalid regex panics", func(t *testing.T) {
		t.Parallel()
		reg := &patternRegistry{
			patterns: make(map[string]*regexp.Regexp),
			errors:   make(map[string]string),
		}

		// Invalid regex pattern (unclosed bracket) should panic
		assert.Panics(t, func() {
			reg.Register("bad_pattern", "^[A-Z", "error message")
		})
	})
}

func TestNewPattern(t *testing.T) {
	t.Parallel()

	t.Run("register pattern successfully", func(t *testing.T) {
		t.Parallel()

		NewPattern("test_new_pattern_id", "^[a-z_]+$", "must be lowercase with underscores")

		regex, errMsg, ok := registry.Get("test_new_pattern_id")
		assert.True(t, ok)
		assert.NotNil(t, regex)
		assert.Equal(t, "must be lowercase with underscores", errMsg)
	})
}

func TestGetPatternErrorMessage(t *testing.T) {
	t.Parallel()

	t.Run("get error message for registered pattern", func(t *testing.T) {
		t.Parallel()

		NewPattern("test_email_pattern", "^[a-z]+@[a-z]+\\.[a-z]+$", "must be valid email")

		msg, ok := getPatternErrorMessage("test_email_pattern")
		assert.True(t, ok)
		assert.Equal(t, "must be valid email", msg)
	})

	t.Run("Get error message for non-existent pattern", func(t *testing.T) {
		t.Parallel()

		_, ok := getPatternErrorMessage("nonexistent_pattern_xyz123")
		assert.False(t, ok)
	})
}

func TestGetPatternValidator(t *testing.T) {
	t.Run("Pattern validator with valid pattern", func(t *testing.T) {
		NewPattern("test_uppercase_validator", "^[A-Z]+$", "must be all uppercase")

		type TestStruct struct {
			Name string `validate:"pattern=test_uppercase_validator"`
		}

		t.Run("Valid value", func(t *testing.T) {
			data := TestStruct{Name: "HELLO"}
			errs, err := rules.ValidateStruct(data, "", GetPatternValidator())
			require.NoError(t, err)
			assert.Nil(t, errs)
		})

		t.Run("Invalid value", func(t *testing.T) {
			data := TestStruct{Name: "Hello"}
			errs, err := rules.ValidateStruct(data, "", GetPatternValidator())
			require.NoError(t, err)
			assert.NotNil(t, errs)
			assert.Len(t, errs, 1)
		})
	})

	t.Run("Pattern validator with non-existent pattern", func(t *testing.T) {
		type TestStruct struct {
			Name string `validate:"pattern=nonexistent_validator_pattern"`
		}

		data := TestStruct{Name: "anything"}
		errs, err := rules.ValidateStruct(data, "", GetPatternValidator())
		require.NoError(t, err)
		assert.NotNil(t, errs)
		assert.Len(t, errs, 1)
	})
}

func TestPatternValidatorIntegration(t *testing.T) {
	t.Run("Multiple patterns registered", func(t *testing.T) {
		NewPattern("test_snake_case", "^[a-z][a-z0-9_]*$", "must be snake_case")
		NewPattern("test_camel_case", "^[a-z][a-zA-Z0-9]*$", "must be camelCase")
		NewPattern("test_pascal_case", "^[A-Z][a-zA-Z0-9]*$", "must be PascalCase")

		type TestStruct struct {
			SnakeField  string `validate:"pattern=test_snake_case"`
			CamelField  string `validate:"pattern=test_camel_case"`
			PascalField string `validate:"pattern=test_pascal_case"`
		}

		t.Run("All valid", func(t *testing.T) {
			data := TestStruct{
				SnakeField:  "valid_snake_case",
				CamelField:  "validCamelCase",
				PascalField: "ValidPascalCase",
			}
			errs, err := rules.ValidateStruct(data, "", GetPatternValidator())
			require.NoError(t, err)
			assert.Nil(t, errs)
		})

		t.Run("All invalid", func(t *testing.T) {
			data := TestStruct{
				SnakeField:  "InvalidSnake",
				CamelField:  "InvalidCamel",
				PascalField: "invalidPascal",
			}
			errs, err := rules.ValidateStruct(data, "", GetPatternValidator())
			require.NoError(t, err)
			assert.NotNil(t, errs)
			assert.Len(t, errs, 3)
		})
	})
}
