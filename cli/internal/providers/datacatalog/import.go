package datacatalog

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

func (p *Provider) LoadImportable(ctx context.Context, idNamer namer.Namer) (*resources.ResourceCollection, error) {
	collection := resources.NewResourceCollection()

	for _, provider := range p.providerStore {
		if _, ok := provider.(resourceImportProvider); !ok {
			continue
		}

		resources, err := provider.(resourceImportProvider).LoadImportable(ctx, idNamer)
		if err != nil {
			return nil, fmt.Errorf("loading importable resources from provider %w", err)
		}

		collection, err = collection.Merge(resources)
		if err != nil {
			return nil, fmt.Errorf("merging importable resource collection for provider %w", err)
		}
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

	for _, provider := range p.providerStore {
		if _, ok := provider.(resourceImportProvider); !ok {
			continue
		}

		entities, err := provider.(resourceImportProvider).FormatForExport(
			ctx,
			collection,
			idNamer,
			resolver,
		)
		if err != nil {
			return nil, fmt.Errorf("normalizing for import for provider %w", err)
		}

		normalized = append(normalized, entities...)
	}
	return normalized, nil
}
