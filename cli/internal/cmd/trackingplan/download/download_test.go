package download_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rudderlabs/rudder-iac/api/client/catalog"
	"github.com/rudderlabs/rudder-iac/cli/internal/cmd/trackingplan/download"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCatalog implements the catalog.DataCatalog interface for testing
type MockCatalog struct {
	trackingPlans []catalog.TrackingPlan
	events        []catalog.Event
	properties    []catalog.Property
	customTypes   []catalog.CustomType
	categories    []catalog.Category
	err           error
}

func (m *MockCatalog) ListTrackingPlans(ctx context.Context) ([]catalog.TrackingPlan, error) {
	return m.trackingPlans, m.err
}

func (m *MockCatalog) GetTrackingPlan(ctx context.Context, id string) (*catalog.TrackingPlanWithIdentifiers, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Find the matching tracking plan and return detailed version
	for _, tp := range m.trackingPlans {
		if tp.ID == id {
			return &catalog.TrackingPlanWithIdentifiers{
				ID:           tp.ID,
				Name:         tp.Name,
				Description:  tp.Description,
				CreationType: tp.CreationType,
				Version:      tp.Version,
				WorkspaceID:  tp.WorkspaceID,
				CreatedAt:    tp.CreatedAt,
				UpdatedAt:    tp.UpdatedAt,
				Events: []catalog.TrackingPlanEventPropertyIdentifiers{
					{
						ID:          "event-" + tp.ID,
						Name:        "Test Event for " + tp.Name,
						Description: "Test event description",
						EventType:   "track",
						WorkspaceId: tp.WorkspaceID,
						CreatedAt:   tp.CreatedAt,
						UpdatedAt:   tp.UpdatedAt,
						Properties: []*catalog.TrackingPlanEventProperty{
							{
								ID:       "prop-" + tp.ID,
								Name:     "test_property",
								Required: true,
							},
						},
					},
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("tracking plan not found: %s", id)
}

func (m *MockCatalog) ListEvents(ctx context.Context, trackingPlanIds []string, page int) (*catalog.EventListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &catalog.EventListResponse{
		Data:        m.events,
		Total:       len(m.events),
		CurrentPage: page,
		PageSize:    10,
	}, nil
}

func (m *MockCatalog) ListProperties(ctx context.Context, trackingPlanIds []string, page int) (*catalog.PropertyListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &catalog.PropertyListResponse{
		Data:        m.properties,
		Total:       len(m.properties),
		CurrentPage: page,
		PageSize:    10,
	}, nil
}

func (m *MockCatalog) ListCustomTypes(ctx context.Context, page int) (*catalog.CustomTypeListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &catalog.CustomTypeListResponse{
		Data:        m.customTypes,
		Total:       len(m.customTypes),
		CurrentPage: page,
		PageSize:    10,
	}, nil
}

func (m *MockCatalog) ListCategories(ctx context.Context, page int) (*catalog.CategoryListResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &catalog.CategoryListResponse{
		Data:        m.categories,
		Total:       len(m.categories),
		CurrentPage: page,
		PageSize:    10,
	}, nil
}

// Stub implementations for other catalog.DataCatalog interface methods
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
func (m *MockCatalog) GetTrackingPlanEventSchema(ctx context.Context, id string, eventId string) (*catalog.TrackingPlanEventSchema, error) {
	return nil, nil
}
func (m *MockCatalog) GetTrackingPlanEventWithIdentifiers(ctx context.Context, id, eventId string) (*catalog.TrackingPlanEventPropertyIdentifiers, error) {
	return nil, nil
}
func (m *MockCatalog) UpdateTrackingPlanEvent(ctx context.Context, id string, input catalog.EventIdentifierDetail) (*catalog.TrackingPlan, error) {
	return nil, nil
}
func (m *MockCatalog) ListTrackingPlansWithFilter(ctx context.Context, ids []string) ([]catalog.TrackingPlan, error) {
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

func TestDownloadTrackingPlans(t *testing.T) {
	ctx := context.Background()

	// Create temporary directory for test output
	tmpDir, err := os.MkdirTemp("", "tp-download-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test data
	now := time.Now()
	trackingPlans := []catalog.TrackingPlan{
		{
			ID:          "tp-1",
			Name:        "E-commerce Plan",
			Description: strPtr("Main tracking plan"),
			Version:     1,
			WorkspaceID: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	events := []catalog.Event{
		{
			ID:          "event-1",
			Name:        "Page View",
			Description: "User viewed a page",
			EventType:   "track",
			WorkspaceId: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	properties := []catalog.Property{
		{
			ID:          "prop-1",
			Name:        "page_url",
			Description: "URL of the page",
			Type:        "string",
			WorkspaceId: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	customTypes := []catalog.CustomType{
		{
			ID:          "ct-1",
			Name:        "Custom String",
			Description: "Custom string type",
			Type:        "string",
			WorkspaceId: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	categories := []catalog.Category{
		{
			ID:          "cat-1",
			Name:        "User Actions",
			WorkspaceID: "ws-1",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	mockCatalog := &MockCatalog{
		trackingPlans: trackingPlans,
		events:        events,
		properties:    properties,
		customTypes:   customTypes,
		categories:    categories,
	}

	// Create downloader
	downloader := download.NewTrackingPlanDownloader(mockCatalog, tmpDir)

	// Test download
	err = downloader.Download(ctx)
	require.NoError(t, err)

	// Verify files were created
	jsonDir := filepath.Join(tmpDir, "json")

	// Check tracking-plans.json
	tpFile := filepath.Join(jsonDir, "tracking-plans.json")
	assert.FileExists(t, tpFile)

	var downloadedPlans []catalog.TrackingPlanWithIdentifiers
	data, err := os.ReadFile(tpFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &downloadedPlans)
	require.NoError(t, err)
	assert.Len(t, downloadedPlans, 1)
	assert.Equal(t, "tp-1", downloadedPlans[0].ID)
	assert.Equal(t, "E-commerce Plan", downloadedPlans[0].Name)
	// Verify events are included
	assert.Len(t, downloadedPlans[0].Events, 1)
	assert.Equal(t, "event-tp-1", downloadedPlans[0].Events[0].ID)
	assert.Equal(t, "Test Event for E-commerce Plan", downloadedPlans[0].Events[0].Name)

	// Check events.json
	eventsFile := filepath.Join(jsonDir, "events.json")
	assert.FileExists(t, eventsFile)

	var downloadedEvents []catalog.Event
	data, err = os.ReadFile(eventsFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &downloadedEvents)
	require.NoError(t, err)
	assert.Len(t, downloadedEvents, 1)
	assert.Equal(t, "event-1", downloadedEvents[0].ID)

	// Check properties.json
	propsFile := filepath.Join(jsonDir, "properties.json")
	assert.FileExists(t, propsFile)

	var downloadedProps []catalog.Property
	data, err = os.ReadFile(propsFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &downloadedProps)
	require.NoError(t, err)
	assert.Len(t, downloadedProps, 1)
	assert.Equal(t, "prop-1", downloadedProps[0].ID)

	// Check custom-types.json
	ctFile := filepath.Join(jsonDir, "custom-types.json")
	assert.FileExists(t, ctFile)

	var downloadedCTs []catalog.CustomType
	data, err = os.ReadFile(ctFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &downloadedCTs)
	require.NoError(t, err)
	assert.Len(t, downloadedCTs, 1)
	assert.Equal(t, "ct-1", downloadedCTs[0].ID)

	// Check categories.json
	catFile := filepath.Join(jsonDir, "categories.json")
	assert.FileExists(t, catFile)

	var downloadedCats []catalog.Category
	data, err = os.ReadFile(catFile)
	require.NoError(t, err)
	err = json.Unmarshal(data, &downloadedCats)
	require.NoError(t, err)
	assert.Len(t, downloadedCats, 1)
	assert.Equal(t, "cat-1", downloadedCats[0].ID)
}

func TestDownloadTrackingPlansError(t *testing.T) {
	ctx := context.Background()

	// Create temporary directory for test output
	tmpDir, err := os.MkdirTemp("", "tp-download-test-error")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	mockCatalog := &MockCatalog{
		err: assert.AnError,
	}

	downloader := download.NewTrackingPlanDownloader(mockCatalog, tmpDir)

	// Test download with error
	err = downloader.Download(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download tracking plans")
}

func strPtr(s string) *string {
	return &s
}