package renderer

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
)

// ruleDocRef points at the rule's entry in the generated rule catalog, which is
// the single source of truth for every code. The fragment is the stable code
// (== RuleID), so tooling can resolve a diagnostic's code to its documentation.
const ruleDocBase = "docs/generated/rules.yaml"

// jsonDiagnostic is the stable, machine-readable shape emitted by the JSON
// renderer. Field names and the `code` value (the rule ID) form a contract with
// downstream tooling (editor diagnostics, MCP validate, agents) and must remain
// backward compatible across releases.
type jsonDiagnostic struct {
	// Code is the stable rule identifier (e.g. "datacatalog/properties/spec-syntax-valid").
	// It is the primary key tooling keys off; it never changes for a given rule.
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Kind     string `json:"kind,omitempty"`
	Resource string `json:"resource,omitempty"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	RuleDoc  string `json:"ruleDoc,omitempty"`
}

// jsonOutput wraps the diagnostics array in an object so the schema can grow
// (e.g. summary counts) without breaking consumers that parse the top level.
type jsonOutput struct {
	Diagnostics []jsonDiagnostic `json:"diagnostics"`
}

// JSONRenderer renders validation diagnostics as machine-readable JSON.
type JSONRenderer struct {
	w io.Writer
}

func NewJSONRenderer(w io.Writer) Renderer {
	return &JSONRenderer{w: w}
}

func (r *JSONRenderer) Render(diagnostics validation.Diagnostics) error {
	out := jsonOutput{Diagnostics: make([]jsonDiagnostic, 0, len(diagnostics))}

	for _, d := range diagnostics {
		out.Diagnostics = append(out.Diagnostics, jsonDiagnostic{
			Code:     d.RuleID,
			Severity: d.Severity.String(),
			Message:  d.Message,
			Kind:     d.Kind,
			File:     d.File,
			Line:     d.Position.Line,
			Col:      d.Position.Column,
			RuleDoc:  fmt.Sprintf("%s#%s", ruleDocBase, d.RuleID),
		})
	}

	enc := json.NewEncoder(r.w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encoding diagnostics as JSON: %w", err)
	}
	return nil
}
