package rules

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Name string `validate:"required"`
}

func TestValidateStruct(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		s := TestStruct{Name: "test"}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("invalid struct", func(t *testing.T) {
		s := TestStruct{Name: ""}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		require.Len(t, results, 1)
		assert.Equal(t, "name", results[0].Field())
		assert.Equal(t, "required", results[0].Tag())
	})

	t.Run("invalid struct with basePath", func(t *testing.T) {
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
		s := &TestStruct{Name: "test"}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("pointer to pointer to struct creates an invalid validation error", func(t *testing.T) {
		s := &TestStruct{Name: "test"}
		results, err := ValidateStruct(&s, "")
		require.Error(t, err)

		var invalidValidationError *validator.InvalidValidationError
		assert.ErrorAs(t, err, &invalidValidationError)
		assert.Empty(t, results)
	})
}
