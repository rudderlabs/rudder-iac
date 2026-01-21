package validation

import (
	"context"
	"fmt"
	"os"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ValidationEngine is the main orchestrator for the two-phase validation framework.
// It coordinates syntactic validation (pre-graph) and semantic validation (post-graph),
// manages rule execution, and aggregates diagnostics across all validated files.
type ValidationEngine interface {
	// Validate runs two-phase validation on the project.
	// It runs syntactic validation on all specs, builds the resource graph,
	// then runs semantic validation on them.
	Validate(ctx context.Context, specs map[string]*specs.Spec) ([]Diagnostic, error)
}

type Provider interface {
	provider.SpecLoader
	provider.RuleProvider
}

type validationEngine struct {
	registry rules.Registry
	provider Provider
	log      *logger.Logger
}

// NewValidationEngine creates a new validation engine instance.
// It collects and registers rules from the provider.
func NewValidationEngine(
	registry rules.Registry,
	p Provider,
	log *logger.Logger,
) (ValidationEngine, error) {
	return &validationEngine{
		registry: registry,
		provider: p,
		log:      log,
	}, nil
}

func (e *validationEngine) Validate(ctx context.Context, rawSpecs map[string]*specs.Spec) ([]Diagnostic, error) {
	collector := newDiagnosticCollector()

	for fpath, spec := range rawSpecs {
		// No resource graph exists yet for the syntactic validation rules
		diagnostics, err := e.runValidationRules(
			fpath,
			e.registry.SyntacticRulesForKind(spec.Kind),
			spec,
			nil,
		)

		if err != nil {
			return nil, fmt.Errorf("syntactic validation for %s: %w", fpath, err)
		}

		collector.add(diagnostics...)
	}

	if collector.hasErrors() {
		e.log.Debug("syntactic errors found", "count", len(collector.getErrors()))
		collector.sortDiagnostics()
		return collector.getAll(), nil
	}

	e.log.Debug("building resource graph from all specs")
	graph, err := e.buildResourceGraph(rawSpecs)
	if err != nil {
		return nil, fmt.Errorf("building resource graph: %w", err)
	}

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
	return collector.getAll(), nil
}

func (e *validationEngine) buildResourceGraph(specsMap map[string]*specs.Spec) (*resources.Graph, error) {
	for path, spec := range specsMap {
		// _, err := e.provider.ParseSpec(path, spec)
		// if err != nil {
		// 	e.log.Warn("failed to parse spec during graph building", "path", path, "error", err)
		// 	continue
		// }

		if err := e.provider.LoadSpec(path, spec); err != nil {
			return nil, fmt.Errorf("loading spec: %w", err)
		}
	}

	graph, err := e.provider.ResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("getting resource graph: %w", err)
	}

	_, err = graph.DetectCycles()
	if err != nil {
		return nil, fmt.Errorf("graph contains cycles: %w", err)
	}

	return graph, nil
}

func (e *validationEngine) runValidationRules(
	fpath string,
	toValidateAgainst []rules.Rule,
	spec *specs.Spec,
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

	diagnostics := make([]Diagnostic, 0)
	for _, rule := range toValidateAgainst {
		results := rule.Validate(&rules.ValidationContext{
			Spec:     spec.Spec,
			Kind:     spec.Kind,
			Version:  spec.Version,
			Metadata: spec.Metadata,
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
