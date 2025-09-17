package list_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/list"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCatalog implements the needed parts of catalog.DataCatalog for testing
type MockCatalog struct {
	trackingPlans []catalog.TrackingPlan
	err           error
}

func (m *MockCatalog) ListTrackingPlans(ctx context.Context) ([]catalog.TrackingPlan, error) {
	return m.trackingPlans, m.err
}

// Stub implementations for catalog.DataCatalog interface
func (m *MockCatalog) CreateTrackingPlan(ctx context.Context, input catalog.TrackingPlanCreate) (*catalog.TrackingPlan, error) {
	return nil, nil
}
func (m *MockCatalog) UpsertTrackingPlan(ctx context.Context, id string, input catalog.TrackingPlanUpsertEvent) (*catalog.TrackingPlan, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateTrackingPlan(ctx context.Context, id string, name, description string) (*catalog.TrackingPlan, error) {
	return nil, nil
}
func (m *MockCatalog) DeleteTrackingPlan(ctx context.Context, id string) error { return nil }
func (m *MockCatalog) DeleteTrackingPlanEvent(ctx context.Context, trackingPlanId string, eventId string) error {
	return nil
}
func (m *MockCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlanWithIdentifiers, error) {
	return nil, nil
}
func (m *MockCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return nil, nil
}
func (m *MockCatalog) GetTrackingPlanEventWithIdentifiers(ctx context.Context, id, eventId string) (*catalog.TrackingPlanEventPropertyIdentifiers, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateTrackingPlanEvent(ctx context.Context, id string, input catalog.EventIdentifierDetail) (*catalog.TrackingPlan, error) {
	return nil, nil
}
func (m *MockCatalog) CreateEvent(ctx context.Context, input catalog.EventCreate) (*catalog.Event, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateEvent(ctx context.Context, id string, input *catalog.Event) (*catalog.Event, error) {
	return nil, nil
}
func (m *MockCatalog) DeleteEvent(ctx context.Context, id string) error { return nil }
func (m *MockCatalog) GetEvent(ctx context.Context, id string) (*catalog.Event, error) {
	return nil, nil
}
func (m *MockCatalog) CreateProperty(ctx context.Context, input catalog.PropertyCreate) (*catalog.Property, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateProperty(ctx context.Context, id string, input *catalog.Property) (*catalog.Property, error) {
	return nil, nil
}
func (m *MockCatalog) DeleteProperty(ctx context.Context, id string) error { return nil }
func (m *MockCatalog) GetProperty(ctx context.Context, id string) (*catalog.Property, error) {
	return nil, nil
}
func (m *MockCatalog) CreateCustomType(ctx context.Context, input catalog.CustomTypeCreate) (*catalog.CustomType, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateCustomType(ctx context.Context, id string, input *catalog.CustomType) (*catalog.CustomType, error) {
	return nil, nil
}
func (m *MockCatalog) DeleteCustomType(ctx context.Context, id string) error { return nil }
func (m *MockCatalog) GetCustomType(ctx context.Context, id string) (*catalog.CustomType, error) {
	return nil, nil
}
func (m *MockCatalog) CreateCategory(ctx context.Context, input catalog.CategoryCreate) (*catalog.Category, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateCategory(ctx context.Context, id string, input catalog.CategoryUpdate) (*catalog.Category, error) {
	return nil, nil
}
func (m *MockCatalog) DeleteCategory(ctx context.Context, id string) error { return nil }
func (m *MockCatalog) GetCategory(ctx context.Context, id string) (*catalog.Category, error) {
	return nil, nil
}
func (m *MockCatalog) ReadState(ctx context.Context) (*catalog.State, error) {
	return nil, nil
}
func (m *MockCatalog) PutResourceState(ctx context.Context, req catalog.PutStateRequest) error {
	return nil
}
func (m *MockCatalog) DeleteResourceState(ctx context.Context, req catalog.DeleteStateRequest) error {
	return nil
}

// Add the new List methods
func (m *MockCatalog) ListTrackingPlansWithFilter(ctx context.Context, ids []string) ([]catalog.TrackingPlan, error) {
	return nil, nil
}
func (m *MockCatalog) ListEvents(ctx context.Context, trackingPlanIds []string, page int) (*catalog.EventListResponse, error) {
	return nil, nil
}
func (m *MockCatalog) ListProperties(ctx context.Context, trackingPlanIds []string, page int) (*catalog.PropertyListResponse, error) {
	return nil, nil
}
func (m *MockCatalog) ListCustomTypes(ctx context.Context, page int) (*catalog.CustomTypeListResponse, error) {
	return nil, nil
}
func (m *MockCatalog) ListCategories(ctx context.Context, page int) (*catalog.CategoryListResponse, error) {
	return nil, nil
}

func TestListTrackingPlans(t *testing.T) {
	ctx := context.Background()

	// Test data
	trackingPlans := []catalog.TrackingPlan{
		{
			ID:          "tp-1",
			Name:        "E-commerce Plan",
			Description: strPtr("Main tracking plan"),
			Version:     1,
			WorkspaceID: "ws-1",
		},
		{
			ID:          "tp-2",
			Name:        "Analytics Plan",
			Description: strPtr("Analytics tracking plan"),
			Version:     2,
			WorkspaceID: "ws-1",
		},
	}

	mockCatalog := &MockCatalog{
		trackingPlans: trackingPlans,
	}

	// Create lister
	lister := list.NewTrackingPlanLister(mockCatalog)

	// Test listing
	plans, err := lister.List(ctx)
	require.NoError(t, err)
	require.Len(t, plans, 2)

	assert.Equal(t, "tp-1", plans[0].ID)
	assert.Equal(t, "E-commerce Plan", plans[0].Name)
	assert.Equal(t, "tp-2", plans[1].ID)
	assert.Equal(t, "Analytics Plan", plans[1].Name)
}

func TestListTrackingPlansError(t *testing.T) {
	ctx := context.Background()

	mockCatalog := &MockCatalog{
		err: assert.AnError,
	}

	lister := list.NewTrackingPlanLister(mockCatalog)

	plans, err := lister.List(ctx)
	require.Error(t, err)
	assert.Nil(t, plans)
}

func strPtr(s string) *string {
	return &s
}