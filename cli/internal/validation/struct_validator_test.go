package validation

import (
	"testing"

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
		assert.Len(t, results, 1)
		assert.Equal(t, "/name", results[0].Reference)
		assert.Equal(t, "'name' is required", results[0].Message)
	})

	t.Run("invalid struct with basePath", func(t *testing.T) {
		s := TestStruct{Name: ""}
		results, err := ValidateStruct(s, "/metadata")
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "/metadata/name", results[0].Reference)
		assert.Equal(t, "'name' is required", results[0].Message)
	})

	t.Run("pointer to struct", func(t *testing.T) {
		s := &TestStruct{Name: "test"}
		results, err := ValidateStruct(s, "")
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("pointer to pointer to struct should not panic", func(t *testing.T) {
		s := &TestStruct{Name: "test"}
		assert.NotPanics(t, func() {
			ValidateStruct(&s, "")
		})
	})
}
