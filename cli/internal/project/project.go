package project

import (
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/samber/lo"
)

// Loader defines the interface for loading project specifications.
type Loader interface {
	// Load loads specifications from the specified location.
	Load(location string) (map[string]*specs.Spec, error)
}

type ProjectProvider interface {
	provider.SpecLoader
	provider.Validator
}

type Project interface {
	Location() string
	Load() error
	ResourceGraph() (*resources.Graph, error)
	Specs() map[string]*specs.Spec
}

type project struct {
	location        string
	provider        ProjectProvider
	loader          Loader
	specs           map[string]*specs.Spec
	loadLegacySpecs bool
}

// ProjectOption defines a functional option for configuring a Project.
type ProjectOption func(*project)

// WithSpecLoader allows providing a custom SpecLoader.
func WithLoader(l Loader) ProjectOption {
	return func(p *project) {
		if l != nil {
			p.loader = l
		}
	}
}

// WithLegacySpecSupport enables loading legacy specs (rudder/0.1).
func WithLegacySpecSupport() ProjectOption {
	return func(p *project) {
		p.loadLegacySpecs = true
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
		if !p.loadLegacySpecs {
			return fmt.Errorf("spec version %s is no longer supported. Please migrate to version %s using the migrate project command", specs.SpecVersionV0_1, specs.SpecVersionV1)
		}
		return p.provider.LoadLegacySpec(path, spec)
	case spec.Version == specs.SpecVersionV1:
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

	// The rest of the logic from old loadSpecs
	for path, spec := range p.specs {
		parsed, err := p.provider.ParseSpec(path, spec)
		if err != nil {
			return fmt.Errorf("provider failed to parse spec from path %s: %w", path, err)
		}

		if err := ValidateSpec(spec, parsed); err != nil {
			return fmt.Errorf("provider failed to validate spec from path %s: %w", path, err)
		}

		if err := p.loadSpec(path, spec); err != nil {
			return fmt.Errorf("failed to load spec from path %s: %w", path, err)
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
