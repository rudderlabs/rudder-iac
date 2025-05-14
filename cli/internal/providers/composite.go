package providers

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"golang.org/x/exp/maps"
)

type CompositeProvider struct {
	Providers []project.Provider
}

func NewCompositeProvider(providers ...project.Provider) *CompositeProvider {
	return &CompositeProvider{
		Providers: providers,
	}
}

func (p *CompositeProvider) GetSupportedKinds() []string {
	kinds := make(map[string]bool)
	for _, provider := range p.Providers {
		for _, kind := range provider.GetSupportedKinds() {
			kinds[kind] = true
		}
	}
	return maps.Keys(kinds)
}

func (p *CompositeProvider) GetSupportedTypes() []string {
	types := make(map[string]bool)
	for _, provider := range p.Providers {
		for _, t := range provider.GetSupportedTypes() {
			types[t] = true
		}
	}
	return maps.Keys(types)
}

func (p *CompositeProvider) Validate() error {
	for _, provider := range p.Providers {
		if err := provider.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (p *CompositeProvider) LoadSpec(path string, s *specs.Spec) error {
	provider := p.providerForKind(s.Kind)
	if provider == nil {
		return fmt.Errorf("no provider found for kind %s", s.Kind)
	}
	return provider.LoadSpec(path, s)
}

func (p *CompositeProvider) GetResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()
	for _, provider := range p.Providers {
		g, err := provider.GetResourceGraph()
		if err != nil {
			return nil, err
		}
		graph.Merge(g)
	}
	return graph, nil
}

func (p *CompositeProvider) LoadState(ctx context.Context) (*state.State, error) {
	var state *state.State = nil
	for _, provider := range p.Providers {
		s, err := provider.LoadState(ctx)
		if err != nil {
			return nil, err
		}

		if state == nil {
			state = s
		} else {
			state.Merge(s)
		}
	}
	return state, nil
}

func (p *CompositeProvider) PutResourceState(ctx context.Context, URN string, state *state.ResourceState) error {
	provider := p.providerForType(state.Type)
	if provider == nil {
		return fmt.Errorf("no provider found for resource type %s", state.Type)
	}
	return provider.PutResourceState(ctx, URN, state)
}

func (p *CompositeProvider) DeleteResourceState(ctx context.Context, state *state.ResourceState) error {
	provider := p.providerForType(state.Type)
	if provider == nil {
		return fmt.Errorf("no provider found for resource type %s", state.Type)
	}
	return provider.DeleteResourceState(ctx, state)
}

func (p *CompositeProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	provider := p.providerForType(resourceType)
	if provider == nil {
		return nil, fmt.Errorf("no provider found for resource type %s", resourceType)
	}
	return provider.Create(ctx, ID, resourceType, data)
}

func (p *CompositeProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	provider := p.providerForType(resourceType)
	if provider == nil {
		return nil, fmt.Errorf("no provider found for resource type %s", resourceType)
	}
	return provider.Update(ctx, ID, resourceType, data, state)
}

func (p *CompositeProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	provider := p.providerForType(resourceType)
	if provider == nil {
		return fmt.Errorf("no provider found for resource type %s", resourceType)
	}
	return provider.Delete(ctx, ID, resourceType, state)
}

// Helper methods
func (p *CompositeProvider) providerForKind(kind string) project.Provider {
	for _, provider := range p.Providers {
		for _, k := range provider.GetSupportedKinds() {
			if k == kind {
				return provider
			}
		}
	}
	return nil
}

func (p *CompositeProvider) providerForType(resourceType string) project.Provider {
	for _, provider := range p.Providers {
		for _, t := range provider.GetSupportedTypes() {
			if t == resourceType {
				return provider
			}
		}
	}
	return nil
}
