package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
	"golang.org/x/exp/maps"
)

type CompositeProvider struct {
	concurrency     int
	Providers       map[string]Provider
	registeredKinds map[string]Provider
	registeredTypes map[string]Provider
}

func NewCompositeProvider(providers map[string]Provider) (Provider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("at least one provider must be specified")
	}

	registeredKinds := make(map[string]Provider)
	registeredTypes := make(map[string]Provider)

	for _, provider := range providers {
		for _, kind := range provider.SupportedKinds() {
			if _, ok := registeredKinds[kind]; ok {
				return nil, fmt.Errorf("duplicate kind '%s' supported by multiple providers", kind)
			}
			registeredKinds[kind] = provider
		}
		for _, t := range provider.SupportedTypes() {
			if _, ok := registeredTypes[t]; ok {
				return nil, fmt.Errorf("duplicate type '%s' supported by multiple providers", t)
			}
			registeredTypes[t] = provider
		}
	}

	return &CompositeProvider{
		concurrency:     config.GetConfig().Concurrency.CompositeProvider,
		Providers:       providers,
		registeredKinds: registeredKinds,
		registeredTypes: registeredTypes,
	}, nil
}

func (p *CompositeProvider) SupportedKinds() []string {
	return maps.Keys(p.registeredKinds)
}

func (p *CompositeProvider) SupportedTypes() []string {
	return maps.Keys(p.registeredTypes)
}

func (p *CompositeProvider) Validate(graph *resources.Graph) error {
	for _, provider := range p.Providers {
		if err := provider.Validate(graph); err != nil {
			return err
		}
	}
	return nil
}

func (p *CompositeProvider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	provider, err := p.providerForKind(s.Kind)
	if err != nil {
		return nil, err
	}

	return provider.ParseSpec(path, s)
}

func (p *CompositeProvider) LoadSpec(path string, s *specs.Spec) error {
	provider, err := p.providerForKind(s.Kind)
	if err != nil {
		return err
	}

	return provider.LoadSpec(path, s)
}

func (p *CompositeProvider) ResourceGraph() (*resources.Graph, error) {
	graph := resources.NewGraph()
	for _, provider := range p.Providers {
		g, err := provider.ResourceGraph()
		if err != nil {
			return nil, err
		}
		graph.Merge(g)
	}
	return graph, nil
}

func (p *CompositeProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	provider, err := p.providerForType(resourceType)
	if err != nil {
		return nil, err
	}
	return provider.Create(ctx, ID, resourceType, data)
}

func (p *CompositeProvider) CreateRaw(ctx context.Context, resource *resources.Resource) (any, error) {
	provider, err := p.providerForType(resource.Type())
	if err != nil {
		return nil, err
	}
	return provider.CreateRaw(ctx, resource)
}

func (p *CompositeProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	provider, err := p.providerForType(resourceType)
	if err != nil {
		return nil, err
	}
	return provider.Update(ctx, ID, resourceType, data, state)
}

func (p *CompositeProvider) UpdateRaw(ctx context.Context, resource *resources.Resource, oldData any, oldState any) (any, error) {
	provider, err := p.providerForType(resource.Type())
	if err != nil {
		return nil, err
	}
	return provider.UpdateRaw(ctx, resource, oldData, oldState)
}

func (p *CompositeProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	provider, err := p.providerForType(resourceType)
	if err != nil {
		return err
	}
	return provider.Delete(ctx, ID, resourceType, state)
}

func (p *CompositeProvider) DeleteRaw(ctx context.Context, ID string, resourceType string, oldData any, oldState any) error {
	provider, err := p.providerForType(resourceType)
	if err != nil {
		return err
	}
	return provider.DeleteRaw(ctx, ID, resourceType, oldData, oldState)
}

func (p *CompositeProvider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error) {
	provider, err := p.providerForType(resourceType)
	if err != nil {
		return nil, err
	}
	return provider.Import(ctx, ID, resourceType, data, remoteId)
}

func (p *CompositeProvider) ImportRaw(ctx context.Context, resource *resources.Resource, remoteId string) (any, error) {
	provider, err := p.providerForType(resource.Type())
	if err != nil {
		return nil, err
	}
	return provider.ImportRaw(ctx, resource, remoteId)
}

type compositeProviderTask struct {
	name     string
	provider Provider
}

func (t *compositeProviderTask) Id() string {
	return t.name
}

func (t *compositeProviderTask) Dependencies() []string {
	return []string{}
}

var _ tasker.Task = &compositeProviderTask{}

// LoadImportableResources loads the resources from upstream which are
// present in the workspace and ready to be imported.
func (p *CompositeProvider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	var (
		collection = resources.NewRemoteResources()
		err        error
	)

	tasks := make([]tasker.Task, 0)
	for name, provider := range p.Providers {
		tasks = append(tasks, &compositeProviderTask{
			name:     name,
			provider: provider,
		})
	}

	results := tasker.NewResults[*resources.RemoteResources]()
	errs := tasker.RunTasks(ctx, tasks, p.concurrency, false, func(task tasker.Task) error {
		t, ok := task.(*compositeProviderTask)
		if !ok {
			return fmt.Errorf("expected compositeProviderTask, got %T", task)
		}
		importable, err := t.provider.LoadImportable(ctx, idNamer)
		if err != nil {
			return fmt.Errorf("loading importable resources for composite provider %s: %w", t.name, err)
		}
		results.Store(t.name, importable)
		return nil
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("error loading importable resources for composite provider: %w", errors.Join(errs...))
	}

	for _, key := range results.GetKeys() {
		importable, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("importable resource collection not found for composite provider %s", key)
		}
		collection, err = collection.Merge(importable)
		if err != nil {
			return nil, fmt.Errorf("merging importable resource collections for composite provider %s: %w", key, err)
		}
	}

	return collection, nil
}

func (p *CompositeProvider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	formattable := make([]writer.FormattableEntity, 0)

	for name, provider := range p.Providers {
		entities, err := provider.FormatForExport(
			collection,
			idNamer,
			resolver,
		)
		if err != nil {
			return nil, fmt.Errorf("formatting for export for provider %s: %w", name, err)
		}
		formattable = append(formattable, entities...)
	}

	return formattable, nil
}

// Helper methods
func (p *CompositeProvider) providerForKind(kind string) (Provider, error) {
	provider, ok := p.registeredKinds[kind]
	if !ok {
		return nil, &ErrUnsupportedSpecKind{Kind: kind}
	}
	return provider, nil
}

func (p *CompositeProvider) providerForType(resourceType string) (Provider, error) {
	provider, ok := p.registeredTypes[resourceType]
	if !ok {
		return nil, &ErrUnsupportedResourceType{ResourceType: resourceType}
	}
	return provider, nil
}

func (p *CompositeProvider) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	var (
		collection = resources.NewRemoteResources()
		err        error
	)

	tasks := make([]tasker.Task, 0)
	for name, provider := range p.Providers {
		tasks = append(tasks, &compositeProviderTask{
			name:     name,
			provider: provider,
		})
	}

	results := tasker.NewResults[*resources.RemoteResources]()
	errs := tasker.RunTasks(ctx, tasks, p.concurrency, false, func(task tasker.Task) error {
		t, ok := task.(*compositeProviderTask)
		if !ok {
			return fmt.Errorf("expected compositeProviderTask, got %T", task)
		}
		r, err := t.provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return fmt.Errorf("loading resources from remote for composite provider %s: %w", t.name, err)
		}
		results.Store(t.name, r)
		return nil
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("error loading resources from remote for composite provider: %w", errors.Join(errs...))
	}

	for _, key := range results.GetKeys() {
		remoteResources, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("remote resource collection not found for composite provider %s", key)
		}
		collection, err = collection.Merge(remoteResources)
		if err != nil {
			return nil, fmt.Errorf("merging resources from remote for composite provider %s: %w", key, err)
		}
	}

	return collection, nil
}

func (p *CompositeProvider) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	s := state.EmptyState()
	// Load and merge state from all providers
	for _, provider := range p.Providers {
		state, err := provider.MapRemoteToState(collection)
		if err != nil {
			return nil, err
		}
		if s == nil {
			s = state
		} else {
			s, err = s.Merge(state)
			if err != nil {
				return nil, err
			}
		}
	}
	return s, nil
}

// ConsolidateSync calls ConsolidateSync on all registered providers
func (p *CompositeProvider) ConsolidateSync(ctx context.Context, state *state.State) error {
	for _, provider := range p.Providers {
		if err := provider.ConsolidateSync(ctx, state); err != nil {
			return fmt.Errorf("provider consolidate sync failed: %w", err)
		}
	}
	return nil
}
