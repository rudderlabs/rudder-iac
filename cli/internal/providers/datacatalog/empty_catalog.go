package datacatalog

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

func (m *EmptyCatalog) UpdateEvent(ctx context.Context, id string, eventUpdate *catalog.EventUpdate) (*catalog.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteEvent(ctx context.Context, eventID string) error {
	return nil
}

func (m *EmptyCatalog) GetEvent(ctx context.Context, id string) (*catalog.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) CreateProperty(ctx context.Context, propertyCreate catalog.PropertyCreate) (*catalog.Property, error) {

	return nil, nil
}

func (m *EmptyCatalog) UpdateProperty(ctx context.Context, id string, propertyUpdate *catalog.PropertyUpdate) (*catalog.Property, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteProperty(ctx context.Context, propertyID string) error {
	return nil
}

func (m *EmptyCatalog) GetProperty(ctx context.Context, id string) (*catalog.Property, error) {
	return nil, nil
}

func (m *EmptyCatalog) SetPropertyExternalId(ctx context.Context, id string, externalId string) error {
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

func (m *EmptyCatalog) UpdateTrackingPlanEvent(ctx context.Context, trackingPlanID string, input catalog.EventIdentifierDetail) (*catalog.TrackingPlan, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteTrackingPlan(ctx context.Context, trackingPlanID string) error {
	return nil
}

func (m *EmptyCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanID string, eventID string) error {
	return nil
}

func (m *EmptyCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlanWithIdentifiers, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetTrackingPlans(ctx context.Context) ([]*catalog.TrackingPlanWithIdentifiers, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetTrackingPlanEventWithIdentifiers(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventPropertyIdentifiers, error) {
	return nil, nil
}

func (m *EmptyCatalog) CreateCustomType(ctx context.Context, customTypeCreate catalog.CustomTypeCreate) (*catalog.CustomType, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpdateCustomType(ctx context.Context, id string, customTypeUpdate *catalog.CustomTypeUpdate) (*catalog.CustomType, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetCustomType(ctx context.Context, id string) (*catalog.CustomType, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteCustomType(ctx context.Context, customTypeID string) error {
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

func (m *EmptyCatalog) CreateCategory(ctx context.Context, categoryCreate catalog.CategoryCreate) (*catalog.Category, error) {
	return nil, nil
}

func (m *EmptyCatalog) UpdateCategory(ctx context.Context, id string, categoryUpdate catalog.CategoryUpdate) (*catalog.Category, error) {
	return nil, nil
}

func (m *EmptyCatalog) DeleteCategory(ctx context.Context, categoryID string) error {
	return nil
}

func (m *EmptyCatalog) GetCategory(ctx context.Context, id string) (*catalog.Category, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetEvents(ctx context.Context) ([]*catalog.Event, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetProperties(ctx context.Context) ([]*catalog.Property, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetCustomTypes(ctx context.Context) ([]*catalog.CustomType, error) {
	return nil, nil
}

func (m *EmptyCatalog) GetCategories(ctx context.Context) ([]*catalog.Category, error) {
	return nil, nil
}
