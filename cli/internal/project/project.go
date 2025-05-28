package project

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader" // Will be used for default loader
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// SpecLoader defines the interface for loading project specifications.
type Loader interface {
	// Load loads specifications from the specified location.
	Load(location string) (map[string]*specs.Spec, error)
}

type ProjectProvider interface {
	GetSupportedKinds() []string
	GetSupportedTypes() []string
	Validate() error
	LoadSpec(path string, s *specs.Spec) error
	GetResourceGraph() (*resources.Graph, error)
}

type SyncProvider interface {
	LoadState(ctx context.Context) (*state.State, error)
	PutResourceState(ctx context.Context, URN string, state *state.ResourceState) error
	DeleteResourceState(ctx context.Context, state *state.ResourceState) error
	Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error
}

type Provider interface {
	ProjectProvider
	SyncProvider
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
		if !p.providerSupportsKind(spec.Kind) {
			return specs.ErrUnsupportedKind{
				Kind: spec.Kind,
			}
		}
		if err := p.Provider.LoadSpec(path, spec); err != nil {
			return fmt.Errorf("provider failed to load spec from path %s: %w", path, err)
		}
	}

	return p.Provider.Validate()
}

func (p *project) providerSupportsKind(kind string) bool {
	for _, k := range p.Provider.GetSupportedKinds() {
		if k == kind {
			return true
		}
	}
	return false
}

func (p *project) GetResourceGraph() (*resources.Graph, error) {
	return p.Provider.GetResourceGraph()
}
