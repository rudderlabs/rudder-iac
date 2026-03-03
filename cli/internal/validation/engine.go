package validation

import (
	"context"
	"errors"
	"fmt"

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
//  1. Per-spec rules: validates each spec individually (ProjectRules return nil here)
//  2. Project rules: if Phase 1 passes, runs project-wide rules with all specs at once
func (e *validationEngine) ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.RawSpec) (Diagnostics, error) {
	toReturn := make(Diagnostics, 0)

	// Phase 1: per-spec validation (ProjectRules harmlessly return nil from Validate)
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

// runProjectValidationRules discovers rules implementing ProjectRule from the wildcard bucket
// and executes them with all specs at once.
func (e *validationEngine) runProjectValidationRules(rawSpecs map[string]*specs.RawSpec) (Diagnostics, error) {
	// ProjectValidationRules are registered with AppliesTo: ["*"],
	// so they live in the wildcard bucket.
	var projectRules []rules.ProjectRule
	for _, rule := range e.registry.SyntacticRulesFor("", "") {
		if pr, ok := rule.(rules.ProjectRule); ok {
			projectRules = append(projectRules, pr)
		}
	}

	if len(projectRules) == 0 {
		return nil, nil
	}

	contexts := make(map[string]*rules.ValidationContext, len(rawSpecs))
	for path, rawSpec := range rawSpecs {
		contexts[path] = &rules.ValidationContext{
			FilePath: path,
			Spec:     rawSpec.Parsed().Spec,
			Kind:     rawSpec.Parsed().Kind,
			Version:  rawSpec.Parsed().Version,
			Metadata: rawSpec.Parsed().Metadata,
		}
	}

	diagnostics := make(Diagnostics, 0)

	for _, pr := range projectRules {
		rule := pr.(rules.Rule)
		resultsMap := pr.ValidateProject(contexts)

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
