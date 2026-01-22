package validation

import (
	"context"
	"fmt"
	"os"

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
	ValidateSyntax(ctx context.Context, specs map[string]*specs.Spec) (Diagnostics, error)

	// ValidateSemantic runs semantic validation after resource graph is built.
	// Rules receive ValidationContext with populated Graph for cross-resource validation.
	// Returns diagnostics for semantic errors (invalid references, dependency issues, etc.)
	ValidateSemantic(ctx context.Context, specs map[string]*specs.Spec, graph *resources.Graph) (Diagnostics, error)
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
// Rules receive ValidationContext with Graph = nil.
func (e *validationEngine) ValidateSyntax(ctx context.Context, rawSpecs map[string]*specs.Spec) (Diagnostics, error) {
	collector := newDiagnosticCollector()

	for fpath, spec := range rawSpecs {
		diagnostics, err := e.runValidationRules(
			fpath,
			e.registry.SyntacticRulesForKind(spec.Kind),
			spec,
			nil, // No graph for syntactic validation
		)
		if err != nil {
			return nil, fmt.Errorf("syntactic validation for %s: %w", fpath, err)
		}
		collector.add(diagnostics...)
	}

	collector.sortDiagnostics()
	return Diagnostics(collector.getAll()), nil
}

// ValidateSemantic runs semantic validation after resource graph is built.
// Rules receive ValidationContext with populated Graph for cross-resource validation.
func (e *validationEngine) ValidateSemantic(ctx context.Context, rawSpecs map[string]*specs.Spec, graph *resources.Graph) (Diagnostics, error) {
	collector := newDiagnosticCollector()

	for fpath, spec := range rawSpecs {
		diagnostics, err := e.runValidationRules(
			fpath,
			e.registry.SemanticRulesForKind(spec.Kind),
			spec,
			graph,
		)
		if err != nil {
			return nil, fmt.Errorf("semantic validation for %s: %w", fpath, err)
		}
		collector.add(diagnostics...)
	}

	collector.sortDiagnostics()
	return Diagnostics(collector.getAll()), nil
}

func (e *validationEngine) runValidationRules(
	fpath string,
	toValidateAgainst []rules.Rule,
	rawSpec *specs.Spec, // from loader => rawSpec.Spec is a map
	graph *resources.Graph,
) ([]Diagnostic, error) {

	content, err := os.ReadFile(fpath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	pi, err := pathindex.NewPathIndexer(content)
	if err != nil {
		return nil, fmt.Errorf("building path indexer: %w", err)
	}

	// unmarshal -> map -> struct
	// syntactic validation ( spec syntax )
	diagnostics := make([]Diagnostic, 0)
	for _, rule := range toValidateAgainst {
		results := rule.Validate(&rules.ValidationContext{
			Spec:     rawSpec.Spec,
			Kind:     rawSpec.Kind,
			Version:  rawSpec.Version,
			Metadata: rawSpec.Metadata,
			Graph:    graph,
		})

		for _, result := range results {
			position, err := pi.PositionLookup(result.Reference)
			if err != nil {
				return nil, fmt.Errorf("getting position for reference %s: %w", result.Reference, err)
			}
			diagnostics = append(diagnostics, Diagnostic{
				RuleID:   rule.ID(),
				Severity: rule.Severity(),
				Message:  result.Message,
				File:     fpath,
				Position: *position,
				Examples: rule.Examples(),
			})
		}
	}

	return diagnostics, nil
}
