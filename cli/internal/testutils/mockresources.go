package testutils

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

const (
	EventResourceType        = "event"
	PropertyResourceType     = "property"
	TrackingPlanResourceType = "tracking-plan"
)

func NewMockEvent(ID string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(ID, EventResourceType, data, make([]string, 0))
}

func NewMockProperty(ID string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(ID, PropertyResourceType, data, make([]string, 0))
}

func NewMockTrackingPlan(ID string, data resources.ResourceData) *resources.Resource {
	return resources.NewResource(ID, TrackingPlanResourceType, data, make([]string, 0))
}

type OperationLogEntry struct {
	Operation string
	Args      []interface{}
}

type DataCatalogProvider struct {
	InitialState *state.State
	OperationLog []OperationLogEntry
}

func (p *DataCatalogProvider) LoadState(_ context.Context) (*state.State, error) {
	return p.InitialState, nil
}

func (p *DataCatalogProvider) PutResourceState(_ context.Context, ID string, state *state.ResourceState) error {
	p.logOperation("PutResourceState", ID, state)
	return nil
}

func (p *DataCatalogProvider) DeleteResourceState(_ context.Context, state *state.ResourceState) error {
	p.logOperation("DeleteResourceState", state)
	return nil
}

func (p *DataCatalogProvider) Create(_ context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", resourceType, ID)

	p.logOperation("Create", ID, resourceType, data)

	return &payload, nil
}

func (p *DataCatalogProvider) Update(_ context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", resourceType, ID)

	p.logOperation("Update", ID, resourceType, data, state)

	return &payload, nil
}

func (p *DataCatalogProvider) Delete(_ context.Context, ID string, resourceType string, state resources.ResourceData) error {
	p.logOperation("Delete", ID, resourceType, state)
	return nil
}

func (p *DataCatalogProvider) logOperation(operation string, args ...interface{}) {
	p.OperationLog = append(p.OperationLog, OperationLogEntry{
		Operation: operation,
		Args:      args,
	})
}
