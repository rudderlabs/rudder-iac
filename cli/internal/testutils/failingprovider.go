package testutils

import (
	"context"
	"fmt"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// FailingDataCatalogProvider wraps DataCatalogProvider to allow selective failures
type FailingDataCatalogProvider struct {
	*DataCatalogProvider
	FailingResources []string
}

func (p *FailingDataCatalogProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	if slices.Contains(p.FailingResources, ID) {
		return nil, fmt.Errorf("simulated failure for %s", ID)
	}
	return p.DataCatalogProvider.Create(ctx, ID, resourceType, data)
}

func (p *FailingDataCatalogProvider) Delete(ctx context.Context, ID string, resourceType string, state resources.ResourceData) error {
	if slices.Contains(p.FailingResources, ID) {
		return fmt.Errorf("simulated delete failure for %s", ID)
	}
	return p.DataCatalogProvider.Delete(ctx, ID, resourceType, state)
}
