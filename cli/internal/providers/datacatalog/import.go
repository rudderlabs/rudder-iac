package datacatalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

func (p *Provider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()

	tasks := make([]tasker.Task, 0)
	for resourceType, provider := range p.providerStore {
		tasks = append(tasks, &entityProviderTask{
			resourceType: resourceType,
			provider:     provider,
		})
	}

	errs := tasker.RunTasks(ctx, tasks, p.concurrency, false, func(task tasker.Task) error {
		t, ok := task.(*entityProviderTask)
		if !ok {
			return fmt.Errorf("expected entityProviderTask, got %T", task)
		}

		resources, err := t.provider.LoadImportable(ctx, idNamer)
		if err != nil {
			return fmt.Errorf("loading importable resources collection for provider of resource type %s: %w", t.resourceType, err)
		}

		collection, err = collection.Merge(resources)
		if err != nil {
			return fmt.Errorf("merging importable resource collection for provider of resource type %s: %w", t.resourceType, err)
		}

		return nil
	})

	if len(errs) > 0 {
		return nil, fmt.Errorf("error loading importable resources: %w", errors.Join(errs...))
	}

	return collection, nil
}

func (p *Provider) FormatForExport(
	ctx context.Context,
	collection *resources.ResourceCollection,
	idNamer namer.Namer,
	resolver resolver.ReferenceResolver,
) ([]importremote.FormattableEntity, error) {
	normalized := make([]importremote.FormattableEntity, 0)

	for resourceType, provider := range p.providerStore {
		entities, err := provider.FormatForExport(
			ctx,
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
