package project

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

func TestSubstitutionDiagnostics(t *testing.T) {
	errs := []varsubst.SubstitutionError{
		{
			Name:     "DB_PASSWORD",
			Line:     8,
			Column:   15,
			LineText: "    password: {{ .DB_PASSWORD }}",
			Err:      varsubst.ErrUndefinedVariable,
		},
		{
			Name:     "9PORT",
			Line:     5,
			Column:   11,
			LineText: "    port: {{ .9PORT }}",
			Err:      varsubst.ErrInvalidVarSyntax,
		},
		{
			Name:     "VAR",
			Line:     10,
			Column:   11,
			LineText: "    value: {{ VAR }}",
			Err:      varsubst.ErrInvalidVarSyntax,
		},
	}

	got := substitutionDiagnostics("specs/dest.yaml", errs)

	assert.Equal(t, validation.Diagnostics{
		{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  `undefined variable "DB_PASSWORD"`,
			File:     "specs/dest.yaml",
			Position: pathindex.Position{
				Line:     8,
				Column:   15,
				LineText: "    password: {{ .DB_PASSWORD }}",
			},
		},
		{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  `invalid variable syntax "9PORT"`,
			File:     "specs/dest.yaml",
			Position: pathindex.Position{
				Line:     5,
				Column:   11,
				LineText: "    port: {{ .9PORT }}",
			},
		},
		{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  `invalid variable syntax "VAR"`,
			File:     "specs/dest.yaml",
			Position: pathindex.Position{
				Line:     10,
				Column:   11,
				LineText: "    value: {{ VAR }}",
			},
		},
	}, got)
}

func TestMaskSensitiveLines(t *testing.T) {
	t.Parallel()

	substitution := validation.Diagnostics{
		{
			RuleID:   "project/var-substitution",
			File:     "dest.yaml",
			Position: pathindex.Position{Line: 12, Column: 36, LineText: "    api_key: 'fake-literal-{{ .UNDEF }}'"},
		},
		{
			RuleID:   "project/var-substitution",
			File:     "dest.yaml",
			Position: pathindex.Position{Line: 13, Column: 18, LineText: "    api_secret: '{{ .PW }}'"},
		},
		{
			RuleID:   "project/var-substitution",
			File:     "other.yaml",
			Position: pathindex.Position{Line: 12, Column: 18, LineText: "    api_key: '{{ .OTHER }}'"},
		},
	}
	sensitive := validation.Diagnostics{
		{
			RuleID:    "destination/secret-reference",
			File:      "dest.yaml",
			Position:  pathindex.Position{Line: 12, Column: 5, LineText: "api_key: ***"},
			Sensitive: true,
		},
		{
			RuleID:   "test/plain",
			File:     "dest.yaml",
			Position: pathindex.Position{Line: 13, Column: 5, LineText: "api_secret: '{{ .PW }}'"},
		},
	}

	got := maskSensitiveLines(substitution, sensitive)

	assert.Equal(t, validation.Diagnostics{
		{
			RuleID:   "project/var-substitution",
			File:     "dest.yaml",
			Position: pathindex.Position{Line: 12, Column: 36, LineText: "api_key: ***"},
		},
		substitution[1],
		substitution[2],
	}, got)
	assert.Equal(t, "    api_key: 'fake-literal-{{ .UNDEF }}'", substitution[0].Position.LineText, "the input is left as it was")
}
