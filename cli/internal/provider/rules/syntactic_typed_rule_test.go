package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

// Test data structures for testing typed rules

type testSpec struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

type strictTestSpec struct {
	RequiredInt int `json:"required_int"`
}

// Helper functions

func extractReferences(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, r := range results {
		refs[i] = r.Reference
	}
	return refs
}

func extractMessages(results []rules.ValidationResult) []string {
	msgs := make([]string, len(results))
	for i, r := range results {
		msgs[i] = r.Message
	}
	return msgs
}

func createMockContext(kind, version string, spec map[string]any) *rules.ValidationContext {
	return &rules.ValidationContext{
		Spec:     spec,
		Kind:     kind,
		Version:  version,
		Metadata: map[string]any{"name": "test"},
	}
}

func TestTypedRule_Metadata(t *testing.T) {
	t.Parallel()

	expectedExamples := rules.Examples{
		Valid:   []string{"valid example"},
		Invalid: []string{"invalid example"},
	}
	validateFunc := func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
		return nil
	}

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		expectedExamples,
		[]rules.MatchPattern{rules.MatchKind("testKind")},
		validateFunc,
	)

	assert.Equal(t, "test-rule-id", rule.ID(), "ID() should return the correct rule ID")
	assert.Equal(t, rules.Error, rule.Severity(), "Severity() should return the correct severity")
	assert.Equal(t, "test rule description", rule.Description(), "Description() should return the correct description")
	assert.Equal(t, expectedExamples, rule.Examples(), "Examples() should return the correct examples")
	assert.Equal(t, []rules.MatchPattern{rules.MatchKind("testKind")}, rule.AppliesTo(), "AppliesTo() should return the correct match patterns")
}

func TestTypedRule_Validate(t *testing.T) {
	t.Parallel()

	t.Run("empty validation result", func(t *testing.T) {

		var (
			capturedSpec                  testSpec
			capturedKind, capturedVersion string
			capturedMetadata              map[string]any
		)

		validateFunc := func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
			capturedKind = kind
			capturedVersion = version
			capturedMetadata = metadata
			capturedSpec = spec
			return []rules.ValidationResult{}
		}

		rule := NewTypedRule(
			"test-rule-id",
			rules.Error,
			"test rule description",
			rules.Examples{},
			[]rules.MatchPattern{rules.MatchAll()},
			validateFunc,
		)

		expectedKind := "testKind"
		expectedVersion := "v0.1"
		expectedMetadata := map[string]any{"name": "test"}
		ctx := createMockContext(expectedKind, expectedVersion, map[string]any{
			"field1": "testValue",
			"field2": 123,
		})

		results := rule.Validate(ctx)

		assert.Empty(t, results, "Valid spec should not produce validation errors")
		assert.Equal(t, expectedKind, capturedKind, "Kind should be passed to validateFunc")
		assert.Equal(t, expectedVersion, capturedVersion, "Version should be passed to validateFunc")
		assert.Equal(t, expectedMetadata, capturedMetadata, "Metadata should be passed to validateFunc")
		assert.Equal(t, "testValue", capturedSpec.Field1, "Field1 should be correctly unmarshaled")
		assert.Equal(t, 123, capturedSpec.Field2, "Field2 should be correctly unmarshaled")
	})
}

func TestTypedRule_Validate_ReferencePrefixing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		validateFuncRefs []string
		expectedRefs     []string
		expectedMsgCount int
	}{
		{
			name:             "empty reference becomes /spec",
			validateFuncRefs: []string{""},
			expectedRefs:     []string{"/spec"},
			expectedMsgCount: 1,
		},
		{
			name:             "non-empty reference gets prefixed",
			validateFuncRefs: []string{"/field"},
			expectedRefs:     []string{"/spec/field"},
			expectedMsgCount: 1,
		},
		{
			name:             "multiple results all get prefixed",
			validateFuncRefs: []string{"/field1", "/field2", ""},
			expectedRefs:     []string{"/spec/field1", "/spec/field2", "/spec"},
			expectedMsgCount: 3,
		},
		{
			name:             "nested reference gets prefixed",
			validateFuncRefs: []string{"/field/nested"},
			expectedRefs:     []string{"/spec/field/nested"},
			expectedMsgCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateFunc := func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
				results := make([]rules.ValidationResult, len(tt.validateFuncRefs))
				for i, ref := range tt.validateFuncRefs {
					results[i] = rules.ValidationResult{
						Reference: ref,
						Message:   "test error",
					}
				}
				return results
			}

			rule := NewTypedRule(
				"test-rule",
				rules.Error,
				"Test rule",
				rules.Examples{},
				[]rules.MatchPattern{rules.MatchAll()},
				validateFunc,
			)

			ctx := createMockContext("testKind", "v1", map[string]any{
				"field1": "value1",
				"field2": 42,
			})

			results := rule.Validate(ctx)

			assert.Len(t, results, tt.expectedMsgCount, "Should return correct number of results")
			actualRefs := extractReferences(results)
			assert.Equal(t, tt.expectedRefs, actualRefs, "References should be correctly prefixed with /spec")
		})
	}
}

func TestTypedRule_Validate_MarshalError(t *testing.T) {
	t.Parallel()

	validateFunc := func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
		return []rules.ValidationResult{}
	}

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		rules.Examples{},
		[]rules.MatchPattern{rules.MatchAll()},
		validateFunc,
	)

	unmarshalableSpec := map[string]any{
		"field1": "valid",
		"field2": make(chan int),
	}

	ctx := createMockContext("testKind", "v1", unmarshalableSpec)

	results := rule.Validate(ctx)

	assert.Len(t, results, 1, "Should return one validation error for marshal failure")
	assert.Equal(t, "/spec", results[0].Reference, "Error reference should be /spec")
	assert.Contains(t, results[0].Message, "failed to marshal spec", "Error message should mention marshal failure")
}

func TestTypedRule_Validate_UnmarshalError(t *testing.T) {
	t.Parallel()

	validateFunc := func(kind string, version string, metadata map[string]any, spec strictTestSpec) []rules.ValidationResult {
		return []rules.ValidationResult{}
	}

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		rules.Examples{},
		[]rules.MatchPattern{rules.MatchAll()},
		validateFunc,
	)

	invalidSpec := map[string]any{
		"required_int": "not-an-integer",
	}

	ctx := createMockContext("testKind", "v1", invalidSpec)

	results := rule.Validate(ctx)

	assert.Len(t, results, 1, "Should return one validation error for unmarshal failure")
	assert.Equal(t, "/spec", results[0].Reference, "Error reference should be /spec")
	assert.Contains(t, results[0].Message, "failed to unmarshal spec", "Error message should mention unmarshal failure")
}

func TestTypedRule_Validate_ParameterPassing(t *testing.T) {
	t.Parallel()

	var receivedKind, receivedVersion string
	var receivedMetadata map[string]any
	var receivedSpec testSpec

	validateFunc := func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
		receivedKind = kind
		receivedVersion = version
		receivedMetadata = metadata
		receivedSpec = spec
		return []rules.ValidationResult{}
	}

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		rules.Examples{},
		[]rules.MatchPattern{rules.MatchAll()},
		validateFunc,
	)

	expectedKind := "properties"
	expectedVersion := "rudder/v1"
	expectedMetadata := map[string]any{
		"name":   "testProperty",
		"import": true,
	}
	expectedSpec := map[string]any{
		"field1": "expectedValue",
		"field2": 999,
	}

	ctx := &rules.ValidationContext{
		FilePath: "/test/path.yaml",
		FileName: "path.yaml",
		Spec:     expectedSpec,
		Kind:     expectedKind,
		Version:  expectedVersion,
		Metadata: expectedMetadata,
		Graph:    nil,
	}

	results := rule.Validate(ctx)

	assert.Empty(t, results, "Validation should succeed")
	assert.Equal(t, expectedKind, receivedKind, "Kind parameter should be passed correctly")
	assert.Equal(t, expectedVersion, receivedVersion, "Version parameter should be passed correctly")
	assert.Equal(t, expectedMetadata, receivedMetadata, "Metadata parameter should be passed correctly")
	assert.Equal(t, "expectedValue", receivedSpec.Field1, "Spec.Field1 should be unmarshaled correctly")
	assert.Equal(t, 999, receivedSpec.Field2, "Spec.Field2 should be unmarshaled correctly")
}
