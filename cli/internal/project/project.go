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
	GetResourceGraph() (*resources.Graph, error)
}

type project struct {
	location string
	provider ProjectProvider
	loader   Loader
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

// Load loads the project specifications using the configured SpecLoader
// and then validates them with the provider.
func (p *project) Load() error {
	loadedSpecs, err := p.loader.Load(p.location) // Use the specLoader
	if err != nil {
		return fmt.Errorf("failed to load specs using specLoader: %w", err)
	}

	// The rest of the logic from old loadSpecs
	for path, spec := range loadedSpecs {
		parsed, err := p.provider.ParseSpec(path, spec)
		if err != nil {
			return fmt.Errorf("provider failed to parse spec from path %s: %w", path, err)
		}

		if err := ValidateSpec(spec, parsed); err != nil {
			return fmt.Errorf("provider failed to validate spec from path %s: %w", path, err)
		}

		if err := p.provider.LoadSpec(path, spec); err != nil {
			return fmt.Errorf("provider failed to load spec from path %s: %w", path, err)
		}
	}

	graph, err := p.provider.GetResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
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

func (p *project) GetResourceGraph() (*resources.Graph, error) {
	return p.provider.GetResourceGraph()
}
