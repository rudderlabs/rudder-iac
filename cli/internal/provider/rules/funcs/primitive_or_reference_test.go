package funcs

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsPrimitive tests the isPrimitive helper function
func TestIsPrimitive(t *testing.T) {
	t.Parallel()

	primitives := []string{"string", "number", "integer", "boolean", "array", "object", "null"}

	t.Run("value in primitives", func(t *testing.T) {
		result := isPrimitive(" string , number , integer ", primitives)
		assert.True(t, result)
	})

	t.Run("value not in primitives", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "empty string",
				value: "",
			},
			{
				name:  "invalid primitive",
				value: "invalid",
			},
			{
				name:  "mixed valid and invalid",
				value: "string, invalid",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := isPrimitive(tt.value, primitives)
				assert.False(t, result, "Expected %q to be invalid primitive(s)", tt.value)
			})
		}
	})
}

// TestNewPrimitiveOrReference tests the NewPrimitiveOrReference validator function
func TestNewPrimitiveOrReference(t *testing.T) {
	t.Parallel()

	type testStruct struct {
		TypeField string `json:"typeField" validate:"primitive_or_reference"`
	}

	standardPrimitives := []string{
		"string",
		"number",
		"integer",
		"boolean",
		"array",
		"object",
		"null",
	}
	legacyRefRegex := BuildLegacyReferenceRegex([]string{"custom-types"})

	t.Run("success cases", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "valid single primitive",
				value: "string",
			},
			{
				name:  "valid comma-separated primitives",
				value: "string, number",
			},
			{
				name:  "valid legacy reference",
				value: "#/custom-types/user-traits/email",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				validationFunc := NewPrimitiveOrReference(
					standardPrimitives,
					legacyRefRegex,
				)

				validationErrors, err := rules.ValidateStruct(
					testStruct{TypeField: tt.value},
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
				name:  "invalid primitive",
				value: "invalidtype",
			},
			{
				name:  "empty primitive",
				value: "",
			},
			{
				name:  "mixed valid and invalid primitives",
				value: "string, xyz",
			},
			{
				name:  "invalid legacy reference missing id",
				value: "#/custom-types/group",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				validationFunc := NewPrimitiveOrReference(
					standardPrimitives,
					legacyRefRegex,
				)

				validationErrors, err := rules.ValidateStruct(
					testStruct{TypeField: tt.value},
					"",
					validationFunc,
				)

				require.NoError(t, err)
				require.NotEmpty(t, validationErrors, "Expected %q to be invalid", tt.value)
				assert.Contains(t, validationErrors[0].Error(), "validation for 'typeField' failed on the 'primitive_or_reference' tag")
			})
		}
	})
}
