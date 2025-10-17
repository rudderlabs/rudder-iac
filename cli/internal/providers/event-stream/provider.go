package eventstream

import (
	"context"
	"fmt"

	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	sourceHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type handler interface {
	LoadSpec(path string, s *specs.Spec) error
	Validate(graph *resources.Graph) error
	GetResources() ([]*resources.Resource, error)
	LoadState(ctx context.Context) (*state.State, error)
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
	Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error)
	LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error)
	LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error)
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error)
	FormatForExport(
		ctx context.Context,
		collection *resources.ResourceCollection,
		idNamer namer.Namer,
		inputResolver resolver.ReferenceResolver,
	) ([]importremote.FormattableEntity, error)
}

var _ project.Provider = &Provider{}

type Provider struct {
	kindToType map[string]string
	handlers   map[string]handler
}

func New(client esClient.EventStreamStore) *Provider {
	p := &Provider{
		kindToType: map[string]string{
			"event-stream-source": sourceHandler.ResourceType,
		},
		handlers: make(map[string]handler),
	}
	p.handlers[sourceHandler.ResourceType] = sourceHandler.NewHandler(client)
	return p
}

func (p *Provider) GetSupportedKinds() []string {
	kinds := make([]string, 0, len(p.kindToType))
	for kind := range p.kindToType {
		kinds = append(kinds, kind)
	}
	return kinds
}

func (p *Provider) GetSupportedTypes() []string {
	types := make([]string, 0, len(p.kindToType))
	for _, t := range p.kindToType {
		types = append(types, t)
	}
	return types
}

func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	resourceType, ok := p.kindToType[s.Kind]
	if !ok {
		return fmt.Errorf("unsupported kind: %s", s.Kind)
	}
	handler, ok := p.handlers[resourceType]
	if !ok {
		return fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.LoadSpec(path, s)
}

func (p *Provider) Validate(graph *resources.Graph) error {
	for resourceType, handler := range p.handlers {
		if err := handler.Validate(graph); err != nil {
			return fmt.Errorf("validating %s: %w", resourceType, err)
		}
	}
	return nil
}

func (p *Provider) GetResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()
	for resourceType, handler := range p.handlers {
		resources, err := handler.GetResources()
		if err != nil {
			return nil, fmt.Errorf("getting resources for %s: %w", resourceType, err)
		}
		for _, resource := range resources {
			graph.AddResource(resource)
		}
	}
	return graph, nil
}

func (p *Provider) LoadState(ctx context.Context) (*state.State, error) {
	result := state.EmptyState()
	for _, handler := range p.handlers {
		state, err := handler.LoadState(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading state for %s: %w", handler, err)
		}
		result, err = result.Merge(state)
		if err != nil {
			return nil, fmt.Errorf("merging state for %s: %w", handler, err)
		}
	}
	return result, nil
}

func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()
	for resourceType, handler := range p.handlers {
		c, err := handler.LoadResourcesFromRemote(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", resourceType, err)
		}
		collection, err = collection.Merge(c)
		if err != nil {
			return nil, fmt.Errorf("merging collection for %s: %w", resourceType, err)
		}
	}
	return collection, nil
}

func (p *Provider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	s := state.EmptyState()
	for resourceType, handler := range p.handlers {
		providerState, err := handler.LoadStateFromResources(ctx, collection)
		if err != nil {
			return nil, fmt.Errorf("loading state from provider handler %s: %w", resourceType, err)
		}
		s, err = s.Merge(providerState)
		if err != nil {
			return nil, fmt.Errorf("merging provider states: %w", err)
		}
	}
	return s, nil
}

func (p *Provider) PutResourceState(ctx context.Context, URN string, s *state.ResourceState) error {
	// Not required with the stateless CLI approach
	return nil
}

func (p *Provider) DeleteResourceState(ctx context.Context, st *state.ResourceState) error {
	// Not required with the stateless CLI approach
	return nil
}

func (p *Provider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.Create(ctx, ID, data)
}

func (p *Provider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}

	return handler.Update(ctx, ID, data, state)
}

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.Delete(ctx, ID, state)
}

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.Import(ctx, ID, data, remoteId)
}

func (p *Provider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	return nil, nil
}

func (p *Provider) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	return nil, nil
}

func (p *Provider) GetName() string {
	return "event-stream"
}
