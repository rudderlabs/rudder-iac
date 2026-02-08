package funcs

import (
	"reflect"
	"testing"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

// mockFieldError is a mock implementation of validator.FieldError for testing
type mockFieldError struct {
	field     string
	tag       string
	actualTag string
	kind      reflect.Kind
	param     string
}

func (m mockFieldError) Tag() string             { return m.tag }
func (m mockFieldError) ActualTag() string       { return m.actualTag }
func (m mockFieldError) Namespace() string       { return "" }
func (m mockFieldError) StructNamespace() string { return "" }
func (m mockFieldError) Field() string           { return m.field }
func (m mockFieldError) StructField() string     { return "" }
func (m mockFieldError) Value() any              { return nil }
func (m mockFieldError) Param() string           { return m.param }
func (m mockFieldError) Kind() reflect.Kind      { return m.kind }
func (m mockFieldError) Type() reflect.Type      { return nil }
func (m mockFieldError) Translate(translator ut.Translator) string {
	return ""
}
func (m mockFieldError) Error() string {
	return "validation for '" + m.field + "' failed on the '" + m.actualTag + "' tag"
}

// TestNamespaceToJSONPointer tests the namespace to JSON Pointer conversion
func TestNamespaceToJSONPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "removes root struct name",
			namespace: "Spec.name",
			expected:  "/name",
		},
		{
			name:      "converts array indices",
			namespace: "Spec.inners[1].surname",
			expected:  "/inners/1/surname",
		},
		{
			name:      "converts nested fields",
			namespace: "Spec.metadata.tags",
			expected:  "/metadata/tags",
		},
		{
			name:      "handles multiple array indices",
			namespace: "Spec.items[0].subitems[2].value",
			expected:  "/items/0/subitems/2/value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := namespaceToJSONPointer(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetErrorMessage tests error message generation for different validation tags
func TestGetErrorMessage(t *testing.T) {
	t.Parallel()

	// Register a test pattern for the pattern tag test case
	NewPattern("test_pattern_for_error_msg", "^[A-Z]+$", "must match test pattern")

	tests := []struct {
		name     string
		err      validator.FieldError
		expected string
	}{
		{
			name: "required tag",
			err: mockFieldError{
				field:     "Name",
				actualTag: "required",
			},
			expected: "'Name' is required",
		},
		{
			name: "reference tag",
			err: mockFieldError{
				field:     "Ref",
				actualTag: "reference",
			},
			expected: "'Ref' is not a valid reference format",
		},
		{
			name: "primitive_or_reference tag",
			err: mockFieldError{
				field:     "Type",
				actualTag: "primitive_or_reference",
			},
			expected: "'Type' is not a valid primitive type or reference format",
		},
		{
			name: "gte tag with number",
			err: mockFieldError{
				field:     "Count",
				actualTag: "gte",
				kind:      reflect.Int,
				param:     "1",
			},
			expected: "'Count' must be greater than or equal to 1",
		},
		{
			name: "gte tag with string",
			err: mockFieldError{
				field:     "Value",
				actualTag: "gte",
				kind:      reflect.String,
				param:     "5",
			},
			expected: "'Value' length must be greater than or equal to 5",
		},
		{
			name: "lte tag with number",
			err: mockFieldError{
				field:     "Max",
				actualTag: "lte",
				kind:      reflect.Int,
				param:     "100",
			},
			expected: "'Max' must be less than or equal to 100",
		},
		{
			name: "lte tag with string",
			err: mockFieldError{
				field:     "Text",
				actualTag: "lte",
				kind:      reflect.String,
				param:     "50",
			},
			expected: "'Text' length must be less than or equal to 50",
		},
		{
			name: "pattern tag with registered pattern",
			err: mockFieldError{
				field:     "Name",
				actualTag: "pattern",
				param:     "test_pattern_for_error_msg",
			},
			expected: "'Name' is not valid: must match test pattern",
		},
		{
			name: "pattern tag with non-existent pattern",
			err: mockFieldError{
				field:     "Name",
				actualTag: "pattern",
				param:     "nonexistent_pattern",
			},
			expected: "'Name' does not match the required pattern",
		},
		{
			name: "oneof tag",
			err: mockFieldError{
				field:     "EventType",
				actualTag: "oneof",
				param:     "track screen identify group page",
			},
			expected: "'EventType' must be one of [track screen identify group page]",
		},
		{
			name: "min tag with number",
			err: mockFieldError{
				field:     "Priority",
				actualTag: "min",
				kind:      reflect.Int,
				param:     "1",
			},
			expected: "'Priority' must be greater than or equal to 1",
		},
		{
			name: "min tag with string",
			err: mockFieldError{
				field:     "Label",
				actualTag: "min",
				kind:      reflect.String,
				param:     "3",
			},
			expected: "'Label' length must be greater than or equal to 3",
		},
		{
			name: "min tag with slice",
			err: mockFieldError{
				field:     "Items",
				actualTag: "min",
				kind:      reflect.Slice,
				param:     "1",
			},
			expected: "'Items' length must be greater than or equal to 1",
		},
		{
			name: "min tag with array",
			err: mockFieldError{
				field:     "Tags",
				actualTag: "min",
				kind:      reflect.Array,
				param:     "2",
			},
			expected: "'Tags' length must be greater than or equal to 2",
		},
		{
			name: "max tag with number",
			err: mockFieldError{
				field:     "Retries",
				actualTag: "max",
				kind:      reflect.Int,
				param:     "5",
			},
			expected: "'Retries' must be less than or equal to 5",
		},
		{
			name: "max tag with string",
			err: mockFieldError{
				field:     "Title",
				actualTag: "max",
				kind:      reflect.String,
				param:     "100",
			},
			expected: "'Title' length must be less than or equal to 100",
		},
		{
			name: "max tag with slice",
			err: mockFieldError{
				field:     "Variants",
				actualTag: "max",
				kind:      reflect.Slice,
				param:     "1",
			},
			expected: "'Variants' length must be less than or equal to 1",
		},
		{
			name: "max tag with array",
			err: mockFieldError{
				field:     "Choices",
				actualTag: "max",
				kind:      reflect.Array,
				param:     "10",
			},
			expected: "'Choices' length must be less than or equal to 10",
		},
		{
			name: "eq tag",
			err: mockFieldError{
				field:     "Type",
				actualTag: "eq",
				param:     "discriminator",
			},
			expected: "'Type' must equal 'discriminator'",
		},
		{
			name: "unknown tag",
			err: mockFieldError{
				field:     "Field",
				actualTag: "customtag",
			},
			expected: "'Field' is not valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getErrorMessage(tt.err)
			assert.Contains(t, result, tt.expected)
		})
	}
}

// TestParseValidationErrors tests the full validation error parsing flow
func TestParseValidationErrors(t *testing.T) {
	t.Parallel()

	t.Run("parses multiple validation errors", func(t *testing.T) {
		errs := validator.ValidationErrors{
			mockFieldError{
				field:     "Name",
				actualTag: "required",
			},
			mockFieldError{
				field:     "Email",
				actualTag: "required",
			},
		}

		results := ParseValidationErrors(errs)
		assert.Len(t, results, 2)
		assert.Contains(t, results[0].Message, "'Name' is required")
		assert.Contains(t, results[1].Message, "'Email' is required")
	})

	t.Run("returns empty for no errors", func(t *testing.T) {
		errs := validator.ValidationErrors{}
		results := ParseValidationErrors(errs)
		assert.Empty(t, results)
	})
}
