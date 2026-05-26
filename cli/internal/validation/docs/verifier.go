package docs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// verifierEngine is the slice of ValidationEngine the verifier actually uses.
// The spike runs only ValidateSyntax: the three pilot rules all execute in
// that phase (duplicate-urn as a ProjectRule). Extend if a future pilot needs
// semantic-phase verification.
type verifierEngine interface {
	ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.RawSpec) (validation.Diagnostics, error)
}

// Verifier executes each authored InvalidExample through the validation
// engine and asserts that every ExpectedDiagnostic matches at least one
// produced diagnostic (subset semantics — see spec §5).
type Verifier struct {
	engine verifierEngine
	loader *loader.Loader
}

func NewVerifier(reg rules.Registry, log *logger.Logger) (*Verifier, error) {
	eng, err := validation.NewValidationEngine(reg, log)
	if err != nil {
		return nil, fmt.Errorf("initialising verifier engine: %w", err)
	}
	return &Verifier{engine: eng, loader: &loader.Loader{}}, nil
}

// newVerifierForTest is the test-only constructor used by verifier_test.go.
func newVerifierForTest(eng verifierEngine) *Verifier {
	return &Verifier{engine: eng, loader: &loader.Loader{}}
}

// Verify runs every InvalidExample on every rule and returns an aggregated
// error so authors see all failures at once.
func (v *Verifier) Verify(ctx context.Context, doc *RulesDoc) error {
	var errs []error
	for _, rule := range doc.Rules {
		for _, mb := range rule.MatchBehavior {
			for _, ex := range mb.Invalid {
				if err := v.verifyExample(ctx, ex, rule.RuleID); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return aggregateErrors(errs)
}

func (v *Verifier) verifyExample(ctx context.Context, ex InvalidExample, ruleID string) error {
	prefix := fmt.Sprintf("rule %s example %s", ruleID, ex.ExampleID)

	tmp, err := os.MkdirTemp("", "rulesdoc-verify-*")
	if err != nil {
		return fmt.Errorf("%s: mktemp: %w", prefix, err)
	}
	defer os.RemoveAll(tmp)

	for relPath, body := range ex.Files {
		full := filepath.Join(tmp, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return fmt.Errorf("%s: mkdir %s: %w", prefix, full, err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			return fmt.Errorf("%s: write %s: %w", prefix, full, err)
		}
	}

	rawSpecs, err := v.loader.Load(tmp)
	if err != nil {
		return fmt.Errorf("%s: load: %w", prefix, err)
	}
	for _, rs := range rawSpecs {
		if _, err := rs.Parse(); err != nil {
			return fmt.Errorf("%s: parse: %w", prefix, err)
		}
	}

	produced, err := v.engine.ValidateSyntax(ctx, rawSpecs)
	if err != nil {
		return fmt.Errorf("%s: validate: %w", prefix, err)
	}

	normalized := normalizeDiagnostics(produced, tmp)
	for _, exp := range ex.ExpectedDiagnostics {
		if !matchesAny(exp, normalized) {
			return fmt.Errorf(
				"%s: expected diagnostic not produced — file=%s reference=%s severity=%s message_contains=%q",
				prefix, exp.File, exp.Reference, exp.Severity, exp.MessageContains,
			)
		}
	}
	return nil
}

func normalizeDiagnostics(diags validation.Diagnostics, tmp string) validation.Diagnostics {
	out := make(validation.Diagnostics, 0, len(diags))
	for _, d := range diags {
		nd := d
		if rel, err := filepath.Rel(tmp, d.File); err == nil {
			nd.File = rel
		}
		out = append(out, nd)
	}
	return out
}

// matchesAny returns true if any produced diagnostic matches the authored
// expectation on (Severity, File, MessageContains substring).
//
// FUTURE: validation.Diagnostic does not retain the original rule Reference
// (it resolves to Position at engine time). Reference is currently authoring
// metadata only. If pilot authoring shows this ambiguity is a problem, add
// Reference to validation.Diagnostic and compare it here.
func matchesAny(exp ExpectedDiagnostic, produced validation.Diagnostics) bool {
	for _, d := range produced {
		if d.Severity.String() != exp.Severity || d.File != exp.File {
			continue
		}
		if exp.MessageContains != "" && !strings.Contains(d.Message, exp.MessageContains) {
			continue
		}
		return true
	}
	return false
}

func aggregateErrors(errs []error) error {
	if len(errs) == 1 {
		return errs[0]
	}
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = "- " + e.Error()
	}
	return fmt.Errorf("verifier found %d failure(s):\n%s", len(errs), strings.Join(msgs, "\n"))
}
