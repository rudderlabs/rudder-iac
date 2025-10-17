package project

import (
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

// SpecLoader defines the interface for loading project specifications.
type Loader interface {
	// Load loads specifications from the specified location.
	Load(location string) (map[string]*specs.Spec, error)
}

type ProjectProvider interface {
	GetName() string
	GetSupportedKinds() []string
	GetSupportedTypes() []string
	// Validate makes provider specific validations on the resource graph.
	// Providers are expected to validate their own resources only, but can leverage
	// the full graph for cross-resource validations.
	Validate(graph *resources.Graph) error
	LoadSpec(path string, s *specs.Spec) error
	GetResourceGraph() (*resources.Graph, error)
}

type Provider interface {
	ProjectProvider
	syncer.SyncProvider
	importremote.WorkspaceImporter
}

type Project interface {
	Load() error
	GetResourceGraph() (*resources.Graph, error)
}

type project struct {
	Location string
	Provider Provider
	loader   Loader // New field
	// specs      []*specs.Spec // This seems to be handled by the provider internally via LoadSpec
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
func New(location string, provider Provider, opts ...ProjectOption) Project {
	p := &project{
		Location: location,
		Provider: provider,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.loader == nil {
		p.loader = &loader.Loader{}
	}

	return p
}

// Load loads the project specifications using the configured SpecLoader
// and then validates them with the provider.
func (p *project) Load() error {
	loadedSpecs, err := p.loader.Load(p.Location) // Use the specLoader
	if err != nil {
		return fmt.Errorf("failed to load specs using specLoader: %w", err)
	}

	// The rest of the logic from old loadSpecs
	for path, spec := range loadedSpecs {
		if !slices.Contains(p.Provider.GetSupportedKinds(), spec.Kind) {
			return specs.ErrUnsupportedKind{
				Kind: spec.Kind,
			}
		}
		if err := p.Provider.LoadSpec(path, spec); err != nil {
			return fmt.Errorf("provider failed to load spec from path %s: %w", path, err)
		}
	}

	graph, err := p.Provider.GetResourceGraph()
	if err != nil {
		return fmt.Errorf("getting resource graph: %w", err)
	}

	return p.Provider.Validate(graph)
}

func (p *project) GetResourceGraph() (*resources.Graph, error) {
	return p.Provider.GetResourceGraph()
}
