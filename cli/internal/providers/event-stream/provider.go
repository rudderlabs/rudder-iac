package eventstream

import (
	"context"
	"fmt"

	esClient "github.com/rudderlabs/rudder-iac/api/client/event-stream"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	sourceHandler "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
)

type handler interface {
	LoadSpec(path string, s *specs.Spec) error
	Validate(graph *resources.Graph) error
	ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error)
	GetResources() ([]*resources.Resource, error)
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
	Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error)
	LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error)
	MapRemoteToState(collection *resources.RemoteResources) (*state.State, error)
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error)
	FormatForExport(
		collection *resources.RemoteResources,
		idNamer namer.Namer,
		inputResolver resolver.ReferenceResolver,
	) ([]writer.FormattableEntity, error)
}

var _ provider.Provider = &Provider{}

const importDir = "event-stream"

type Provider struct {
	provider.EmptyProvider
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
	p.handlers[sourceHandler.ResourceType] = sourceHandler.NewHandler(client, importDir)
	return p
}

func (p *Provider) SupportedKinds() []string {
	kinds := make([]string, 0, len(p.kindToType))
	for kind := range p.kindToType {
		kinds = append(kinds, kind)
	}
	return kinds
}

func (p *Provider) SupportedTypes() []string {
	types := make([]string, 0, len(p.kindToType))
	for _, t := range p.kindToType {
		types = append(types, t)
	}
	return types
}

func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	resourceType, ok := p.kindToType[s.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported kind: %s", s.Kind)
	}
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.ParseSpec(path, s)
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

func (p *Provider) LoadLegacySpec(path string, s *specs.Spec) error {
	// Empty implementation for now
	return fmt.Errorf("not implemented")
}

func (p *Provider) Validate(graph *resources.Graph) error {
	for resourceType, handler := range p.handlers {
		if err := handler.Validate(graph); err != nil {
			return fmt.Errorf("validating %s: %w", resourceType, err)
		}
	}
	return nil
}

func (p *Provider) ResourceGraph() (*resources.Graph, error) {
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

func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
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

func (p *Provider) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	s := state.EmptyState()
	for resourceType, handler := range p.handlers {
		providerState, err := handler.MapRemoteToState(collection)
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

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.Import(ctx, ID, data, remoteId)
}

func (p *Provider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
	for _, handler := range p.handlers {
		resources, err := handler.LoadImportable(ctx, idNamer)
		if err != nil {
			return nil, fmt.Errorf("loading importable resources from handler %w", err)
		}
		collection, err = collection.Merge(resources)
		if err != nil {
			return nil, fmt.Errorf("merging importable resource collection for handler %w", err)
		}
	}
	return collection, nil
}

func (p *Provider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	result := make([]writer.FormattableEntity, 0)
	for _, handler := range p.handlers {
		entities, err := handler.FormatForExport(
			collection,
			idNamer,
			inputResolver,
		)
		if err != nil {
			return nil, fmt.Errorf("formatting for export for handler %w", err)
		}
		result = append(result, entities...)
	}
	return result, nil
}
