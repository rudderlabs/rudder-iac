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
