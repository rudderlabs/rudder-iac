package ruledoc

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var verifyLog = logger.New("ruledoc-verify")

// verify runs every authored example of a syntactic-phase rule through the real
// engine; semantic-phase examples are skipped (counted, logged once).
func verify(reg rules.Registry, doc docs.DocumentedRules, mode docs.VerifyMode) []error {
	engine, err := validation.NewValidationEngine(reg, verifyLog)
	if err != nil {
		return []error{fmt.Errorf("creating validation engine: %w", err)}
	}

	var (
		errs    []error
		skipped int
	)

	for _, r := range doc.Rules {
		if r.Phase != "syntactic" {
			for _, mb := range r.MatchBehavior {
				skipped += len(mb.Valid) + len(mb.Invalid)
			}
			continue
		}

		for _, mb := range r.MatchBehavior {
			for _, ex := range mb.Invalid {
				produced, err := runSyntactic(engine, ex.Files)
				if err != nil {
					errs = append(errs, fmt.Errorf("rule %s example %s: %w", r.RuleID, ex.ExampleID, err))
					continue
				}
				for _, miss := range docs.MatchInvalid(ex, produced, mode) {
					errs = append(errs, fmt.Errorf("rule %s: %s", r.RuleID, miss))
				}
			}

			for _, ex := range mb.Valid {
				produced, err := runSyntactic(engine, ex.Files)
				if err != nil {
					errs = append(errs, fmt.Errorf("rule %s example %s: %w", r.RuleID, ex.ExampleID, err))
					continue
				}
				for _, miss := range docs.MatchValid(ex, produced) {
					errs = append(errs, fmt.Errorf("rule %s: %s", r.RuleID, miss))
				}
			}
		}
	}

	if skipped > 0 {
		verifyLog.Info("skipped semantic-phase examples (syntactic-only verifier)", "count", skipped)
	}

	return errs
}

// runSyntactic mirrors project.parseSpecs + ValidateSyntax: parse each file,
// collect parse-error diagnostics inline, then run the engine over the
// successfully-parsed specs. The map is keyed by filename so produced
// Diagnostic.File equals the authored example filename.
func runSyntactic(engine validation.ValidationEngine, files map[string]string) ([]docs.ProducedDiagnostic, error) {
	var (
		parsed   = map[string]*specs.RawSpec{}
		produced []docs.ProducedDiagnostic
	)

	for name, content := range files {
		rs := &specs.RawSpec{Data: []byte(content)}
		if _, err := rs.Parse(); err != nil {
			produced = append(produced, docs.ProducedDiagnostic{
				File:     name,
				Severity: rules.Error.String(),
				Message:  fmt.Sprintf("failed to parse spec from path %s: %s", name, err.Error()),
			})
			continue
		}
		parsed[name] = rs
	}

	diags, err := engine.ValidateSyntax(context.Background(), parsed)
	if err != nil {
		return nil, fmt.Errorf("validate syntax: %w", err)
	}

	for _, d := range diags {
		produced = append(produced, docs.ProducedDiagnostic{
			File:     d.File,
			Severity: d.Severity.String(),
			Message:  d.Message,
		})
	}

	return produced, nil
}
