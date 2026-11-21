package project

import (
	"fmt"
	"slices"

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

// maskSensitiveLines gives each diagnostic that shares a file and line with a
// sensitive one that diagnostic's masked line. A substitution error prints its
// line as written, so a literal beside a reference, which a secret warning on
// the same line has masked, would be printed in full right after it.
func maskSensitiveLines(diagnostics, sensitive validation.Diagnostics) validation.Diagnostics {
	type fileLine struct {
		file string
		line int
	}

	masked := make(map[fileLine]string)
	for _, d := range sensitive {
		if d.Sensitive {
			masked[fileLine{d.File, d.Position.Line}] = d.Position.LineText
		}
	}

	out := slices.Clone(diagnostics)
	for i, d := range out {
		if text, ok := masked[fileLine{d.File, d.Position.Line}]; ok {
			out[i].Position.LineText = text
		}
	}
	return out
}
