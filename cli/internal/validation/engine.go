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
	// Returns diagnostics for semantic errors (invalid references, dependency issues, etc.)
	ValidateSemantic(ctx context.Context, rawSpecs map[string]*specs.RawSpec, graph *resources.Graph) (Diagnostics, error)
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
//  1. Per-spec rules: validates each spec individually (MultipleResourceRules return nil here)
//  2. Multiple-resource rules: if Phase 1 passes, runs each with the specs matching its patterns
func (e *validationEngine) ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.RawSpec) (Diagnostics, error) {
	toReturn := make(Diagnostics, 0)

	// Phase 1: per-spec validation (MultipleResourceRules harmlessly return nil from Validate)
	for path, spec := range rawSpecs {
		diagnostics, err := e.runValidationRules(
			path,
			e.registry.SyntacticRulesFor(spec.Parsed().Kind, spec.Parsed().Version),
			spec,
			nil,
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

// runProjectValidationRules discovers rules implementing MultipleResourceRule from all
// registered syntactic rules and executes each with the specs matching its AppliesTo()
// patterns. Unlike per-spec rules, these need more than one spec at once (e.g. cross-file
// uniqueness), but they still only see the specs they apply to — not the whole project.
func (e *validationEngine) runProjectValidationRules(rawSpecs map[string]*specs.RawSpec) (Diagnostics, error) {
	// MultipleResourceRules may have any AppliesTo() pattern (not just MatchAll), so we
	// scan all registered syntactic rules and filter by the MultipleResourceRule interface.
	var multiResourceRules []rules.MultipleResourceRule
	for _, rule := range e.registry.AllSyntacticRules() {
		if mr, ok := rule.(rules.MultipleResourceRule); ok {
			multiResourceRules = append(multiResourceRules, mr)
		}
	}

	if len(multiResourceRules) == 0 {
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

	for _, mr := range multiResourceRules {
		rule := mr.(rules.Rule)
		resultsMap := mr.ValidateProject(filterContextsByPatterns(contexts, rule.AppliesTo()))

		for filePath, results := range resultsMap {
			rawSpec, ok := rawSpecs[filePath]
			if !ok {
				return nil, fmt.Errorf("project rule %s returned results for unknown file: %s", rule.ID(), filePath)
			}

			pi, err := rawSpec.PathIndexer()
			if err != nil {
				return nil, fmt.Errorf("building path indexer for %s: %w", filePath, err)
			}

			for _, result := range results {
				position, err := pi.PositionLookup(result.Reference)
				if err != nil {
					if !errors.Is(err, pathindex.ErrPathNotFound) {
						return nil, fmt.Errorf("getting position for reference %s: %w", result.Reference, err)
					}
					position = pi.NearestPosition(result.Reference)
				}

				diagnostics = append(diagnostics, Diagnostic{
					RuleID:   rule.ID(),
					Severity: rule.Severity(),
					Message:  result.Message,
					File:     filePath,
					Position: *position,
					Examples: rule.Examples(),
				})
			}
		}
	}

	return diagnostics, nil
}

// filterContextsByPatterns returns the subset of contexts whose (kind, version)
// matches at least one of the given patterns. A MultipleResourceRule receives only
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
func (e *validationEngine) ValidateSemantic(ctx context.Context, rawSpecs map[string]*specs.RawSpec, graph *resources.Graph) (Diagnostics, error) {
	toReturn := make(Diagnostics, 0)

	for path, spec := range rawSpecs {
		diagnostics, err := e.runValidationRules(
			path,
			e.registry.SemanticRulesFor(spec.Parsed().Kind, spec.Parsed().Version),
			spec,
			graph,
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
			FilePath: path,
			FileName: filepath.Base(path),
			Spec:     rawSpec.Parsed().Spec,
			Kind:     rawSpec.Parsed().Kind,
			Version:  rawSpec.Parsed().Version,
			Metadata: rawSpec.Parsed().Metadata,
			Graph:    graph,
		})

		for _, result := range results {
			position, err := pi.PositionLookup(result.Reference)
			if err != nil {
				if !errors.Is(
					err,
					pathindex.ErrPathNotFound,
				) {
					return nil, fmt.Errorf("getting position for reference %s: %w", result.Reference, err)
				}

				// if the error is ErrPathNotFound which is usually the case, we
				// use the nearest position to the reference.
				position = pi.NearestPosition(result.Reference)
			}
			diagnostics = append(diagnostics, Diagnostic{
				RuleID:   rule.ID(),
				Severity: rule.Severity(),
				Message:  result.Message,
				File:     path,
				Position: *position,
				Examples: rule.Examples(),
			})
		}
	}

	return diagnostics, nil
}
