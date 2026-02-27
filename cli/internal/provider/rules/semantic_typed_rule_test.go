package rules

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestSemanticTypedRule_Metadata(t *testing.T) {
	t.Parallel()

	expectedExamples := rules.Examples{
		Valid:   []string{"valid example"},
		Invalid: []string{"invalid example"},
	}

	rule := NewSemanticTypedRule(
		"semantic-test-rule",
		rules.Error,
		"semantic test rule description",
		expectedExamples,
		[]string{"testKind"},
		func(_ string, _ string, _ map[string]any, _ testSpec, _ *resources.Graph) []rules.ValidationResult {
			return nil
		},
	)

	assert.Equal(t, "semantic-test-rule", rule.ID())
	assert.Equal(t, rules.Error, rule.Severity())
	assert.Equal(t, "semantic test rule description", rule.Description())
	assert.Equal(t, expectedExamples, rule.Examples())
	assert.Equal(t, []string{"testKind"}, rule.AppliesToKinds())
}

func TestSemanticTypedRule_Validate_PassesGraphToFunc(t *testing.T) {
	t.Parallel()

	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("test_prop", "property", resources.ResourceData{}, nil))

	var capturedGraph *resources.Graph
	var capturedSpec testSpec

	rule := NewSemanticTypedRule(
		"test-rule",
		rules.Error,
		"test",
		rules.Examples{},
		[]string{"*"},
		func(_ string, _ string, _ map[string]any, spec testSpec, g *resources.Graph) []rules.ValidationResult {
			capturedSpec = spec
			capturedGraph = g
			return nil
		},
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
	assert.Same(t, graph, capturedGraph, "Graph should be passed to validate function")
	assert.Equal(t, "hello", capturedSpec.Field1)
	assert.Equal(t, 42, capturedSpec.Field2)
}

func TestSemanticTypedRule_Validate_ReferencePrefixing(t *testing.T) {
	t.Parallel()

	rule := NewSemanticTypedRule(
		"test-rule",
		rules.Error,
		"test",
		rules.Examples{},
		[]string{"*"},
		func(_ string, _ string, _ map[string]any, _ testSpec, _ *resources.Graph) []rules.ValidationResult {
			return []rules.ValidationResult{
				{Reference: "/field1", Message: "error1"},
				{Reference: "/nested/field", Message: "error2"},
			}
		},
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

func TestSemanticTypedRule_Validate_MarshalError(t *testing.T) {
	t.Parallel()

	rule := NewSemanticTypedRule(
		"test-rule",
		rules.Error,
		"test",
		rules.Examples{},
		[]string{"*"},
		func(_ string, _ string, _ map[string]any, _ testSpec, _ *resources.Graph) []rules.ValidationResult {
			return nil
		},
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
