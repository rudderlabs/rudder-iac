package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/location"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/registry"
)

// Engine is the core validation engine that orchestrates the validation flow
type Engine struct {
	registry      *registry.RuleRegistry
	loader        project.Loader
	provider      provider.SpecLoader
	resourceGraph *resources.Graph
	validSpecs    map[string]*specs.Spec
}

// NewEngine creates a new validation engine
func NewEngine(
	folderPath string,
	registry *registry.RuleRegistry,
	provider provider.SpecLoader,
	opts ...EngineOption,
) (*Engine, error) {

	e := &Engine{
		registry: registry,
		loader:   &loader.Loader{},
		provider: provider,
	}

	for _, opt := range opts {
		opt(e)
	}

	// Load all the specs in the folder recursively
	// when spinning up the engine, so it can be used for file
	// based validation later
	loadedSpecs, err := e.loader.Load(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load specs: %w", err)
	}
	validSpecs := make(map[string]*specs.Spec)

	for path, spec := range loadedSpecs {
		err := e.provider.LoadSpec(path, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to load spec into provider: %w", err)
		}
		validSpecs[path] = spec
	}
	e.validSpecs = validSpecs

	graph, err := e.provider.ResourceGraph()
	if err != nil {
		return nil, fmt.Errorf("failed to build resource graph: %w", err)
	}
	e.resourceGraph = graph

	return e, nil
}

type EngineOption func(*Engine)

func WithLoader(l project.Loader) EngineOption {
	return func(e *Engine) {
		e.loader = l
	}
}

func (e *Engine) ValidateFile(paths ...string) []Diagnostic {
	var (
		diagnostics     []Diagnostic
		specsToValidate = make(map[string]*specs.Spec)
	)

	for _, filepath := range paths {

		dat, err := os.ReadFile(filepath)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				File:     filepath,
				Severity: validation.SeverityError,
				Message:  fmt.Sprintf("failed to read file: %v", err),
				Position: location.Position{Line: 1, Column: 1},
			})
			continue
		}

		spec, err := specs.New(dat)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				File:     filepath,
				Severity: validation.SeverityError,
				Message:  fmt.Sprintf("failed to parse spec: %v", err),
				Position: location.Position{Line: 1, Column: 1},
			})
			continue
		}
		specsToValidate[filepath] = spec
	}

	diagnostics = append(diagnostics, e.validateWithRules(specsToValidate)...)
	return diagnostics
}

// validateWithRules validates the given specs with the rules from the registry
// and returns the diagnostics
func (e *Engine) validateWithRules(specs map[string]*specs.Spec) []Diagnostic {
	var diagnostics []Diagnostic

	for path, spec := range specs {
		data, err := os.ReadFile(path)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				File:     path,
				Severity: validation.SeverityError,
				Message:  fmt.Sprintf("failed to read file for position tracking: %v", err),
				Position: location.Position{Line: 1, Column: 1},
			})
			continue
		}

		pathIndex, err := location.YAMLDataIndex(data)
		if err != nil {
			diagnostics = append(diagnostics, Diagnostic{
				File:     path,
				Severity: validation.SeverityError,
				Message:  fmt.Sprintf("failed to build position index: %v", err),
				Position: location.Position{Line: 1, Column: 1},
			})
			continue
		}

		commonMetadata, _ := spec.CommonMetadata()

		vCtx := &validation.ValidationContext{
			Metadata: &validation.Metadata{
				Name: commonMetadata.Name,
			},
			Spec:      spec.Spec,
			Path:      path,
			Filename:  filepath.Base(path),
			Kind:      spec.Kind,
			Version:   spec.Version,
			PathIndex: pathIndex,
		}

		// Get applicable rules for this kind
		// and apply them to the validation context sequentially
		rules := e.registry.RulesForKind(spec.Kind)
		// log.Printf("[engine] Validating with rules: %v", rules)

		for _, rule := range rules {
			errors := rule.Validate(vCtx, e.resourceGraph)
			for _, vErr := range errors {
				// Use line content from position if available, fallback to fragment
				fragment := vErr.Pos.Content
				if fragment == "" {
					fragment = vErr.Fragment
				}
				diagnostics = append(diagnostics, Diagnostic{
					File:     path,
					Rule:     rule.ID(),
					Severity: rule.Severity(),
					Message:  vErr.Msg,
					Position: vErr.Pos,
					Fragment: fragment,
				})
			}
		}
	}

	return diagnostics
}

// Validate executes the validation flow for the project at the given path
func (e *Engine) Validate() []Diagnostic {
	return e.validateWithRules(e.validSpecs)
}

// ResourceGraph returns the resource graph for completion and other features
func (e *Engine) ResourceGraph() *resources.Graph {
	return e.resourceGraph
}
