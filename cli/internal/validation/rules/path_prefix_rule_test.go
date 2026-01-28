package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRule is a configurable mock Rule for testing PathPrefixRule
type testRule struct {
	id          string
	severity    Severity
	description string
	appliesTo   []string
	examples    Examples
	results     []ValidationResult
}

func (r *testRule) ID() string                                         { return r.id }
func (r *testRule) Severity() Severity                                 { return r.severity }
func (r *testRule) Description() string                                { return r.description }
func (r *testRule) AppliesTo() []string                                { return r.appliesTo }
func (r *testRule) Examples() Examples                                 { return r.examples }
func (r *testRule) Validate(ctx *ValidationContext) []ValidationResult { return r.results }

func TestPathPrefixRule_DelegatesAllInterfaceMethods(t *testing.T) {
	t.Parallel()
	inner := &testRule{
		id:          "test-rule",
		severity:    Error,
		description: "Test rule description",
		appliesTo:   []string{"properties", "events"},
		examples: Examples{
			Valid:   []string{"valid example"},
			Invalid: []string{"invalid example"},
		},
	}

	wrapped := NewPathPrefixRule(inner, "/spec")

	assert.Equal(t, "test-rule", wrapped.ID())
	assert.Equal(t, Error, wrapped.Severity())
	assert.Equal(t, "Test rule description", wrapped.Description())
	assert.Equal(t, []string{"properties", "events"}, wrapped.AppliesTo())
	assert.Equal(t, inner.examples, wrapped.Examples())
}

func TestPathPrefixRule_EmptyReferenceRemainsEmpty(t *testing.T) {
	t.Parallel()
	inner := &testRule{
		id: "test-rule",
		results: []ValidationResult{
			{
				RuleID:    "test-rule",
				Severity:  Error,
				Message:   "Test message",
				Reference: "", // Empty reference
			},
		},
	}

	wrapped := NewPathPrefixRule(inner, "/spec")
	results := wrapped.Validate(nil)

	require.Len(t, results, 1)
	assert.Empty(t, results[0].Reference, "Empty reference should remain empty")
}


func TestPathPrefixRule_ResultsGetPrefixed(t *testing.T) {
	t.Parallel()
	inner := &testRule{
		id: "test-rule",
		results: []ValidationResult{
			{
				RuleID:    "test-rule",
				Severity:  Error,
				Message:   "First error",
				Reference: "/properties/0/name",
			},
			{
				RuleID:    "test-rule",
				Severity:  Warning,
				Message:   "Second warning",
				Reference: "/properties/1/type",
			},
			{
				RuleID:    "test-rule",
				Severity:  Info,
				Message:   "Third info",
				Reference: "", // Empty reference should stay empty
			},
		},
	}

	wrapped := NewPathPrefixRule(inner, "/spec")
	results := wrapped.Validate(nil)

	require.Len(t, results, 3)
	assert.Equal(t, "/spec/properties/0/name", results[0].Reference)
	assert.Equal(t, "/spec/properties/1/type", results[1].Reference)
	assert.Empty(t, results[2].Reference, "Empty reference should remain empty")
}

func TestPathPrefixRule_NoResultsFromInnerRule(t *testing.T) {
	t.Parallel()

	inner := &testRule{
		id:      "test-rule",
		results: []ValidationResult{}, // No results
	}

	wrapped := NewPathPrefixRule(inner, "/spec")
	results := wrapped.Validate(nil)

	assert.Empty(t, results)
}

func TestPathPrefixRule_NilResultsFromInnerRule(t *testing.T) {
	t.Parallel()
	inner := &testRule{
		id:      "test-rule",
		results: nil, // Nil results
	}

	wrapped := NewPathPrefixRule(inner, "/spec")
	results := wrapped.Validate(nil)

	assert.Nil(t, results)
}