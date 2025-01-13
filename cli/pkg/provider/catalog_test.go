package provider_test

import (
	"context"

	"github.com/rudderlabs/rudder-iac/api/client"
)

var _ client.DataCatalog = &EmptyCatalog{}

type EmptyCatalog struct {
}

const (
	typeEvent        = "event"
	typeProperty     = "property"
	typeTrackingPlan = "tracking-plan"
)

func (m *EmptyCatalog) CreateEvent(ctx context.Context, eventCreate client.EventCreate) (*client.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpdateEvent(ctx context.Context, id string, eventUpdate *client.Event) (*client.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteEvent(ctx context.Context, eventID string) error {
	return nil
}

func (m *EmptyCatalog) CreateProperty(ctx context.Context, propertyCreate client.PropertyCreate) (*client.Property, error) {

	return nil, nil
}

func (m *EmptyCatalog) UpdateProperty(ctx context.Context, id string, propertyUpdate *client.Property) (*client.Property, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteProperty(ctx context.Context, propertyID string) error {
	return nil
}

func (m *EmptyCatalog) CreateTrackingPlan(ctx context.Context, trackingPlanCreate client.TrackingPlanCreate) (*client.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpsertTrackingPlan(ctx context.Context, trackingPlanID string, trackingPlanUpsertEvent client.TrackingPlanUpsertEvent) (*client.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpdateTrackingPlan(ctx context.Context, trackingPlanID string, name string, description string) (*client.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteTrackingPlan(ctx context.Context, trackingPlanID string) error {
	return nil
}

func (m *EmptyCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	return nil
}
