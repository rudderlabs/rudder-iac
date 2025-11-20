package datacatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	dcstate "github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

var resourceTypeCollection = map[string]catalog.ResourceCollection{
	dcstate.PropertyResourceType:     catalog.ResourceCollectionProperties,
	dcstate.EventResourceType:        catalog.ResourceCollectionEvents,
	dcstate.TrackingPlanResourceType: catalog.ResourceCollectionTrackingPlans,
	dcstate.CustomTypeResourceType:   catalog.ResourceCollectionCustomTypes,
	dcstate.CategoryResourceType:     catalog.ResourceCollectionCategories,
}

type entityProvider interface {
	resourceProvider
	resourceImportProvider
}

type resourceImportProvider interface {
	LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error)
	FormatForExport(
		ctx context.Context,
		collection *resources.ResourceCollection,
		idNamer namer.Namer,
		inputResolver resolver.ReferenceResolver,
	) ([]importremote.FormattableEntity, error)
}

type resourceProvider interface {
	Create(ctx context.Context, ID string, data resources.ResourceData) (*resources.ResourceData, error)
	Update(ctx context.Context, ID string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error)
	Delete(ctx context.Context, ID string, state resources.ResourceData) error
	Import(ctx context.Context, ID string, data resources.ResourceData, remoteId string) (*resources.ResourceData, error)
	LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error)
	LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error)
}

func (p *Provider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Create(ctx, ID, data)
}

func (p *Provider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Update(ctx, ID, data, state)
}

func (p *Provider) Delete(ctx context.Context, ID string, resourceType string, data resources.ResourceData) error {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Delete(ctx, ID, data)
}

func (p *Provider) Import(ctx context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	provider, ok := p.providerStore[resourceType]
	if !ok {
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
	return provider.Import(ctx, ID, data, remoteId)
}

type entityProviderTask struct {
	resourceType string
	provider     entityProvider
}

func (t *entityProviderTask) Id() string {
	return t.resourceType
}

func (t *entityProviderTask) Dependencies() []string {
	return []string{}
}

var _ tasker.Task = &entityProviderTask{}

// LoadResourcesFromRemote loads all resources from remote catalog into a ResourceCollection
func (p *Provider) LoadResourcesFromRemote(ctx context.Context) (*resources.ResourceCollection, error) {
	log.Debug("loading all resources from remote catalog")

	var (
		collection = resources.NewResourceCollection()
		err        error
	)

	tasks := make([]tasker.Task, 0)
	for resourceType, provider := range p.providerStore {
		tasks = append(tasks, &entityProviderTask{
			resourceType: resourceType,
			provider:     provider,
		})
	}

	results := tasker.NewResults[*resources.ResourceCollection]()
	errs := tasker.RunTasks(ctx, tasks, p.concurrency, false, func(task tasker.Task) error {
		t, ok := task.(*entityProviderTask)
		if !ok {
			return fmt.Errorf("expected entityProviderTask, got %T", task)
		}

		c, err := t.provider.LoadResourcesFromRemote(ctx)
		if err != nil {
			return fmt.Errorf("loading resources from remote for provider of resource type %s: %w", t.resourceType, err)
		}

		results.Store(t.resourceType, c)
		return nil
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("error loading resources from remote: %w", errors.Join(errs...))
	}

	for _, key := range results.GetKeys() {
		remoteResources, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("importable resource collection not found for key %s", key)
		}
		collection, err = collection.Merge(remoteResources)
		if err != nil {
			return nil, fmt.Errorf("merging resources from remote: %w", err)
		}
	}

	return collection, nil
}

// LoadStateFromResources reconstructs CLI state from loaded remote resources
func (p *Provider) LoadStateFromResources(ctx context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	log.Debug("reconstructing state from loaded resources")
	s := state.EmptyState()

	// loop over stateless resources and load state
	for resourceType, provider := range p.providerStore {
		providerState, err := provider.LoadStateFromResources(ctx, collection)
		if err != nil {
			return nil, fmt.Errorf("LoadStateFromResources: error loading state from provider store %s: %w", resourceType, err)
		}

		s, err = s.Merge(providerState)
		if err != nil {
			return nil, fmt.Errorf("LoadStateFromResources: error merging provider states: %w", err)
		}
	}

	log.Debug("reconstructed state", "resource_count", len(s.Resources))
	return s, nil
}
