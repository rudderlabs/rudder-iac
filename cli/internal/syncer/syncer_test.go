package syncer_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestSyncerCreate(t *testing.T) {
	event := testutils.NewMockEvent("event1", resources.ResourceData{
		"name":        "Test Event",
		"description": "This is a test event",
	})

	property := testutils.NewMockProperty("property1", resources.ResourceData{
		"name":        "Test Property",
		"description": "This is a test property",
	})

	trackingPlan := testutils.NewMockTrackingPlan("trackingPlan1", resources.ResourceData{
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

	stateManager := &testutils.MemoryStateManager{}
	stateManager.Save(context.Background(), state.EmptyState())
	provider := &testutils.DataCatalogProvider{}

	syncer := syncer.New(provider, stateManager)
	err := syncer.Sync(context.Background(), targetGraph)
	assert.Nil(t, err)

	outputState, _ := stateManager.Load(context.Background())
	assert.NotNil(t, outputState)

	assert.Equal(t, 3, len(outputState.Resources))

	assert.NotNil(t, outputState.GetResource(event.URN()))
	assert.Equal(t, &state.StateResource{
		ID:    event.ID(),
		Type:  event.Type(),
		Input: event.Data(),
		Output: resources.ResourceData{
			"id":          "generated-event-event1",
			"name":        "Test Event",
			"description": "This is a test event",
			"operation":   "create",
		},
	}, outputState.GetResource(event.URN()))

	assert.NotNil(t, outputState.GetResource(property.URN()))
	assert.Equal(t, &state.StateResource{
		ID:    property.ID(),
		Type:  property.Type(),
		Input: property.Data(),
		Output: resources.ResourceData{
			"id":          "generated-property-property1",
			"name":        "Test Property",
			"description": "This is a test property",
			"operation":   "create",
		},
	}, outputState.GetResource(property.URN()))

	assert.NotNil(t, outputState.GetResource(trackingPlan.URN()))
	assert.Equal(t, &state.StateResource{
		ID:    trackingPlan.ID(),
		Type:  trackingPlan.Type(),
		Input: trackingPlan.Data(),
		Output: resources.ResourceData{
			"id":          "generated-tracking_plan-trackingPlan1",
			"name":        "Test Tracking Plan",
			"description": "This is a test tracking plan",
			"event_id":    "generated-event-event1",
			"rules": []interface{}{
				map[string]interface{}{
					"event":    "generated-event-event1",
					"property": "generated-property-property1",
				},
			},
			"operation": "create",
		},
	}, outputState.GetResource(trackingPlan.URN()))

	assert.ElementsMatch(t, []string{
		"create event event1",
		"create property property1",
		"create tracking_plan trackingPlan1",
	}, provider.OperationLog)

	savedStates := statesFromSavedLog(stateManager.SaveLog)
	assert.Equal(t, 4, len(savedStates)) // Initial state + 3 saves, one for each resource

	// each save adds one resource, last log entry should be the final state
	assert.Equal(t, 1, len(savedStates[1].Resources))
	assert.Equal(t, 2, len(savedStates[2].Resources))
	assert.Equal(t, outputState, savedStates[3])
}

func TestSyncerDelete(t *testing.T) {
	event := testutils.NewMockEvent("event1", resources.ResourceData{
		"name":        "Test Event",
		"description": "This is a test event",
	})

	property := testutils.NewMockProperty("property1", resources.ResourceData{
		"name":        "Test Property",
		"description": "This is a test property",
	})

	trackingPlan := testutils.NewMockTrackingPlan("trackingPlan1", resources.ResourceData{
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
	initialState.AddResource(&state.StateResource{
		ID:    event.ID(),
		Type:  event.Type(),
		Input: event.Data(),
		Output: resources.ResourceData{
			"id":          "generated-event-event1",
			"name":        "Test Event",
			"description": "This is a test event",
		},
	})
	initialState.AddResource(&state.StateResource{
		ID:    property.ID(),
		Type:  property.Type(),
		Input: property.Data(),
		Output: resources.ResourceData{
			"id":          "generated-property-property1",
			"name":        "Test Property",
			"description": "This is a test property",
		},
	})

	initialState.AddResource(&state.StateResource{
		ID:    trackingPlan.ID(),
		Type:  trackingPlan.Type(),
		Input: trackingPlan.Data(),
		Output: resources.ResourceData{
			"id":          "generated-tracking_plan-trackingPlan1",
			"name":        "Test Tracking Plan",
			"description": "This is a test tracking plan",
			"event_id":    "generated-event-event1",
			"rules": []interface{}{
				map[string]interface{}{
					"event":    "generated-event-event1",
					"property": "generated-property-property1",
				},
			},
		},
	})

	stateManager := &testutils.MemoryStateManager{}
	stateManager.Save(context.Background(), initialState)
	provider := &testutils.DataCatalogProvider{}

	// Create empty target graph (all resources should be deleted)
	targetGraph := resources.NewGraph()

	syncer := syncer.New(provider, stateManager)
	err := syncer.Sync(context.Background(), targetGraph)
	assert.Nil(t, err)

	outputState, _ := stateManager.Load(context.Background())
	assert.NotNil(t, outputState)

	// Verify all resources were deleted
	assert.Equal(t, 0, len(outputState.Resources))
	assert.Nil(t, outputState.GetResource(trackingPlan.URN()))
	assert.Nil(t, outputState.GetResource(event.URN()))
	assert.Nil(t, outputState.GetResource(property.URN()))

	assert.Equal(t, []string{
		"delete tracking_plan trackingPlan1",
		"delete property property1",
		"delete event event1",
	}, provider.OperationLog)

	savedStates := statesFromSavedLog(stateManager.SaveLog)
	assert.Equal(t, 4, len(savedStates)) // Initial state + 3 saves, one for each resource

	// each save removes one resource, last log entry should be the final state
	assert.Equal(t, 2, len(savedStates[1].Resources))
	assert.Equal(t, 1, len(savedStates[2].Resources))
	assert.Equal(t, outputState, savedStates[3])
}

func statesFromSavedLog(log []json.RawMessage) []*state.State {
	states := make([]*state.State, 0, len(log))
	for _, entry := range log {
		s, _ := state.FromJSON(entry)
		states = append(states, s)
	}
	return states
}
