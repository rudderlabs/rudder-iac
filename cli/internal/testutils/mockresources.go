package testutils

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

const (
	EventResourceType        = "event"
	PropertyResourceType     = "property"
	TrackingPlanResourceType = "tracking_plan"
)

func NewMockEvent(ID string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(ID, EventResourceType, data)
}

func NewMockProperty(ID string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(ID, PropertyResourceType, data)
}

func NewMockTrackingPlan(ID string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(ID, TrackingPlanResourceType, data)
}

type DataCatalogProvider struct {
	OperationLog []string
}

func (p *DataCatalogProvider) Create(_ context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)

	for k, v := range data {
		payload[k] = v
	}

	payload["id"] = fmt.Sprintf("generated-%s-%s", resourceType, ID)
	payload["operation"] = "create"

	p.logOperation(fmt.Sprintf("create %s %s", resourceType, ID))

	return &payload, nil
}

func (p *DataCatalogProvider) Update(_ context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)

	for k, v := range data {
		payload[k] = v
	}

	payload["operation"] = "update"
	p.logOperation(fmt.Sprintf("update %s %s", resourceType, ID))

	return &payload, nil
}

func (p *DataCatalogProvider) Delete(_ context.Context, ID string, resourceType string, state resources.ResourceData) error {
	payload := make(resources.ResourceData)

	for k, v := range state {
		payload[k] = v
	}

	p.logOperation(fmt.Sprintf("delete %s %s", resourceType, ID))

	return nil
}

func (p *DataCatalogProvider) logOperation(operation string) {
	p.OperationLog = append(p.OperationLog, operation)
}
