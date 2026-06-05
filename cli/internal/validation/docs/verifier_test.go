package docs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchExample_SubsetHit(t *testing.T) {
	ex := InvalidExample{
		ExampleID: "ex1",
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Severity: "error", MessageContains: "name is required"},
		},
	}
	produced := []ProducedDiagnostic{
		{File: "a.yaml", Severity: "error", Message: "spec.name is required"},
		{File: "a.yaml", Severity: "warning", Message: "unrelated"},
	}

	misses := MatchInvalid(ex, produced, ModeSubset)
	assert.Empty(t, misses)
}

func TestMatchExample_SubsetMiss(t *testing.T) {
	ex := InvalidExample{
		ExampleID: "ex2",
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Severity: "error", MessageContains: "name is required"},
		},
	}
	produced := []ProducedDiagnostic{
		{File: "a.yaml", Severity: "error", Message: "something else"},
	}

	misses := MatchInvalid(ex, produced, ModeSubset)
	assert.Len(t, misses, 1)
}

func TestMatchExample_EmptyMessageContains_MatchesOnFileSeverity(t *testing.T) {
	ex := InvalidExample{
		ExampleID: "ex3",
		ExpectedDiagnostics: []ExpectedDiagnostic{
			{File: "a.yaml", Severity: "error", MessageContains: ""},
		},
	}
	produced := []ProducedDiagnostic{
		{File: "a.yaml", Severity: "error", Message: "anything"},
	}

	misses := MatchInvalid(ex, produced, ModeSubset)
	assert.Empty(t, misses)
}

func TestMatchValid_ProducesDiagnostic_Fails(t *testing.T) {
	ex := ValidExample{
		ExampleID: "v",
	}
	produced := []ProducedDiagnostic{
		{File: "a.yaml", Severity: "error", Message: "boom"},
	}

	misses := MatchValid(ex, produced)
	assert.NotEmpty(t, misses)
}
