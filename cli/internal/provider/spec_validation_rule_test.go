package provider

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSpecFactory is a test implementation of SpecFactory
type mockSpecFactory struct {
	kind          string
	specFieldName string
	examples      rules.Examples
}

func (m *mockSpecFactory) Kind() string {
	return m.kind
}

func (m *mockSpecFactory) NewSpec() any {
	return &mockSpec{}
}

func (m *mockSpecFactory) SpecFieldName() string {
	return m.specFieldName
}

func (m *mockSpecFactory) Examples() rules.Examples {
	return m.examples
}

// mockSpec is a test spec structure with validation tags
type mockSpec struct {
	Items []mockItem `json:"items" validate:"required,dive"`
}

type mockItem struct {
	ID   string `json:"id" validate:"required"`
	Name string `json:"name" validate:"required"`
}

func TestSpecValidationRule_ID(t *testing.T) {
	factory := &mockSpecFactory{kind: "test-kind"}
	rule := NewSpecValidationRule(factory)

	assert.Equal(t, "test-kind/spec-syntax-valid", rule.ID())
}

func TestSpecValidationRule_Severity(t *testing.T) {
	factory := &mockSpecFactory{kind: "test-kind"}
	rule := NewSpecValidationRule(factory)

	assert.Equal(t, rules.Error, rule.Severity())
}

func TestSpecValidationRule_Description(t *testing.T) {
	factory := &mockSpecFactory{kind: "test-kind"}
	rule := NewSpecValidationRule(factory)

	assert.Equal(t, "validates test-kind spec structure", rule.Description())
}

func TestSpecValidationRule_AppliesTo(t *testing.T) {
	factory := &mockSpecFactory{kind: "test-kind"}
	rule := NewSpecValidationRule(factory)

	assert.Equal(t, []string{"test-kind"}, rule.AppliesTo())
}

func TestSpecValidationRule_Examples(t *testing.T) {
	examples := rules.Examples{
		Valid:   []string{"valid example"},
		Invalid: []string{"invalid example"},
	}
	factory := &mockSpecFactory{
		kind:     "test-kind",
		examples: examples,
	}
	rule := NewSpecValidationRule(factory)

	assert.Equal(t, examples, rule.Examples())
}

func TestSpecValidationRule_Validate_ValidSpec(t *testing.T) {
	factory := &mockSpecFactory{
		kind:          "test-kind",
		specFieldName: "items",
	}
	rule := NewSpecValidationRule(factory)

	ctx := &rules.ValidationContext{
		Spec: map[string]any{
			"items": []any{
				map[string]any{
					"id":   "item1",
					"name": "Item 1",
				},
				map[string]any{
					"id":   "item2",
					"name": "Item 2",
				},
			},
		},
	}

	results := rule.Validate(ctx)
	assert.Empty(t, results, "expected no validation errors for valid spec")
}

func TestSpecValidationRule_Validate_MissingRequiredField(t *testing.T) {
	factory := &mockSpecFactory{
		kind:          "test-kind",
		specFieldName: "items",
	}
	rule := NewSpecValidationRule(factory)

	ctx := &rules.ValidationContext{
		Spec: map[string]any{
			"items": []any{
				map[string]any{
					"id": "item1",
					// Missing required "name" field
				},
			},
		},
	}

	results := rule.Validate(ctx)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "name")
	assert.Contains(t, results[0].Message, "required")
	assert.Equal(t, "/items/0/name", results[0].Reference)
}

func TestSpecValidationRule_Validate_InvalidStructure(t *testing.T) {
	factory := &mockSpecFactory{
		kind:          "test-kind",
		specFieldName: "items",
	}
	rule := NewSpecValidationRule(factory)

	ctx := &rules.ValidationContext{
		Spec: map[string]any{
			"items": "invalid-not-an-array",
		},
	}

	results := rule.Validate(ctx)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "spec structure should be valid")
	assert.Equal(t, "/items", results[0].Reference)
}

func TestSpecValidationRule_Validate_EmptySpec(t *testing.T) {
	factory := &mockSpecFactory{
		kind:          "test-kind",
		specFieldName: "items",
	}
	rule := NewSpecValidationRule(factory)

	ctx := &rules.ValidationContext{
		Spec: map[string]any{},
	}

	results := rule.Validate(ctx)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Message, "items")
	assert.Contains(t, results[0].Message, "required")
}

func TestSpecValidationRule_ImplementsRuleInterface(t *testing.T) {
	factory := &mockSpecFactory{kind: "test-kind"}
	rule := NewSpecValidationRule(factory)

	// Compile-time check that rule implements rules.Rule
	var _ rules.Rule = rule
}
