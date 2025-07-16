package syncer_test

import (
	"context"
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

	provider := &testutils.DataCatalogProvider{
		InitialState: state.EmptyState(),
	}

	s, _ := syncer.New(provider)
	err := s.Sync(context.Background(), targetGraph, syncer.SyncOptions{})
	assert.Nil(t, err)

	assert.Len(t, provider.OperationLog, 6)
	assert.ElementsMatch(t, []testutils.OperationLogEntry{
		{Operation: "Create", Args: []interface{}{event.ID(), event.Type(), event.Data()}},
		{Operation: "PutResourceState", Args: []interface{}{event.URN(), &state.ResourceState{
			ID:    event.ID(),
			Type:  event.Type(),
			Input: event.Data(),
			Output: resources.ResourceData{
				"id": "generated-event-event1",
			},
			Dependencies: []string{},
		}}},
		{Operation: "Create", Args: []interface{}{property.ID(), property.Type(), property.Data()}},
		{Operation: "PutResourceState", Args: []interface{}{property.URN(), &state.ResourceState{
			ID:    property.ID(),
			Type:  property.Type(),
			Input: property.Data(),
			Output: resources.ResourceData{
				"id": "generated-property-property1",
			},
			Dependencies: []string{},
		}}},
		{Operation: "Create", Args: []interface{}{trackingPlan.ID(), trackingPlan.Type(), resources.ResourceData{
			"name":        "Test Tracking Plan",
			"description": "This is a test tracking plan",
			"event_id":    resources.PropertyRef{URN: event.URN(), Property: "id", ResolvedValue: "generated-event-event1"},
			"rules": []interface{}{
				map[string]interface{}{
					"event":    resources.PropertyRef{URN: event.URN(), Property: "id", ResolvedValue: "generated-event-event1"},
					"property": resources.PropertyRef{URN: property.URN(), Property: "id", ResolvedValue: "generated-property-property1"},
				},
			},
		}}},
		{Operation: "PutResourceState", Args: []interface{}{trackingPlan.URN(), &state.ResourceState{
			ID:    trackingPlan.ID(),
			Type:  trackingPlan.Type(),
			Input: trackingPlan.Data(),
			Output: resources.ResourceData{
				"id": "generated-tracking-plan-trackingPlan1",
			},
			Dependencies: []string{},
		}}},
	}, provider.OperationLog)
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

	provider := &testutils.DataCatalogProvider{
		InitialState: initialState,
	}

	// Create empty target graph (all resources should be deleted)
	targetGraph := resources.NewGraph()

	s, _ := syncer.New(provider)
	err := s.Sync(context.Background(), targetGraph, syncer.SyncOptions{})
	assert.Nil(t, err)

	assert.Len(t, provider.OperationLog, 6)
	assert.ElementsMatch(t, []testutils.OperationLogEntry{
		{Operation: "DeleteResourceState", Args: []interface{}{&state.ResourceState{
			ID:    trackingPlan.ID(),
			Type:  trackingPlan.Type(),
			Input: trackingPlan.Data(),
			Output: resources.ResourceData{
				"id": "generated-tracking-plan-trackingPlan1",
			}}}},
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
		{Operation: "DeleteResourceState", Args: []interface{}{&state.ResourceState{
			ID:    property.ID(),
			Type:  property.Type(),
			Input: property.Data(),
			Output: resources.ResourceData{
				"id": "generated-property-property1",
			}}}},
		{Operation: "Delete", Args: []interface{}{property.ID(), property.Type(), resources.ResourceData{
			"name":        "Test Property",
			"description": "This is a test property",
			"id":          "generated-property-property1",
		}}},
		{Operation: "DeleteResourceState", Args: []interface{}{&state.ResourceState{
			ID:    event.ID(),
			Type:  event.Type(),
			Input: event.Data(),
			Output: resources.ResourceData{
				"id": "generated-event-event1",
			}}}},
		{Operation: "Delete", Args: []interface{}{event.ID(), event.Type(), resources.ResourceData{
			"name":        "Test Event",
			"description": "This is a test event",
			"id":          "generated-event-event1",
		}}},
	}, provider.OperationLog)
}
