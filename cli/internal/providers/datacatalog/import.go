package datacatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
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

func (p *Provider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, error) {
	normalized := make([]writer.FormattableEntity, 0)

	for resourceType, provider := range p.providerStore {
		entities, err := provider.FormatForExport(
			collection,
			idNamer,
			resolver,
		)
		if err != nil {
			return nil, fmt.Errorf("formatting for export for provider %s: %w", resourceType, err)
		}

		normalized = append(normalized, entities...)
	}
	return normalized, nil
}
