package validation

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
)

func TestDiagnostics_HasErrorsAndWarnings(t *testing.T) {
	tests := []struct {
		name        string
		diagnostics Diagnostics
		hasErrors   bool
		hasWarnings bool
	}{
		{
			name:        "empty diagnostics",
			diagnostics: Diagnostics{},
			hasErrors:   false,
			hasWarnings: false,
		},
		{
			name: "only errors",
			diagnostics: Diagnostics{
				{Severity: rules.Error},
				{Severity: rules.Error},
			},
			hasErrors:   true,
			hasWarnings: false,
		},
		{
			name: "only warnings",
			diagnostics: Diagnostics{
				{Severity: rules.Warning},
				{Severity: rules.Warning},
			},
			hasErrors:   false,
			hasWarnings: true,
		},
		{
			name: "only info",
			diagnostics: Diagnostics{
				{Severity: rules.Info},
			},
			hasErrors:   false,
			hasWarnings: false,
		},
		{
			name: "mixed errors and warnings",
			diagnostics: Diagnostics{
				{Severity: rules.Error},
				{Severity: rules.Warning},
				{Severity: rules.Info},
			},
			hasErrors:   true,
			hasWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasErrors, tt.diagnostics.HasErrors())
			assert.Equal(t, tt.hasWarnings, tt.diagnostics.HasWarnings())
		})
	}
}

func TestDiagnostics_Sort(t *testing.T) {
	diagnostics := Diagnostics{
		{
			File:     "b.yaml",
			Position: pathindex.Position{Line: 1, Column: 1},
		},
		{
			File:     "a.yaml",
			Position: pathindex.Position{Line: 5, Column: 10},
		},
		{
			File:     "a.yaml",
			Position: pathindex.Position{Line: 5, Column: 5},
		},
		{
			File:     "a.yaml",
			Position: pathindex.Position{Line: 2, Column: 1},
		},
	}

	diagnostics.Sort()

	assert.Equal(t, "a.yaml", diagnostics[0].File)
	assert.Equal(t, 2, diagnostics[0].Position.Line)

	assert.Equal(t, "a.yaml", diagnostics[1].File)
	assert.Equal(t, 5, diagnostics[1].Position.Line)
	assert.Equal(t, 5, diagnostics[1].Position.Column)

	assert.Equal(t, "a.yaml", diagnostics[2].File)
	assert.Equal(t, 5, diagnostics[2].Position.Line)
	assert.Equal(t, 10, diagnostics[2].Position.Column)

	assert.Equal(t, "b.yaml", diagnostics[3].File)
}

func TestDiagnostics_ErrorsAndWarnings(t *testing.T) {
	diagnostics := Diagnostics{
		{RuleID: "rule1", Severity: rules.Error},
		{RuleID: "rule2", Severity: rules.Warning},
		{RuleID: "rule3", Severity: rules.Info},
		{RuleID: "rule4", Severity: rules.Error},
		{RuleID: "rule5", Severity: rules.Warning},
	}

	errors := diagnostics.Errors()
	assert.Len(t, errors, 2)
	assert.Equal(t, rules.Error, errors[0].Severity)
	assert.Equal(t, rules.Error, errors[1].Severity)
	assert.Equal(t, "rule1", errors[0].RuleID)
	assert.Equal(t, "rule4", errors[1].RuleID)

	warnings := diagnostics.Warnings()
	assert.Len(t, warnings, 2)
	assert.Equal(t, rules.Warning, warnings[0].Severity)
	assert.Equal(t, rules.Warning, warnings[1].Severity)
	assert.Equal(t, "rule2", warnings[0].RuleID)
	assert.Equal(t, "rule5", warnings[1].RuleID)
}
