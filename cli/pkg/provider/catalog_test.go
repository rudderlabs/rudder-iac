package provider_test

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
)

var _ catalog.DataCatalog = &EmptyCatalog{}

type EmptyCatalog struct {
}

func (m *EmptyCatalog) CreateEvent(ctx context.Context, eventCreate catalog.EventCreate) (*catalog.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpdateEvent(ctx context.Context, id string, eventUpdate *catalog.Event) (*catalog.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteEvent(ctx context.Context, eventID string) error {
	return nil
}

func (m *EmptyCatalog) CreateProperty(ctx context.Context, propertyCreate catalog.PropertyCreate) (*catalog.Property, error) {

	return nil, nil
}

func (m *EmptyCatalog) UpdateProperty(ctx context.Context, id string, propertyUpdate *catalog.Property) (*catalog.Property, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteProperty(ctx context.Context, propertyID string) error {
	return nil
}

func (m *EmptyCatalog) CreateTrackingPlan(ctx context.Context, trackingPlanCreate catalog.TrackingPlanCreate) (*catalog.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpsertTrackingPlan(ctx context.Context, trackingPlanID string, trackingPlanUpsertEvent catalog.TrackingPlanUpsertEvent) (*catalog.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpdateTrackingPlan(ctx context.Context, trackingPlanID string, name string, description string) (*catalog.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteTrackingPlan(ctx context.Context, trackingPlanID string) error {
	return nil
}

func (m *EmptyCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	return nil
}

func (m *EmptyCatalog) ReadState(ctx context.Context) (*catalog.State, error) {
	return nil, nil
}

func (m *EmptyCatalog) PutResourceState(ctx context.Context, _ catalog.PutStateRequest) error {
	return nil
}

func (m *EmptyCatalog) DeleteResourceState(ctx context.Context, _ catalog.DeleteStateRequest) error {
	return nil
}
