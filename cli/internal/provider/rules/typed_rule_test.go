package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

type testSpec struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

type strictTestSpec struct {
	RequiredInt int `json:"required_int"`
}

type altSpec struct {
	Name string `json:"name"`
}

func extractReferences(results []rules.ValidationResult) []string {
	refs := make([]string, len(results))
	for i, r := range results {
		refs[i] = r.Reference
	}
	return refs
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

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		expectedExamples,
		NewVariant(
			[]rules.MatchPattern{rules.MatchKind("testKind")},
			func(_ string, _ string, _ map[string]any, _ testSpec) []rules.ValidationResult {
				return nil
			},
		),
	)

	assert.Equal(t, "test-rule-id", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "test rule description", rule.Description())
	assert.Equal(t, expectedExamples, rule.Examples())
	assert.Equal(t, []rules.MatchPattern{rules.MatchKind("testKind")}, rule.AppliesTo())
}

func TestTypedRule_Validate(t *testing.T) {
	t.Parallel()

	t.Run("empty validation result", func(t *testing.T) {
		var (
			capturedSpec                  testSpec
			capturedKind, capturedVersion string
			capturedMetadata              map[string]any
		)

		rule := NewTypedRule(
			"test-rule-id",
			rules.Error,
			"test rule description",
			rules.Examples{},
			NewVariant(
				[]rules.MatchPattern{rules.MatchAll()},
				func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
					capturedKind = kind
					capturedVersion = version
					capturedMetadata = metadata
					capturedSpec = spec
					return []rules.ValidationResult{}
				},
			),
		)

		ctx := createMockContext("testKind", "v0.1", map[string]any{
			"field1": "testValue",
			"field2": 123,
		})

		results := rule.Validate(ctx)

		assert.Empty(t, results)
		assert.Equal(t, "testKind", capturedKind)
		assert.Equal(t, "v0.1", capturedVersion)
		assert.Equal(t, map[string]any{"name": "test"}, capturedMetadata)
		assert.Equal(t, "testValue", capturedSpec.Field1)
		assert.Equal(t, 123, capturedSpec.Field2)
	})
}

func TestTypedRule_Validate_ReferencePrefixing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		validateFuncRefs []string
		expectedRefs     []string
	}{
		{
			name:             "empty reference becomes /spec",
			validateFuncRefs: []string{""},
			expectedRefs:     []string{"/spec"},
		},
		{
			name:             "non-empty reference gets prefixed",
			validateFuncRefs: []string{"/field"},
			expectedRefs:     []string{"/spec/field"},
		},
		{
			name:             "multiple results all get prefixed",
			validateFuncRefs: []string{"/field1", "/field2", ""},
			expectedRefs:     []string{"/spec/field1", "/spec/field2", "/spec"},
		},
		{
			name:             "nested reference gets prefixed",
			validateFuncRefs: []string{"/field/nested"},
			expectedRefs:     []string{"/spec/field/nested"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewTypedRule(
				"test-rule",
				rules.Error,
				"Test rule",
				rules.Examples{},
				NewVariant(
					[]rules.MatchPattern{rules.MatchAll()},
					func(_ string, _ string, _ map[string]any, _ testSpec) []rules.ValidationResult {
						results := make([]rules.ValidationResult, len(tt.validateFuncRefs))
						for i, ref := range tt.validateFuncRefs {
							results[i] = rules.ValidationResult{Reference: ref, Message: "test error"}
						}
						return results
					},
				),
			)

			ctx := createMockContext("testKind", "v1", map[string]any{
				"field1": "value1",
				"field2": 42,
			})

			results := rule.Validate(ctx)

			assert.Len(t, results, len(tt.expectedRefs))
			assert.Equal(t, tt.expectedRefs, extractReferences(results))
		})
	}
}

func TestTypedRule_Validate_MarshalError(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(_ string, _ string, _ map[string]any, _ testSpec) []rules.ValidationResult {
				return nil
			},
		),
	)

	ctx := createMockContext("testKind", "v1", map[string]any{
		"field1": "valid",
		"field2": make(chan int),
	})

	results := rule.Validate(ctx)

	assert.Len(t, results, 1)
	assert.Equal(t, "/spec", results[0].Reference)
	assert.Contains(t, results[0].Message, "failed to marshal spec")
}

func TestTypedRule_Validate_UnmarshalError(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(_ string, _ string, _ map[string]any, _ strictTestSpec) []rules.ValidationResult {
				return nil
			},
		),
	)

	ctx := createMockContext("testKind", "v1", map[string]any{
		"required_int": "not-an-integer",
	})

	results := rule.Validate(ctx)

	assert.Len(t, results, 1)
	assert.Equal(t, "/spec", results[0].Reference)
	assert.Contains(t, results[0].Message, "failed to unmarshal spec")
}

func TestTypedRule_Validate_ParameterPassing(t *testing.T) {
	t.Parallel()

	var (
		receivedKind, receivedVersion string
		receivedMetadata              map[string]any
		receivedSpec                  testSpec
	)

	rule := NewTypedRule(
		"test-rule-id",
		rules.Error,
		"test rule description",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(kind string, version string, metadata map[string]any, spec testSpec) []rules.ValidationResult {
				receivedKind = kind
				receivedVersion = version
				receivedMetadata = metadata
				receivedSpec = spec
				return nil
			},
		),
	)

	ctx := &rules.ValidationContext{
		FilePath: "/test/path.yaml",
		FileName: "path.yaml",
		Spec:     map[string]any{"field1": "expectedValue", "field2": 999},
		Kind:     "properties",
		Version:  "rudder/v1",
		Metadata: map[string]any{"name": "testProperty", "import": true},
	}

	results := rule.Validate(ctx)

	assert.Empty(t, results)
	assert.Equal(t, "properties", receivedKind)
	assert.Equal(t, "rudder/v1", receivedVersion)
	assert.Equal(t, map[string]any{"name": "testProperty", "import": true}, receivedMetadata)
	assert.Equal(t, "expectedValue", receivedSpec.Field1)
	assert.Equal(t, 999, receivedSpec.Field2)
}

func TestSemanticVariant_PassesGraphToFunc(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("test_prop", "property", resources.ResourceData{}, nil))

	var (
		capturedGraph *resources.Graph
		capturedSpec  testSpec
	)

	rule := NewTypedRule(
		"test-rule",
		rules.Error,
		"test",
		rules.Examples{},
		NewSemanticVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(_ string, _ string, _ map[string]any, spec testSpec, g *resources.Graph) []rules.ValidationResult {
				capturedSpec = spec
				capturedGraph = g
				return nil
			},
		),
	)

	ctx := &rules.ValidationContext{
		Spec:     map[string]any{"field1": "hello", "field2": 42},
		Kind:     "testKind",
		Version:  "v1",
		Metadata: map[string]any{},
		Graph:    graph,
	}

	results := rule.Validate(ctx)

	assert.Empty(t, results)
	assert.Same(t, graph, capturedGraph)
	assert.Equal(t, "hello", capturedSpec.Field1)
	assert.Equal(t, 42, capturedSpec.Field2)
}

func TestSemanticVariant_ReferencePrefixing(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"test-rule",
		rules.Error,
		"test",
		rules.Examples{},
		NewSemanticVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(_ string, _ string, _ map[string]any, _ testSpec, _ *resources.Graph) []rules.ValidationResult {
				return []rules.ValidationResult{
					{Reference: "/field1", Message: "error1"},
					{Reference: "/nested/field", Message: "error2"},
				}
			},
		),
	)

	ctx := &rules.ValidationContext{
		Spec:     map[string]any{"field1": "val", "field2": 1},
		Kind:     "test",
		Version:  "v1",
		Metadata: map[string]any{},
		Graph:    resources.NewGraph(),
	}

	results := rule.Validate(ctx)

	assert.Len(t, results, 2)
	assert.Equal(t, "/spec/field1", results[0].Reference)
	assert.Equal(t, "/spec/nested/field", results[1].Reference)
}

func TestSemanticVariant_MarshalError(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"test-rule",
		rules.Error,
		"test",
		rules.Examples{},
		NewSemanticVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(_ string, _ string, _ map[string]any, _ testSpec, _ *resources.Graph) []rules.ValidationResult {
				return nil
			},
		),
	)

	ctx := &rules.ValidationContext{
		Spec:     map[string]any{"field1": make(chan int)},
		Kind:     "test",
		Version:  "v1",
		Metadata: map[string]any{},
		Graph:    resources.NewGraph(),
	}

	results := rule.Validate(ctx)

	assert.Len(t, results, 1)
	assert.Equal(t, "/spec", results[0].Reference)
	assert.Contains(t, results[0].Message, "failed to marshal spec")
}

func TestMultiVariant_DispatchesToMatchingVariant(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"multi-variant-rule",
		rules.Error,
		"multi variant dispatch",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchKindVersion("props", "v1")},
			func(_ string, _ string, _ map[string]any, spec testSpec) []rules.ValidationResult {
				return []rules.ValidationResult{{Reference: "/field1", Message: "v1: " + spec.Field1}}
			},
		),
		NewVariant(
			[]rules.MatchPattern{rules.MatchKindVersion("props", "v2")},
			func(_ string, _ string, _ map[string]any, spec altSpec) []rules.ValidationResult {
				return []rules.ValidationResult{{Reference: "/name", Message: "v2: " + spec.Name}}
			},
		),
	)

	t.Run("dispatches to v1 variant", func(t *testing.T) {
		ctx := &rules.ValidationContext{
			Spec:     map[string]any{"field1": "hello", "field2": 1},
			Kind:     "props",
			Version:  "v1",
			Metadata: map[string]any{},
		}

		results := rule.Validate(ctx)

		assert.Len(t, results, 1)
		assert.Equal(t, "/spec/field1", results[0].Reference)
		assert.Equal(t, "v1: hello", results[0].Message)
	})

	t.Run("dispatches to v2 variant with different spec type", func(t *testing.T) {
		ctx := &rules.ValidationContext{
			Spec:     map[string]any{"name": "world"},
			Kind:     "props",
			Version:  "v2",
			Metadata: map[string]any{},
		}

		results := rule.Validate(ctx)

		assert.Len(t, results, 1)
		assert.Equal(t, "/spec/name", results[0].Reference)
		assert.Equal(t, "v2: world", results[0].Message)
	})

	t.Run("returns nil when no variant matches", func(t *testing.T) {
		ctx := &rules.ValidationContext{
			Spec:     map[string]any{"field1": "hello"},
			Kind:     "props",
			Version:  "v99",
			Metadata: map[string]any{},
		}

		results := rule.Validate(ctx)

		assert.Nil(t, results)
	})
}

func TestMultiVariant_AppliesToReturnsUnionOfPatterns(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"multi-variant-rule",
		rules.Error,
		"test",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchKindVersion("events", "v1")},
			func(_ string, _ string, _ map[string]any, _ testSpec) []rules.ValidationResult {
				return nil
			},
		),
		NewSemanticVariant(
			[]rules.MatchPattern{rules.MatchKindVersion("events", "v2")},
			func(_ string, _ string, _ map[string]any, _ altSpec, _ *resources.Graph) []rules.ValidationResult {
				return nil
			},
		),
	)

	patterns := rule.AppliesTo()

	assert.Len(t, patterns, 2)
	assert.Equal(t, rules.MatchKindVersion("events", "v1"), patterns[0])
	assert.Equal(t, rules.MatchKindVersion("events", "v2"), patterns[1])
}

func TestMultiVariant_FirstMatchWins(t *testing.T) {
	t.Parallel()

	rule := NewTypedRule(
		"first-match-rule",
		rules.Error,
		"test",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchKind("events")},
			func(_ string, _ string, _ map[string]any, _ testSpec) []rules.ValidationResult {
				return []rules.ValidationResult{{Reference: "/first", Message: "first variant"}}
			},
		),
		NewVariant(
			[]rules.MatchPattern{rules.MatchAll()},
			func(_ string, _ string, _ map[string]any, _ testSpec) []rules.ValidationResult {
				return []rules.ValidationResult{{Reference: "/second", Message: "second variant"}}
			},
		),
	)

	ctx := &rules.ValidationContext{
		Spec:     map[string]any{"field1": "val", "field2": 1},
		Kind:     "events",
		Version:  "v1",
		Metadata: map[string]any{},
	}

	results := rule.Validate(ctx)

	assert.Len(t, results, 1)
	assert.Equal(t, "/spec/first", results[0].Reference)
	assert.Equal(t, "first variant", results[0].Message)
}

func TestMultiVariant_MixedSyntacticAndSemantic(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()

	rule := NewTypedRule(
		"mixed-rule",
		rules.Error,
		"test",
		rules.Examples{},
		NewVariant(
			[]rules.MatchPattern{rules.MatchKindVersion("props", "v1")},
			func(_ string, _ string, _ map[string]any, spec testSpec) []rules.ValidationResult {
				return []rules.ValidationResult{{Reference: "/field1", Message: "syntactic: " + spec.Field1}}
			},
		),
		NewSemanticVariant(
			[]rules.MatchPattern{rules.MatchKindVersion("props", "v2")},
			func(_ string, _ string, _ map[string]any, spec altSpec, g *resources.Graph) []rules.ValidationResult {
				if g == nil {
					return []rules.ValidationResult{{Message: "graph is nil"}}
				}
				return []rules.ValidationResult{{Reference: "/name", Message: "semantic: " + spec.Name}}
			},
		),
	)

	t.Run("syntactic variant ignores graph", func(t *testing.T) {
		ctx := &rules.ValidationContext{
			Spec:     map[string]any{"field1": "hello", "field2": 1},
			Kind:     "props",
			Version:  "v1",
			Metadata: map[string]any{},
		}

		results := rule.Validate(ctx)

		assert.Len(t, results, 1)
		assert.Equal(t, "syntactic: hello", results[0].Message)
	})

	t.Run("semantic variant receives graph", func(t *testing.T) {
		ctx := &rules.ValidationContext{
			Spec:     map[string]any{"name": "world"},
			Kind:     "props",
			Version:  "v2",
			Metadata: map[string]any{},
			Graph:    graph,
		}

		results := rule.Validate(ctx)

		assert.Len(t, results, 1)
		assert.Equal(t, "semantic: world", results[0].Message)
	})
}
