package docs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
)

// Verifier executes every InvalidExample on every documented rule through
// the validation engine and asserts that the authored ExpectedDiagnostics
// each match at least one produced diagnostic (subset semantics, spec §5).
//
// The engine is injected by factory so tests can swap in a fake without
// touching the file-materialization helper.
type Verifier struct {
	engineFactory func() validation.ValidationEngine
}

func NewVerifier(factory func() validation.ValidationEngine) *Verifier {
	return &Verifier{engineFactory: factory}
}

func (v *Verifier) Verify(doc *RulesDoc) error {
	for _, rule := range doc.Rules {
		for _, mb := range rule.MatchBehavior {
			for _, ex := range mb.Invalid {
				if err := v.verifyExample(rule.RuleID, ex); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *Verifier) verifyExample(ruleID string, ex InvalidExample) error {
	diags, err := v.runEngineOnFiles(ex.Files)
	if err != nil {
		return fmt.Errorf("rule %s example %s: %w", ruleID, ex.ExampleID, err)
	}
	for _, expected := range ex.ExpectedDiagnostics {
		if !matchesAny(expected, diags) {
			return fmt.Errorf(
				"rule %s example %s: no produced diagnostic matched expected {file=%s reference=%s severity=%s message_contains=%q}",
				ruleID, ex.ExampleID,
				expected.File, expected.Reference, expected.Severity, expected.MessageContains,
			)
		}
	}
	return nil
}

func (v *Verifier) runEngineOnFiles(files map[string]string) (validation.Diagnostics, error) {
	tmp, err := os.MkdirTemp("", "ruledoc-verify-*")
	if err != nil {
		return nil, fmt.Errorf("creating tmpdir: %w", err)
	}
	defer os.RemoveAll(tmp)

	for name, body := range files {
		full := filepath.Join(tmp, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("creating subdir for %s: %w", name, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", name, err)
		}
	}

	l := &loader.Loader{}
	rawSpecs, err := l.Load(tmp)
	if err != nil {
		return nil, fmt.Errorf("loading tmpdir: %w", err)
	}

	// Loader keys specs by absolute path. Authors write File: "main.yaml"
	// (relative). Rewrite the map keys before calling the engine so the
	// File field on produced diagnostics matches the author's relative name.
	relSpecs := make(map[string]*specs.RawSpec, len(rawSpecs))
	for k, vRaw := range rawSpecs {
		rel := strings.TrimPrefix(k, tmp+string(filepath.Separator))
		relSpecs[rel] = vRaw
	}

	engine := v.engineFactory()
	return engine.ValidateSyntax(context.Background(), relSpecs)
}

// matchesAny implements the subset-match semantics (spec §5): an expected
// diagnostic matches any produced diagnostic that agrees on file, severity,
// and (when set) message substring. The Reference comparison is deliberately
// omitted in the spike — the Diagnostic struct does not carry a Reference
// field; the rule's Reference goes in via ValidationResult and is converted
// to a Position before emit. TODO(spike DEX-370): match Reference once
// Diagnostic.Reference exists.
func matchesAny(expected ExpectedDiagnostic, produced validation.Diagnostics) bool {
	for _, d := range produced {
		if d.File != expected.File {
			continue
		}
		if d.Severity.String() != expected.Severity {
			continue
		}
		if expected.MessageContains != "" && !strings.Contains(d.Message, expected.MessageContains) {
			continue
		}
		return true
	}
	return false
}
