package validation

import (
	"context"
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
	ValidateSyntax(ctx context.Context, input []*InputSpec) (Diagnostics, error)

	// ValidateSemantic runs semantic validation after resource graph is built.
	// Rules receive ValidationContext with populated Graph for cross-resource validation.
	// Returns diagnostics for semantic errors (invalid references, dependency issues, etc.)
	ValidateSemantic(ctx context.Context, input []*InputSpec, graph *resources.Graph) (Diagnostics, error)
}

type InputSpec struct {
	Path   string
	Raw    []byte
	Parsed *specs.Spec
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
func (e *validationEngine) ValidateSyntax(ctx context.Context, input []*InputSpec) (Diagnostics, error) {
	collector := newDiagnosticCollector()

	for _, spec := range input {
		diagnostics, err := e.runValidationRules(
			spec.Path,
			e.registry.SyntacticRulesForKind(spec.Parsed.Kind),
			spec,
			nil, // No graph for syntactic validation
		)
		if err != nil {
			return nil, fmt.Errorf("syntactic validation for %s: %w", spec.Path, err)
		}
		collector.add(diagnostics...)
	}

	collector.sortDiagnostics()
	return Diagnostics(collector.getAll()), nil
}

// ValidateSemantic runs semantic validation after resource graph is built.
// Rules receive ValidationContext with populated Graph for cross-resource validation.
func (e *validationEngine) ValidateSemantic(ctx context.Context, input []*InputSpec, graph *resources.Graph) (Diagnostics, error) {
	collector := newDiagnosticCollector()

	for _, spec := range input {
		diagnostics, err := e.runValidationRules(
			spec.Path,
			e.registry.SemanticRulesForKind(spec.Parsed.Kind),
			spec,
			graph,
		)
		if err != nil {
			return nil, fmt.Errorf("semantic validation for %s: %w", spec.Path, err)
		}
		collector.add(diagnostics...)
	}

	collector.sortDiagnostics()
	return Diagnostics(collector.getAll()), nil
}

func (e *validationEngine) runValidationRules(
	fpath string,
	toValidateAgainst []rules.Rule,
	inputSpec *InputSpec,
	graph *resources.Graph,
) ([]Diagnostic, error) {

	// The path indexer only works on the raw spec data
	// from the input spec, it is not necessary the path resolves to filesystem path.
	// It could also be a path over the network
	pi, err := pathindex.NewPathIndexer(inputSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf("building path indexer: %w", err)
	}

	// unmarshal -> map -> struct
	// syntactic validation ( spec syntax )
	diagnostics := make([]Diagnostic, 0)
	for _, rule := range toValidateAgainst {
		results := rule.Validate(&rules.ValidationContext{
			Spec:     inputSpec.Parsed.Spec,
			Kind:     inputSpec.Parsed.Kind,
			Version:  inputSpec.Parsed.Version,
			Metadata: inputSpec.Parsed.Metadata,
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
