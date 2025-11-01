package testutils

import (
	"context"
	"fmt"
	"sync"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
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
	InitialState       *state.State
	ReconstructedState *state.State
	InitialResources   *resources.ResourceCollection
	OperationLog       []OperationLogEntry
	operationMutex     sync.Mutex
}

func (p *DataCatalogProvider) LoadResourcesFromRemote(_ context.Context) (*resources.ResourceCollection, error) {
	return p.InitialResources, nil
}

func (p *DataCatalogProvider) LoadStateFromResources(_ context.Context, collection *resources.ResourceCollection) (*state.State, error) {
	return p.ReconstructedState, nil
}

func (p *DataCatalogProvider) Create(_ context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", resourceType, ID)

	p.logOperation("Create", ID, resourceType, data)

	return &payload, nil
}

func (p *DataCatalogProvider) CreateRaw(_ context.Context, data *resources.Resource) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", "mock-type", "mock-id")

	p.logOperation("Create", "mock-id", "mock-type", data)

	return &payload, nil
}

func (p *DataCatalogProvider) Import(_ context.Context, ID string, resourceType string, data resources.ResourceData, workspaceId, remoteId string) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", resourceType, ID)

	p.logOperation("Import", ID, resourceType, data, workspaceId, remoteId)

	return &payload, nil
}

func (p *DataCatalogProvider) ImportRaw(_ context.Context, data *resources.Resource, remoteId string) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", "mock-type", "mock-id")

	p.logOperation("Import", "mock-id", "mock-type", data, remoteId)

	return &payload, nil
}

func (p *DataCatalogProvider) Update(_ context.Context, ID string, resourceType string, data resources.ResourceData, state resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", resourceType, ID)

	p.logOperation("Update", ID, resourceType, data, state)

	return &payload, nil
}

func (p *DataCatalogProvider) UpdateRaw(_ context.Context, data *resources.Resource, state resources.ResourceData) (*resources.ResourceData, error) {
	payload := make(resources.ResourceData)
	payload["id"] = fmt.Sprintf("generated-%s-%s", "mock-type", "mock-id")

	p.logOperation("Update", "mock-id", "mock-type", data, state)

	return &payload, nil
}

func (p *DataCatalogProvider) Delete(_ context.Context, ID string, resourceType string, state resources.ResourceData) error {
	p.logOperation("Delete", ID, resourceType, state)
	return nil
}

func (p *DataCatalogProvider) logOperation(operation string, args ...interface{}) {
	p.operationMutex.Lock()
	defer p.operationMutex.Unlock()
	p.OperationLog = append(p.OperationLog, OperationLogEntry{
		Operation: operation,
		Args:      args,
	})
}
