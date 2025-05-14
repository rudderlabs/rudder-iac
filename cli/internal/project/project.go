package project

import (
	"context"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/loader"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

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
}

func New(location string, provider Provider) Project {
	return &project{
		Location: location,
		Provider: provider,
	}
}

func (p *project) Load() error {
	if err := p.loadSpecs(p.Location); err != nil {
		return err
	}
	return p.Provider.Validate()
}

func (p *project) loadSpecs(location string) error {
	l := loader.New(location)
	s, err := l.Load()
	if err != nil {
		return err
	}

	for path, spec := range s {
		if !p.providerSupportsKind(spec.Kind) {
			return specs.ErrUnsupportedKind{
				Kind: spec.Kind,
			}
		}
		if err := p.Provider.LoadSpec(path, spec); err != nil {
			return err
		}
	}
	return nil
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
