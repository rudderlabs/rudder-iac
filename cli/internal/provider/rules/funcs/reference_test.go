package funcs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildLegacyReferenceRegex tests the legacy regex builder for #/<kind>/<group>/<id> format
func TestBuildLegacyReferenceRegex(t *testing.T) {
	t.Parallel()

	t.Run("matches valid legacy references", func(t *testing.T) {
		regex := BuildLegacyReferenceRegex([]string{"custom-types", "properties"})

		assert.True(t, regex.MatchString("#/custom-types/user-traits/email"))
		assert.True(t, regex.MatchString("#/properties/tracking/page_viewed"))
	})

	t.Run("rejects invalid legacy references", func(t *testing.T) {
		regex := BuildLegacyReferenceRegex([]string{"custom-types"})

		assert.False(t, regex.MatchString("#/custom-types/missing-id"))
		assert.False(t, regex.MatchString("#custom-types:email"))
		assert.False(t, regex.MatchString("not-a-reference"))
	})
}

// TestBuildReferenceRegex tests the new regex builder for #kind:id format
func TestBuildReferenceRegex(t *testing.T) {
	t.Parallel()

	t.Run("matches valid references", func(t *testing.T) {
		regex := BuildReferenceRegex([]string{"custom-types", "properties"})

		assert.True(t, regex.MatchString("#custom-types:email"))
		assert.True(t, regex.MatchString("#properties:page_viewed"))
	})

	t.Run("rejects invalid references", func(t *testing.T) {
		regex := BuildReferenceRegex([]string{"custom-types"})

		assert.False(t, regex.MatchString("#/custom-types/group/email"))
		assert.False(t, regex.MatchString("#invalidkind:email"))
		assert.False(t, regex.MatchString("not-a-reference"))
	})
}

// TestNewLegacyReferenceValidateFunc tests the legacy reference validator
func TestNewLegacyReferenceValidateFunc(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		RefField string `json:"refField" validate:"reference"`
	}

	t.Run("success cases", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "valid legacy reference",
				value: "#/custom-types/user-traits/email",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				validationFunc := NewLegacyReferenceValidateFunc(
					[]string{"custom-types"},
				)

				validationErrors, err := rules.ValidateStruct(
					testStruct{RefField: tt.value},
					"",
					validationFunc,
				)

				require.NoError(t, err)
				assert.Empty(t, validationErrors, "Expected %q to be valid", tt.value)
			})
		}
	})

	t.Run("failure cases", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "missing id segment",
				value: "#/custom-types/group",
			},
			{
				name:  "wrong format",
				value: "#custom-types:email",
			},
			{
				name:  "invalid kind",
				value: "#/properties/group/id",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				validationFunc := NewLegacyReferenceValidateFunc([]string{"custom-types"})

				validationErrors, err := rules.ValidateStruct(
					testStruct{RefField: tt.value},
					"",
					validationFunc,
				)

				require.NoError(t, err)
				require.NotEmpty(t, validationErrors, "Expected %q to be invalid", tt.value)
				assert.Contains(
					t, validationErrors[0].Error(), "validation for 'refField' failed on the 'reference' tag")
			})
		}
	})
}

// TestNewReferenceValidateFunc tests the new reference validator
func TestNewReferenceValidateFunc(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		RefField string `json:"refField" validate:"reference"`
	}

	t.Run("success cases", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "valid reference",
				value: "#custom-types:email",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				validationFunc := NewReferenceValidateFunc([]string{"custom-types"})

				validationErrors, err := rules.ValidateStruct(
					testStruct{RefField: tt.value},
					"",
					validationFunc,
				)

				require.NoError(t, err)
				assert.Empty(t, validationErrors, "Expected %q to be valid", tt.value)
			})
		}
	})

	t.Run("failure cases", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "legacy format",
				value: "#/custom-types/group/email",
			},
			{
				name:  "invalid kind",
				value: "#properties:email",
			},
			{
				name:  "not a reference",
				value: "invalid",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				validationFunc := NewReferenceValidateFunc([]string{"custom-types"})

				validationErrors, err := rules.ValidateStruct(
					testStruct{RefField: tt.value},
					"",
					validationFunc,
				)

				require.NoError(t, err)
				require.NotEmpty(t, validationErrors, "Expected %q to be invalid", tt.value)
				assert.Contains(t, validationErrors[0].Error(), "validation for 'refField' failed on the 'reference' tag")
			})
		}
	})
}
