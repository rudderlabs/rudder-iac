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
	Providers       []project.Provider
	registeredKinds map[string]project.Provider
	registeredTypes map[string]project.Provider
}

func NewCompositeProvider(providers ...project.Provider) (*CompositeProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider must be specified")
	}

	registeredKinds := make(map[string]project.Provider)
	registeredTypes := make(map[string]project.Provider)

	for _, provider := range providers {
		for _, kind := range provider.GetSupportedKinds() {
			if _, ok := registeredKinds[kind]; ok {
				return nil, fmt.Errorf("duplicate kind '%s' supported by multiple providers", kind)
			}
			registeredKinds[kind] = provider
		}
		for _, t := range provider.GetSupportedTypes() {
			if _, ok := registeredTypes[t]; ok {
				return nil, fmt.Errorf("duplicate type '%s' supported by multiple providers", t)
			}
			registeredTypes[t] = provider
		}
	}

	return &CompositeProvider{
		Providers:       providers,
		registeredKinds: registeredKinds,
		registeredTypes: registeredTypes,
	}, nil
}

func (p *CompositeProvider) GetSupportedKinds() []string {
	return maps.Keys(p.registeredKinds)
}

func (p *CompositeProvider) GetSupportedTypes() []string {
	return maps.Keys(p.registeredTypes)
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
			state, err = state.Merge(s)
			if err != nil {
				return nil, fmt.Errorf("error merging provider states")
			}
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
	return p.registeredKinds[kind]
}

func (p *CompositeProvider) providerForType(resourceType string) project.Provider {
	return p.registeredTypes[resourceType]
}
