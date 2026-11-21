package validation

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

// ValidationEngine is the main orchestrator for the two-phase validation framework.
// It coordinates syntactic validation (pre-graph) and semantic validation (post-graph),
// manages rule execution, and aggregates diagnostics across all validated files.
type ValidationEngine interface {
	// ValidateSyntax runs syntactic validation on raw specs before resource graph is built.
	// Rules receive ValidationContext with Graph = nil.
	// Returns diagnostics for syntax errors (missing fields, invalid formats, etc.)
	ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.RawSpec) (Diagnostics, error)

	// ValidateSemantic runs semantic validation after resource graph is built.
	// Rules receive ValidationContext with populated Graph for cross-resource validation.
	// workspaceID is the run's target workspace ("" when none), surfaced to
	// workspace-scoped rules via ValidationContext.WorkspaceID.
	// Returns diagnostics for semantic errors (invalid references, dependency issues, etc.)
	ValidateSemantic(ctx context.Context, rawSpecs map[string]*specs.RawSpec, graph *resources.Graph, workspaceID string) (Diagnostics, error)

	// ValidateSources runs the rules that read specs as written (see
	// rules.SourceRule). sources holds the specs before variable substitution,
	// and positions point into those bytes.
	ValidateSources(ctx context.Context, sources map[string]*specs.RawSpec) (Diagnostics, error)
}

type validationEngine struct {
	registry rules.Registry
	log      *logger.Logger
}

// NewValidationEngine creates a new validation engine instance.
// Registry should be pre-populated with syntactic and semantic rules.
func NewValidationEngine(
	registry rules.Registry,
	log *logger.Logger,
) (ValidationEngine, error) {
	return &validationEngine{
		registry: registry,
		log:      log,
	}, nil
}

// ValidateSyntax runs syntactic validation on raw specs before resource graph is built.
// It operates in two phases:
//  1. Per-spec rules: validates each spec individually (MultiSpecRules return nil here)
//  2. Multiple-resource rules: if Phase 1 passes, runs each with the specs matching its patterns
func (e *validationEngine) ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.RawSpec) (Diagnostics, error) {
	toReturn := make(Diagnostics, 0)

	// Phase 1: per-spec validation (MultiSpecRules harmlessly return nil from Validate)
	for path, spec := range rawSpecs {
		diagnostics, err := e.runValidationRules(
			path,
			e.registry.SyntacticRulesFor(spec.Parsed().Kind, spec.Parsed().Version),
			spec,
			nil,
			"", // syntactic validation is workspace-agnostic
		)
		if err != nil {
			return nil, fmt.Errorf("syntactic validation for %s: %w", path, err)
		}

		toReturn = append(toReturn, diagnostics...)
	}

	if toReturn.HasErrors() {
		toReturn.Sort()
		return toReturn, nil
	}

	// Phase 2: project-wide validation
	projectDiags, err := e.runProjectValidationRules(rawSpecs)
	if err != nil {
		return nil, fmt.Errorf("project-wide validation: %w", err)
	}
	toReturn = append(toReturn, projectDiags...)

	toReturn.Sort()
	return toReturn, nil
}

// runProjectValidationRules discovers rules implementing MultiSpecRule from all
// registered syntactic rules and executes each with the specs matching its AppliesTo()
// patterns. Unlike per-spec rules, these need more than one spec at once (e.g. cross-file
// uniqueness), but they still only see the specs they apply to — not the whole project.
func (e *validationEngine) runProjectValidationRules(rawSpecs map[string]*specs.RawSpec) (Diagnostics, error) {
	// MultiSpecRules may have any AppliesTo() pattern (not just MatchAll), so we
	// scan all registered syntactic rules and filter by the MultiSpecRule interface.
	var multiSpecRules []rules.MultiSpecRule
	for _, rule := range e.registry.AllSyntacticRules() {
		if mr, ok := rule.(rules.MultiSpecRule); ok {
			multiSpecRules = append(multiSpecRules, mr)
		}
	}

	if len(multiSpecRules) == 0 {
		return nil, nil
	}

	contexts := make(map[string]*rules.ValidationContext, len(rawSpecs))
	for path, rawSpec := range rawSpecs {
		contexts[path] = &rules.ValidationContext{
			FilePath: path,
			FileName: filepath.Base(path),
			Spec:     rawSpec.Parsed().Spec,
			Kind:     rawSpec.Parsed().Kind,
			Version:  rawSpec.Parsed().Version,
			Metadata: rawSpec.Parsed().Metadata,
		}
	}

	diagnostics := make(Diagnostics, 0)

	for _, mr := range multiSpecRules {
		rule := mr.(rules.Rule)
		resultsMap := mr.ValidateSpecs(filterContextsByPatterns(contexts, rule.AppliesTo()))

		for filePath, results := range resultsMap {
			rawSpec, ok := rawSpecs[filePath]
			if !ok {
				return nil, fmt.Errorf("project rule %s returned results for unknown file: %s", rule.ID(), filePath)
			}

			pi, err := rawSpec.PathIndexer()
			if err != nil {
				return nil, fmt.Errorf("building path indexer for %s: %w", filePath, err)
			}

			fileDiagnostics, err := ruleDiagnostics(rule, filePath, pi, results)
			if err != nil {
				return nil, err
			}
			diagnostics = append(diagnostics, fileDiagnostics...)
		}
	}

	return diagnostics, nil
}

// filterContextsByPatterns returns the subset of contexts whose (kind, version)
// matches at least one of the given patterns. A MultiSpecRule receives only
// the specs it applies to, so resource-only rules never see manifest specs (and vice
// versa) even though they all run in the same project-wide validation pass.
func filterContextsByPatterns(
	contexts map[string]*rules.ValidationContext,
	patterns []rules.MatchPattern,
) map[string]*rules.ValidationContext {
	filtered := make(map[string]*rules.ValidationContext)
	for path, ctx := range contexts {
		for _, p := range patterns {
			if p.Matches(ctx.Kind, ctx.Version) {
				filtered[path] = ctx
				break
			}
		}
	}
	return filtered
}

// ValidateSemantic runs semantic validation after resource graph is built.
// Rules receive ValidationContext with populated Graph for cross-resource validation.
func (e *validationEngine) ValidateSemantic(ctx context.Context, rawSpecs map[string]*specs.RawSpec, graph *resources.Graph, workspaceID string) (Diagnostics, error) {
	toReturn := make(Diagnostics, 0)

	for path, spec := range rawSpecs {
		diagnostics, err := e.runValidationRules(
			path,
			e.registry.SemanticRulesFor(spec.Parsed().Kind, spec.Parsed().Version),
			spec,
			graph,
			workspaceID,
		)
		if err != nil {
			return nil, fmt.Errorf("semantic validation for %s: %w", path, err)
		}
		toReturn = append(toReturn, diagnostics...)
	}

	toReturn.Sort()
	return toReturn, nil
}

func (e *validationEngine) runValidationRules(
	path string,
	toValidateAgainst []rules.Rule,
	rawSpec *specs.RawSpec,
	graph *resources.Graph,
	workspaceID string,
) ([]Diagnostic, error) {

	pi, err := rawSpec.PathIndexer()
	if err != nil {
		return nil, fmt.Errorf("building path indexer: %w", err)
	}

	// unmarshal -> map -> struct
	// syntactic validation ( spec syntax )
	diagnostics := make([]Diagnostic, 0)
	for _, rule := range toValidateAgainst {
		results := rule.Validate(&rules.ValidationContext{
			FilePath:    path,
			FileName:    filepath.Base(path),
			Spec:        rawSpec.Parsed().Spec,
			Kind:        rawSpec.Parsed().Kind,
			Version:     rawSpec.Parsed().Version,
			Metadata:    rawSpec.Parsed().Metadata,
			Graph:       graph,
			WorkspaceID: workspaceID,
		})

		ruleDiags, err := ruleDiagnostics(rule, path, pi, results)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, ruleDiags...)
	}

	return diagnostics, nil
}

// ValidateSources runs every registered rule that implements SourceRule, once,
// with the sources whose kind and version match the rule. It reads kind and
// version without substituting, so it reports on a spec whose variables are
// not defined yet.
func (e *validationEngine) ValidateSources(ctx context.Context, sources map[string]*specs.RawSpec) (Diagnostics, error) {
	var sourceRules []rules.Rule
	for _, rule := range e.registry.AllSyntacticRules() {
		if _, ok := rule.(rules.SourceRule); ok {
			sourceRules = append(sourceRules, rule)
		}
	}

	if len(sourceRules) == 0 {
		return nil, nil
	}

	contexts := make(map[string]*rules.ValidationContext, len(sources))
	for path, source := range sources {
		contexts[path] = rules.NewSourceContext(path, source)
	}

	diagnostics := make(Diagnostics, 0)
	for _, rule := range sourceRules {
		resultsMap := rule.(rules.SourceRule).ValidateSources(filterContextsByPatterns(contexts, rule.AppliesTo()))

		for filePath, results := range resultsMap {
			source, ok := sources[filePath]
			if !ok {
				return nil, fmt.Errorf("source rule %s returned results for unknown file: %s", rule.ID(), filePath)
			}

			fileDiagnostics, err := sourceDiagnostics(rule, filePath, source, contexts[filePath].References, results)
			if err != nil {
				return nil, err
			}
			diagnostics = append(diagnostics, fileDiagnostics...)
		}
	}

	diagnostics.Sort()
	return diagnostics, nil
}

// sourceDiagnostics places a source rule's results in the source as written.
// A source that is not YAML as written, such as one with an unquoted reference
// followed by more text, is indexed with its references swapped for
// placeholders, since that copy parses. Each position found in that copy is
// then mapped back to the source, and each line a diagnostic shows is given
// its references back, so no placeholder is printed. When even that copy does
// not parse, results are placed at the start of the file.
func sourceDiagnostics(rule rules.Rule, file string, source *specs.RawSpec, replaced varsubst.Replaced, results []rules.ValidationResult) (Diagnostics, error) {
	if pi, err := source.PathIndexer(); err == nil {
		return ruleDiagnostics(rule, file, pi, results)
	}

	pi, err := pathindex.NewPathIndexer(replaced.Data)
	if err != nil {
		pi = nil
	}
	diagnostics, err := ruleDiagnostics(rule, file, pi, results)
	if err != nil {
		return nil, err
	}
	for i := range diagnostics {
		position := &diagnostics[i].Position
		position.Line, position.Column = replaced.SourcePosition(position.Line, position.Column)
		position.LineText = replaced.Restore(position.LineText)
	}
	return diagnostics, nil
}

// ruleDiagnostics turns rule's results for file into diagnostics, each placed
// where its reference points in pi.
func ruleDiagnostics(rule rules.Rule, file string, pi pathindex.PathIndexer, results []rules.ValidationResult) (Diagnostics, error) {
	diagnostics := make(Diagnostics, 0, len(results))
	for _, result := range results {
		position, err := resultPosition(pi, result)
		if err != nil {
			return nil, err
		}
		diagnostics = append(diagnostics, Diagnostic{
			RuleID:    rule.ID(),
			Severity:  rule.Severity(),
			Message:   result.Message,
			File:      file,
			Position:  position,
			Sensitive: result.Sensitive,
			Examples:  rule.Examples(),
		})
	}
	return diagnostics, nil
}

// resultPosition resolves a result's reference to its position in the file,
// falling back to the nearest ancestor when the path does not exist, and to the
// start of the file when there is no indexer. A sensitive result has the value
// on its line masked here, before any renderer sees it.
func resultPosition(pi pathindex.PathIndexer, result rules.ValidationResult) (pathindex.Position, error) {
	if pi == nil {
		return pathindex.StartingPosition, nil
	}

	if result.Sensitive {
		return *pi.MaskedPosition(result.Reference), nil
	}

	position, err := pi.PositionLookup(result.Reference)
	if err != nil {
		if !errors.Is(err, pathindex.ErrPathNotFound) {
			return pathindex.Position{}, fmt.Errorf("getting position for reference %s: %w", result.Reference, err)
		}

		// if the error is ErrPathNotFound which is usually the case, we
		// use the nearest position to the reference.
		position = pi.NearestPosition(result.Reference)
	}
	return *position, nil
}
