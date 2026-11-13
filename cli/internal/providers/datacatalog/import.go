package datacatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

func (p *Provider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.RemoteResources, error) {
	var (
		collection = resources.NewRemoteResources()
		err        error
	)

	tasks := make([]tasker.Task, 0)
	for resourceType, provider := range p.providerStore {
		tasks = append(tasks, &entityProviderTask{
			resourceType: resourceType,
			provider:     provider,
		})
	}

	results := tasker.NewResults[*resources.RemoteResources]()
	errs := tasker.RunTasks(ctx, tasks, p.concurrency, false, func(task tasker.Task) error {
		t, ok := task.(*entityProviderTask)
		if !ok {
			return fmt.Errorf("expected entityProviderTask, got %T", task)
		}

		importable, err := t.provider.LoadImportable(ctx, idNamer)
		if err != nil {
			return fmt.Errorf("loading importable resources collection for provider of resource type %s: %w", t.resourceType, err)
		}

		results.Store(t.resourceType, importable)
		return nil
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("error loading importable resources: %w", errors.Join(errs...))
	}

	for _, key := range results.GetKeys() {
		importable, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("importable resource collection not found for key %s", key)
		}
		collection, err = collection.Merge(importable)
		if err != nil {
			return nil, fmt.Errorf("merging importable resource collections: %w", err)
		}
	}

	return collection, nil
}

// LoadManaged loads the managed catalog entities, keyed by their real externalId.
func (p *Provider) LoadManaged(ctx context.Context) (*resources.RemoteResources, error) {
	collection := resources.NewRemoteResources()
	for resourceType, ip := range p.providerStore {
		m, ok := ip.(provider.ManagedExportLoader)
		if !ok {
			return nil, fmt.Errorf("provider for %s cannot export managed resources", resourceType)
		}
		c, err := m.LoadManaged(ctx)
		if err != nil {
			return nil, fmt.Errorf("loading managed %s: %w", resourceType, err)
		}
		if collection, err = collection.Merge(c); err != nil {
			return nil, fmt.Errorf("merging managed %s: %w", resourceType, err)
		}
	}
	return collection, nil
}

func (p *Provider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	normalized := make([]writer.FormattableEntity, 0)
	var entries []importmanifest.ImportEntry

	for resourceType, provider := range p.providerStore {
		entities, providerEntries, err := provider.FormatForExport(
			collection,
			idNamer,
			resolver,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("formatting for export for provider %s: %w", resourceType, err)
		}

		normalized = append(normalized, entities...)
		entries = append(entries, providerEntries...)
	}
	return normalized, entries, nil
}
