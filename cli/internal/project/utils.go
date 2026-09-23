package project

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// substitutionDiagnostics converts variable-substitution errors into project
// validation diagnostics. Lives here (not in varsubst) so the substitution
// engine stays free of any dependency on the validation/diagnostics layer.
func substitutionDiagnostics(filePath string, errs []varsubst.SubstitutionError) validation.Diagnostics {
	diagnostics := make(validation.Diagnostics, 0, len(errs))
	for _, e := range errs {
		diagnostics = append(diagnostics, validation.Diagnostic{
			RuleID:   "project/var-substitution",
			Severity: rules.Error,
			Message:  fmt.Sprintf("%s %q", e.Err, e.Name),
			File:     filePath,
			Position: pathindex.Position{
				Line:     e.Line,
				Column:   e.Column,
				LineText: e.LineText,
			},
		})
	}
	return diagnostics
}
