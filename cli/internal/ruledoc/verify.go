package ruledoc

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var verifyLog = logger.New("ruledoc-verify")

// verify runs every authored example through the real engine. Syntactic-phase
// examples go through ValidateSyntax; semantic-phase examples are loaded into a
// fresh provider (built by newProvider) and run through the full
// parse -> syntax -> graph -> semantic pipeline.
//
// newProvider supplies an isolated provider per semantic example: providers are
// stateful (LoadSpec accumulates resources, ResourceGraph reads that state), so
// each example needs its own. It originates in package app — which can build
// providers — and is threaded down as a plain func value so ruledoc need not
// import app. When nil, encountering a semantic example is a hard error rather
// than a panic; syntactic-only callers (most unit tests) never trip it.
func verify(reg rules.Registry, doc docs.DocumentedRules, mode docs.VerifyMode, newProvider func() (provider.Provider, error)) []error {
	engine, err := validation.NewValidationEngine(reg, verifyLog)
	if err != nil {
		return []error{fmt.Errorf("creating validation engine: %w", err)}
	}

	var errs []error

	for _, r := range doc.Rules {
		semantic := r.Phase == "semantic"

		for _, mb := range r.MatchBehavior {
			for _, ex := range mb.Invalid {
				produced, err := runExample(engine, ex.Files, semantic, newProvider)
				if err != nil {
					errs = append(errs, fmt.Errorf("rule %s example %s: %w", r.RuleID, ex.ExampleID, err))
					continue
				}
				for _, miss := range docs.MatchInvalid(ex, produced, mode) {
					errs = append(errs, fmt.Errorf("rule %s: %s", r.RuleID, miss))
				}
			}

			for _, ex := range mb.Valid {
				produced, err := runExample(engine, ex.Files, semantic, newProvider)
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

	return errs
}

// runExample dispatches an example to the syntactic or semantic execution path.
func runExample(
	engine validation.ValidationEngine,
	files map[string]string,
	semantic bool,
	newProvider func() (provider.Provider, error),
) ([]docs.ProducedDiagnostic, error) {
	if !semantic {
		return runSyntactic(engine, files)
	}
	if newProvider == nil {
		return nil, fmt.Errorf("semantic example requires a provider factory but none was supplied")
	}
	return runSemantic(engine, newProvider, files)
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

// runSemantic mirrors project.handleValidation's semantic half: parse each
// example file, load every parsed spec into a fresh isolated provider (legacy
// vs v1 branch, exactly like project.loadSpec), build the resource graph,
// detect cycles, then run ValidateSemantic over the parsed specs and convert
// the diagnostics to ProducedDiagnostic.
//
// A fresh provider per call is essential: providers accumulate loaded specs as
// state, so reusing one across examples would leak resources between them and
// corrupt the graph. The factory yields that isolation.
//
// Syntax is validated first. A semantic example whose files don't even parse or
// pass syntax isn't a faithful semantic fixture — it's a broken one — so those
// cases return an error (surfaced as a verification failure) rather than being
// silently reported as produced diagnostics. Only a syntactically clean
// mini-project proceeds to graph building and semantic validation.
func runSemantic(
	engine validation.ValidationEngine,
	newProvider func() (provider.Provider, error),
	files map[string]string,
) ([]docs.ProducedDiagnostic, error) {
	p, err := newProvider()
	if err != nil {
		return nil, fmt.Errorf("building provider: %w", err)
	}

	parsed := map[string]*specs.RawSpec{}
	for name, content := range files {
		rs := &specs.RawSpec{Data: []byte(content)}
		if _, err := rs.Parse(); err != nil {
			return nil, fmt.Errorf("parsing fixture file %s: %w", name, err)
		}
		parsed[name] = rs
	}

	ctx := context.Background()

	// Mirror project.handleValidation ordering exactly: validate syntax over the
	// parsed specs *before* loading them into the provider. LoadLegacySpec/LoadSpec
	// mutate the spec while ingesting it (e.g. rewriting legacy refs), so loading
	// first would corrupt what ValidateSyntax inspects.
	//
	// A semantic fixture must be syntactically valid; otherwise the example isn't
	// exercising the semantic rule it claims to, so flag it as a bad fixture
	// instead of leaking syntax diagnostics into the semantic match.
	syntaxDiags, err := engine.ValidateSyntax(ctx, parsed)
	if err != nil {
		return nil, fmt.Errorf("validate syntax: %w", err)
	}
	if syntaxDiags.HasErrors() {
		return nil, fmt.Errorf("semantic fixture is not syntactically valid: %s", diagSummary(syntaxDiags.Errors()))
	}

	for name, rs := range parsed {
		if err := loadSpec(p, name, rs.Parsed()); err != nil {
			return nil, fmt.Errorf("loading fixture file %s: %w", name, err)
		}
	}

	graph, err := p.ResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("building resource graph: %w", err)
	}

	if _, err := graph.DetectCycles(); err != nil {
		return nil, fmt.Errorf("cycle detected in resource graph: %w", err)
	}

	diags, err := engine.ValidateSemantic(ctx, parsed, graph)
	if err != nil {
		return nil, fmt.Errorf("validate semantic: %w", err)
	}

	produced := make([]docs.ProducedDiagnostic, 0, len(diags))
	for _, d := range diags {
		produced = append(produced, docs.ProducedDiagnostic{
			File:     d.File,
			Severity: d.Severity.String(),
			Message:  d.Message,
		})
	}

	return produced, nil
}

// diagSummary renders diagnostics into a compact one-line summary for fixture
// error messages — enough to point an author at what their fixture got wrong.
func diagSummary(diags validation.Diagnostics) string {
	parts := make([]string, 0, len(diags))
	for _, d := range diags {
		parts = append(parts, fmt.Sprintf("%s: %s", d.File, d.Message))
	}
	return strings.Join(parts, "; ")
}

// loadSpec mirrors project.loadSpec's legacy-vs-v1 branch so semantic examples
// are loaded into the provider exactly as project validation would load them.
func loadSpec(p provider.Provider, path string, spec *specs.Spec) error {
	switch {
	case spec.IsLegacyVersion():
		return p.LoadLegacySpec(path, spec)
	case spec.Version == specs.SpecVersionV1:
		return p.LoadSpec(path, spec)
	default:
		return fmt.Errorf("unsupported spec version: %s", spec.Version)
	}
}
