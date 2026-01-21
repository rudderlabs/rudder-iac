package project

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/project/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/samber/lo"
)

var log = logger.New("project")

// Loader defines the interface for loading project specifications.
type Loader interface {
	// Load loads specifications from the specified location.
	Load(location string) (map[string]*specs.Spec, error)
}

type ProjectProvider interface {
	provider.SpecLoader
	provider.Validator
	provider.RuleProvider
	provider.TypeProvider
}

type Project interface {
	Location() string
	Load() error
	ResourceGraph() (*resources.Graph, error)
	Specs() map[string]*specs.Spec
}

type project struct {
	location            string
	provider            ProjectProvider
	loader              Loader
	specs               map[string]*specs.Spec
	loadV1Specs         bool
	validationEngine    validation.ValidationEngine
	renderer            renderer.Renderer
	validateUsingEngine bool
}

// ProjectOption defines a functional option for configuring a Project.
type ProjectOption func(*project)

func WithValidationEngine(e validation.ValidationEngine) ProjectOption {
	return func(p *project) {
		p.validationEngine = e
	}
}

func WithValidateUsingEngine() ProjectOption {
	return func(p *project) {
		p.validateUsingEngine = true
	}
}

// WithSpecLoader allows providing a custom SpecLoader.
func WithLoader(l Loader) ProjectOption {
	return func(p *project) {
		if l != nil {
			p.loader = l
		}
	}
}

// WithV1SpecSupport enables loading v1 specs (rudder/v1).
func WithV1SpecSupport() ProjectOption {
	return func(p *project) {
		p.loadV1Specs = true
	}
}

// New creates a new Project instance.
// By default, it uses a loader.Loader.
func New(location string, provider provider.Provider, opts ...ProjectOption) Project {
	p := &project{
		location: location,
		provider: provider,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.loader == nil {
		p.loader = &loader.Loader{}
	}

	if p.renderer == nil {
		// fallback to default text renderer if
		// no renderer is provided
		p.renderer = renderer.NewTextRenderer(
			os.Stdout,
			os.Stderr)
	}

	return p
}

// GetLocation returns the root directory configured for this project, containing all specs.
func (p *project) Location() string {
	return p.location
}

func (p *project) Specs() map[string]*specs.Spec {
	return p.specs
}

func (p *project) loadSpec(path string, spec *specs.Spec) error {
	switch {
	case spec.IsLegacyVersion():
		return p.provider.LoadLegacySpec(path, spec)
	case spec.Version == specs.SpecVersionV1 && p.loadV1Specs:
		return p.provider.LoadSpec(path, spec)
	default:
		return fmt.Errorf("unsupported spec version: %s", spec.Version)
	}
}

// Load loads the project specifications using the configured SpecLoader
// and then validates them with the provider.
func (p *project) Load() error {
	var err error

	p.specs, err = p.loader.Load(p.location) // Use the specLoader
	if err != nil {
		return fmt.Errorf("failed to load specs using specLoader: %w", err)
	}

	if p.validateUsingEngine {
		return p.handleValidation()
	}

	// TODO: once the validation engine is stable, remove this
	// legacy validation handler.
	return p.handleLegacyValidation()
}

func (p *project) handleLegacyValidation() error {
	// loop over the raw specs and hydrate the provider's state
	// by parsing the spec and then loading it into the provider.
	for path, spec := range p.specs {
		parsed, err := p.provider.ParseSpec(path, spec)
		if err != nil {
			return fmt.Errorf("provider failed to parse spec from path %s: %w", path, err)
		}

		if err := ValidateSpec(spec, parsed); err != nil {
			return fmt.Errorf("provider failed to validate spec from path %s: %w", path, err)
		}

		if err := p.loadSpec(path, spec); err != nil {
			return fmt.Errorf("provider failed to load spec from path %s: %w", path, err)
		}
	}

	graph, err := p.provider.ResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	// Detect circular dependencies
	_, err = graph.DetectCycles()
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return p.provider.Validate(graph)
}

// handleValidation validates the raw specs using the validation engine along
// with rendering the diagnostics.
func (p *project) handleValidation() error {
	// setup the registry with the project-level rules along
	// with the rules from the provider.
	registry, err := p.registry()
	if err != nil {
		return fmt.Errorf("setting up registry: %w", err)
	}

	engine, err := validation.NewValidationEngine(registry, p.provider, log)
	if err != nil {
		return fmt.Errorf("initialising validation engine: %w", err)
	}

	diagnostics, err := engine.Validate(context.Background(), p.specs)
	if err != nil {
		return fmt.Errorf("validating specs: %w", err)
	}

	if err := p.renderer.Render(diagnostics); err != nil {
		return fmt.Errorf("rendering diagnostics: %w", err)
	}

	if len(diagnostics) > 0 {
		return fmt.Errorf("validation failed")
	}

	return nil
}

func (p *project) registry() (rules.Registry, error) {
	baseRegistry := rules.NewRegistry()

	// Add project-level syntactic rules
	// to the syntactic rules from the provider
	syntactic := p.provider.SyntacticRules()
	syntactic = append(
		syntactic,
		prules.NewKindValidRule(p.provider.SupportedKinds()),
		prules.NewMetadataNameValidRule(),
		// prules.NewMetadataSyntaxValidRule(),
		prules.NewSpecSyntaxValidRule(),
		prules.NewVersionValidRule(),
	)
	for _, rule := range syntactic {
		if err := baseRegistry.RegisterSyntactic(rule); err != nil {
			return nil, fmt.Errorf("registering syntactic rule %s: %w", rule.ID(), err)
		}
	}

	semantic := p.provider.SemanticRules()
	for _, rule := range semantic {
		if err := baseRegistry.RegisterSemantic(rule); err != nil {
			return nil, fmt.Errorf("registering semantic rule %s: %w", rule.ID(), err)
		}
	}

	return baseRegistry, nil
}

func ValidateSpec(spec *specs.Spec, parsed *specs.ParsedSpec) error {
	var metadataIds []string
	metadata, err := spec.CommonMetadata()
	if err != nil {
		return err
	}

	err = metadata.Validate()
	if err != nil {
		return fmt.Errorf("invalid spec metadata: %w", err)
	}

	if metadata.Import != nil {
		for _, workspace := range metadata.Import.Workspaces {
			for _, resource := range workspace.Resources {
				metadataIds = append(metadataIds, resource.LocalID)
			}
		}

		_, missingInSpec := lo.Difference(parsed.ExternalIDs, metadataIds)
		if len(missingInSpec) > 0 {
			return fmt.Errorf("local_id from import metadata missing in spec: %s", strings.Join(missingInSpec, ", "))
		}
	}

	return nil
}

func (p *project) ResourceGraph() (*resources.Graph, error) {
	return p.provider.ResourceGraph()
}
