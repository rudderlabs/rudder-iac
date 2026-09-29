package renderer

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// JSONRenderer emits diagnostics as a single JSON document for machine
// consumers — CI gates, editors, and coding agents that run validate, read the
// failures, and correct the specs themselves.
//
// Each Load path renders exactly once (substitution, syntax, or semantic — they
// return immediately after), so one Render call means one document on the
// writer. Rendering nothing is still a document: an empty diagnostics array is
// how a consumer distinguishes "validated clean" from "did not get that far".
type JSONRenderer struct {
	w io.Writer
}

func NewJSONRenderer(w io.Writer) Renderer {
	return &JSONRenderer{w: w}
}

// jsonDiagnostic is the wire shape, kept separate from validation.Diagnostic so
// the internal type stays free to change without breaking consumers parsing this
// output. Examples are deliberately omitted — they would dwarf the diagnostics;
// `rudder-cli docs export-validation-rules` carries them, keyed by ruleId.
type jsonDiagnostic struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

type jsonSummary struct {
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

type jsonOutput struct {
	Diagnostics []jsonDiagnostic `json:"diagnostics"`
	Summary     jsonSummary      `json:"summary"`
}

func (r *JSONRenderer) Render(diagnostics validation.Diagnostics) error {
	out := jsonOutput{Diagnostics: make([]jsonDiagnostic, 0, len(diagnostics))}

	for _, d := range diagnostics {
		switch d.Severity {
		case rules.Error:
			out.Summary.Errors++
		case rules.Warning:
			out.Summary.Warnings++
		}

		out.Diagnostics = append(out.Diagnostics, jsonDiagnostic{
			RuleID:   d.RuleID,
			Severity: d.Severity.String(),
			Message:  d.Message,
			File:     d.File,
			Line:     d.Position.Line,
			Column:   d.Position.Column,
		})
	}

	encoder := json.NewEncoder(r.w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		return fmt.Errorf("encoding diagnostics: %w", err)
	}

	return nil
}
