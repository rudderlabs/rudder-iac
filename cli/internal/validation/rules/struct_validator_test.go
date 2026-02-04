package rules

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name string `validate:"required"`
}

type CustomTagStruct struct {
	Value string `validate:"mockvalidator"`
}

type OverrideStruct struct {
	Field string `validate:"override"`
}

func TestValidateStruct(t *testing.T) {
	t.Parallel()

	t.Run("valid struct", func(t *testing.T) {
		t.Parallel()

		s := TestStruct{Name: "test"}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("invalid struct", func(t *testing.T) {
		t.Parallel()

		s := TestStruct{Name: ""}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "name", results[0].Field())
		assert.Equal(t, "required", results[0].Tag())
	})

	t.Run("invalid struct with basePath", func(t *testing.T) {
		t.Parallel()

		s := TestStruct{Name: ""}
		results, err := ValidateStruct(s, "/metadata")
		require.NoError(t, err)
		require.Len(t, results, 1)
		// The basePath parameter exists in the function signature but is not used in the current implementation
		// The validator.FieldError just returns the field name, not the full path
		assert.Equal(t, "name", results[0].Field())
		assert.Equal(t, "required", results[0].Tag())
	})

	t.Run("pointer to struct", func(t *testing.T) {
		t.Parallel()

		s := &TestStruct{Name: "test"}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("pointer to pointer to struct creates an invalid validation error", func(t *testing.T) {
		t.Parallel()

		s := &TestStruct{Name: "test"}
		results, err := ValidateStruct(&s, "")
		require.Error(t, err)

		var invalidValidationError *validator.InvalidValidationError
		assert.ErrorAs(t, err, &invalidValidationError)
		assert.Empty(t, results)
	})

	t.Run("with default validator", func(t *testing.T) {
		// NOTE: Cannot use t.Parallel() here because we modify global defaultValidators slice
		// which is not thread-safe

		// Save and restore original state
		originalValidators := defaultValidators
		t.Cleanup(func() {
			defaultValidators = originalValidators
		})

		defaultValidators = nil

		// Mock validator that checks if string starts with "valid-"
		mockValidator := CustomValidateFunc{
			Tag: "mockvalidator",
			Func: func(fl validator.FieldLevel) bool {
				value := fl.Field().String()
				return strings.HasPrefix(value, "valid-")
			},
		}

		RegisterDefaultValidator(mockValidator)

		// Test with valid value
		s := CustomTagStruct{Value: "valid-test"}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		assert.Empty(t, results)

		// Test with invalid value
		s = CustomTagStruct{Value: "invalid-test"}
		results, err = ValidateStruct(s, "")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "value", results[0].Field())
		assert.Equal(t, "mockvalidator", results[0].Tag())
	})

	t.Run("with custom validate func", func(t *testing.T) {
		// Save and restore to be safe, though this test doesn't modify defaultValidators
		originalValidators := defaultValidators
		t.Cleanup(func() {
			defaultValidators = originalValidators
		})

		defaultValidators = nil

		// Mock validator that checks if string length is even
		evenLengthValidator := CustomValidateFunc{
			Tag: "evenlength",
			Func: func(fl validator.FieldLevel) bool {
				value := fl.Field().String()
				return len(value)%2 == 0
			},
		}

		type TestEvenStruct struct {
			Value string `validate:"evenlength"`
		}

		// Test with even length (valid)
		s := TestEvenStruct{Value: "ab"}
		results, err := ValidateStruct(s, "", evenLengthValidator)
		require.NoError(t, err)
		assert.Empty(t, results)

		// Test with odd length (invalid)
		s = TestEvenStruct{Value: "abc"}
		results, err = ValidateStruct(s, "", evenLengthValidator)
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "value", results[0].Field())
		assert.Equal(t, "evenlength", results[0].Tag())
	})
}
