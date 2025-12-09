package syncer_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
	internalTestutils "github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockWorkspace() *client.Workspace {
	return &client.Workspace{
		ID:          "test-workspace-id",
		Name:        "Test Workspace",
		Environment: "DEVELOPMENT",
		Status:      "ACTIVE",
		Region:      "US",
	}
}

func TestSyncerCreate(t *testing.T) {
	event := internalTestutils.NewMockEvent("event1", resources.ResourceData{
		"name":        "Test Event",
		"description": "This is a test event",
	})

	property := internalTestutils.NewMockProperty("property1", resources.ResourceData{
		"name":        "Test Property",
		"description": "This is a test property",
	})

	trackingPlan := internalTestutils.NewMockTrackingPlan("trackingPlan1", resources.ResourceData{
		"name":        "Test Tracking Plan",
		"description": "This is a test tracking plan",
		"event_id":    resources.PropertyRef{URN: event.URN(), Property: "id"},
		"rules": []interface{}{
			map[string]interface{}{
				"event":    resources.PropertyRef{URN: event.URN(), Property: "id"},
				"property": resources.PropertyRef{URN: property.URN(), Property: "id"},
			},
		},
	})

	targetGraph := resources.NewGraph()
	targetGraph.AddResource(event)
	targetGraph.AddResource(property)
	targetGraph.AddResource(trackingPlan)

	provider := &internalTestutils.DataCatalogProvider{
		InitialState:       state.EmptyState(),
		ReconstructedState: state.EmptyState(),
	}

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(provider, mockWorkspace(), syncer.WithReporter(mockReporter))
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	assert.Nil(t, err)

	// Verify provider operations
	assert.Len(t, provider.OperationLog, 3)
	assert.ElementsMatch(t, []internalTestutils.OperationLogEntry{
		{Operation: "Create", Args: []interface{}{event.ID(), event.Type(), event.Data()}},
		{Operation: "Create", Args: []interface{}{property.ID(), property.Type(), property.Data()}},
		{Operation: "Create", Args: []interface{}{trackingPlan.ID(), trackingPlan.Type(), resources.ResourceData{
			"name":        "Test Tracking Plan",
			"description": "This is a test tracking plan",
			"event_id":    "generated-event-event1",
			"rules": []interface{}{
				map[string]interface{}{
					"event":    "generated-event-event1",
					"property": "generated-property-property1",
				},
			},
		}}},
	}, provider.OperationLog)

	// Verify reporter calls
	assert.Len(t, mockReporter.ReportPlanCalls, 1, "ReportPlan should be called once")
	assert.Equal(t, 0, mockReporter.AskConfirmationCalls, "AskConfirmation should not be called in non-interactive mode")
	assert.Len(t, mockReporter.SyncStartedCalls, 1, "SyncStarted should be called once")
	assert.Equal(t, 3, mockReporter.SyncStartedCalls[0], "Should report 3 total tasks")
	assert.Equal(t, 1, mockReporter.SyncCompletedCalls, "SyncCompleted should be called once")
	assert.Len(t, mockReporter.TaskStartedCalls, 3, "TaskStarted should be called 3 times")
	assert.Len(t, mockReporter.TaskCompletedCalls, 3, "TaskCompleted should be called 3 times")

	expectedURNs := []string{
		event.URN(),
		property.URN(),
		trackingPlan.URN(),
	}

	for _, urn := range expectedURNs {
		assert.Contains(t, mockReporter.TaskStartedCalls, testutils.TaskCall{
			TaskID:      urn,
			Description: "Create " + urn,
		}, "TaskStarted should contain creation task for "+urn)

		assert.Contains(t, mockReporter.TaskCompletedCalls, testutils.TaskCompletionCall{
			TaskID:      urn,
			Description: "Create " + urn,
			Err:         nil,
		}, "TaskCompleted should contain creation task for "+urn)
	}
}

func TestSyncerDelete(t *testing.T) {
	event := internalTestutils.NewMockEvent("event1", resources.ResourceData{
		"name":        "Test Event",
		"description": "This is a test event",
	})

	property := internalTestutils.NewMockProperty("property1", resources.ResourceData{
		"name":        "Test Property",
		"description": "This is a test property",
	})

	trackingPlan := internalTestutils.NewMockTrackingPlan("trackingPlan1", resources.ResourceData{
		"name":        "Test Tracking Plan",
		"description": "This is a test tracking plan",
		"event_id":    resources.PropertyRef{URN: event.URN(), Property: "id"},
		"rules": []interface{}{
			map[string]interface{}{
				"event":    resources.PropertyRef{URN: event.URN(), Property: "id"},
				"property": resources.PropertyRef{URN: property.URN(), Property: "id"},
			},
		},
	})

	// Create initial state with resources
	initialState := state.EmptyState()
	initialState.AddResource(&state.ResourceState{
		ID:    event.ID(),
		Type:  event.Type(),
		Input: event.Data(),
		Output: resources.ResourceData{
			"id": "generated-event-event1",
		},
	})
	initialState.AddResource(&state.ResourceState{
		ID:    property.ID(),
		Type:  property.Type(),
		Input: property.Data(),
		Output: resources.ResourceData{
			"id": "generated-property-property1",
		},
	})

	initialState.AddResource(&state.ResourceState{
		ID:    trackingPlan.ID(),
		Type:  trackingPlan.Type(),
		Input: trackingPlan.Data(),
		Output: resources.ResourceData{
			"id": "generated-tracking-plan-trackingPlan1",
		},
	})

	provider := &internalTestutils.DataCatalogProvider{
		InitialState:       initialState,
		ReconstructedState: initialState,
	}

	// Create empty target graph (all resources should be deleted)
	targetGraph := resources.NewGraph()

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(provider, mockWorkspace(), syncer.WithReporter(mockReporter))
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	assert.Nil(t, err)

	// Verify provider operations
	assert.Len(t, provider.OperationLog, 3)
	assert.ElementsMatch(t, []internalTestutils.OperationLogEntry{
		{Operation: "Delete", Args: []interface{}{trackingPlan.ID(), trackingPlan.Type(), resources.ResourceData{
			"name":        "Test Tracking Plan",
			"description": "This is a test tracking plan",
			"event_id":    resources.PropertyRef{URN: event.URN(), Property: "id"},
			"rules": []interface{}{
				map[string]interface{}{
					"event":    resources.PropertyRef{URN: event.URN(), Property: "id"},
					"property": resources.PropertyRef{URN: property.URN(), Property: "id"},
				},
			},
			"id": "generated-tracking-plan-trackingPlan1",
		}}},
		{Operation: "Delete", Args: []interface{}{property.ID(), property.Type(), resources.ResourceData{
			"name":        "Test Property",
			"description": "This is a test property",
			"id":          "generated-property-property1",
		}}},
		{Operation: "Delete", Args: []interface{}{event.ID(), event.Type(), resources.ResourceData{
			"name":        "Test Event",
			"description": "This is a test event",
			"id":          "generated-event-event1",
		}}},
	}, provider.OperationLog)

	// Verify reporter calls
	assert.Len(t, mockReporter.ReportPlanCalls, 1, "ReportPlan should be called once")
	assert.Equal(t, 0, mockReporter.AskConfirmationCalls, "AskConfirmation should not be called in non-interactive mode")
	assert.Len(t, mockReporter.SyncStartedCalls, 1, "SyncStarted should be called once")
	assert.Equal(t, 3, mockReporter.SyncStartedCalls[0], "Should report 3 total tasks for deletion")
	assert.Equal(t, 1, mockReporter.SyncCompletedCalls, "SyncCompleted should be called once")
	assert.Len(t, mockReporter.TaskStartedCalls, 3, "TaskStarted should be called 3 times")
	assert.Len(t, mockReporter.TaskCompletedCalls, 3, "TaskCompleted should be called 3 times")

	expectedURNs := []string{
		event.URN(),
		property.URN(),
		trackingPlan.URN(),
	}

	for _, urn := range expectedURNs {
		assert.Contains(t, mockReporter.TaskStartedCalls, testutils.TaskCall{
			TaskID:      urn,
			Description: "Delete " + urn,
		}, "TaskStarted should contain deletion task for "+urn)

		assert.Contains(t, mockReporter.TaskCompletedCalls, testutils.TaskCompletionCall{
			TaskID:      urn,
			Description: "Delete " + urn,
			Err:         nil,
		}, "TaskCompleted should contain deletion task for "+urn)
	}
}
