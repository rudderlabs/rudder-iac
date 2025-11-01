package provider

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

type Handler interface {
	LoadSpec(path string, s *specs.Spec) error
	Validate(graph *resources.Graph) error
	ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error)
	GetResources() ([]*resources.Resource, error)
	Create(ctx context.Context, data any) (*resources.ResourceData, error)
	Update(ctx context.Context, data any, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
	Import(ctx context.Context, data any, remoteId string) (*resources.ResourceData, error)
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

type BaseProvider struct {
	name       string
	handlers   map[string]Handler
	kindToType map[string]string
}

func NewBaseProvider(name string, handlers map[string]Handler, kindToType map[string]string) *BaseProvider {
	return &BaseProvider{name, handlers, kindToType}
}

func (p *BaseProvider) GetName() string {
	return p.name
}

func (p *BaseProvider) GetSupportedKinds() []string {
	kinds := make([]string, 0, len(p.kindToType))
	for kind := range p.kindToType {
		kinds = append(kinds, kind)
	}
	return kinds
}

func (p *BaseProvider) GetSupportedTypes() []string {
	types := make([]string, 0, len(p.kindToType))
	for _, t := range p.kindToType {
		types = append(types, t)
	}
	return types
}

func (p *BaseProvider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
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

func (p *BaseProvider) LoadSpec(path string, s *specs.Spec) error {
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

func (p *BaseProvider) Validate(graph *resources.Graph) error {
	for resourceType, handler := range p.handlers {
		if err := handler.Validate(graph); err != nil {
			return fmt.Errorf("validating %s: %w", resourceType, err)
		}
	}
	return nil
}

func (p *BaseProvider) GetResourceGraph() (*resources.Graph, error) {
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

func (p *BaseProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
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

func (p *BaseProvider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
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

func (p *BaseProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *BaseProvider) CreateRaw(ctx context.Context, resource *resources.Resource) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resource.Type()]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resource.Type())
	}
	return handler.Create(ctx, resource.RawData())
}

func (p *BaseProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *BaseProvider) UpdateRaw(ctx context.Context, resource *resources.Resource, state resources.ResourceData) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resource.Type()]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resource.Type())
	}
	return handler.Update(ctx, resource.RawData(), state)
}

func (p *BaseProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	handler, ok := p.handlers[resourceType]
	if !ok {
		return fmt.Errorf("no handler for resource type: %s", resourceType)
	}
	return handler.Delete(ctx, ID, state)
}

func (p *BaseProvider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	return nil, errNotImplemented
}

func (p *BaseProvider) ImportRaw(ctx context.Context, resource *resources.Resource, remoteId string) (*resources.ResourceData, error) {
	handler, ok := p.handlers[resource.Type()]
	if !ok {
		return nil, fmt.Errorf("no handler for resource type: %s", resource.Type())
	}
	return handler.Import(ctx, resource.RawData(), remoteId)
}

func (p *BaseProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()
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

func (p *BaseProvider) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	result := make([]importremote.FormattableEntity, 0)
	for _, handler := range p.handlers {
		entities, err := handler.FormatForExport(
			ctx,
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
