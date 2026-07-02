package providers

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
)

// NewRemoteCatalogPlanProvider builds a plan provider that fetches the tracking
// plan from the remote workspace by ID, initialising the data catalog API client
// from the configured credentials.
func NewRemoteCatalogPlanProvider(trackingPlanID string) (*JSONSchemaPlanProvider, error) {
	if trackingPlanID == "" {
		return nil, fmt.Errorf("tracking-plan-id is required")
	}

	deps, err := app.NewDeps()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dependencies: %w", err)
	}

	dataCatalogClient, err := catalog.NewRudderDataCatalog(
		deps.Client(),
		catalog.WithConcurrency(config.GetConfig().Concurrency.CatalogClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize data catalog client: %w", err)
	}

	return NewJSONSchemaPlanProvider(trackingPlanID, dataCatalogClient), nil
}
