package syncer_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/testutils"
	internalTestutils "github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enableConcurrentSyncs(t *testing.T) {
	t.Helper()

	viper.Set("experimental", true)
	viper.Set("flags.concurrentSyncs", true)

	t.Cleanup(func() {
		viper.Set("experimental", false)
		viper.Set("flags.concurrentSyncs", false)
	})
}

func TestSyncerConcurrencyCreate(t *testing.T) {
	enableConcurrentSyncs(t)

	events, properties := createBasicResources()
	trackingPlans := createTrackingPlans(events, properties)
	targetGraph := createGraphWithResources(events, properties, trackingPlans)

	// Create provider with empty initial state
	provider := &internalTestutils.DataCatalogProvider{
		InitialState:       state.EmptyState(),
		ReconstructedState: state.EmptyState(),
	}

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(provider, mockWorkspace(), syncer.WithConcurrency(3), syncer.WithReporter(mockReporter))
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	require.NoError(t, err)

	// We expect 6 creates
	assert.Len(t, provider.OperationLog, 6)
	createOps := make([]internalTestutils.OperationLogEntry, 0)
	putStateOps := make([]internalTestutils.OperationLogEntry, 0)

	for _, op := range provider.OperationLog {
		switch op.Operation {
		case "Create":
			createOps = append(createOps, op)
		case "PutResourceState":
			putStateOps = append(putStateOps, op)
		}
	}
	assert.Len(t, createOps, 6)
	assert.Len(t, putStateOps, 0)

	eventCreateCount := 0
	propertyCreateCount := 0
	trackingPlanCreateCount := 0

	for _, op := range createOps {
		resourceType := op.Args[1].(string)
		switch resourceType {
		case "event":
			eventCreateCount++
		case "property":
			propertyCreateCount++
		case "tracking-plan":
			trackingPlanCreateCount++
		}
	}

	assert.Equal(t, 2, eventCreateCount)
	assert.Equal(t, 2, propertyCreateCount)
	assert.Equal(t, 2, trackingPlanCreateCount)

	// Verify that all resources were created successfully
	expectedResourceIds := []string{"event1", "event2", "property1", "property2", "trackingPlan1", "trackingPlan2"}
	actualResourceIds := make([]string, 0)

	for _, op := range createOps {
		if op.Operation == "Create" {
			actualResourceIds = append(actualResourceIds, op.Args[0].(string))
		}
	}

	assert.ElementsMatch(t, expectedResourceIds, actualResourceIds)

	// Verify dependency order: each tracking plan should be created after its specific dependencies
	// trackingPlan1 depends on event1 and property1
	// trackingPlan2 depends on event2 and property2

	// Find positions of each resource type
	event1Pos := -1
	event2Pos := -1
	property1Pos := -1
	property2Pos := -1
	trackingPlan1Pos := -1
	trackingPlan2Pos := -1

	for i, op := range createOps {
		resourceID := op.Args[0].(string)
		switch resourceID {
		case "event1":
			event1Pos = i
		case "event2":
			event2Pos = i
		case "property1":
			property1Pos = i
		case "property2":
			property2Pos = i
		case "trackingPlan1":
			trackingPlan1Pos = i
		case "trackingPlan2":
			trackingPlan2Pos = i
		}
	}

	assert.Less(t, event1Pos, trackingPlan1Pos, "event1 should be created before trackingPlan1")
	assert.Less(t, property1Pos, trackingPlan1Pos, "property1 should be created before trackingPlan1")
	assert.Less(t, event2Pos, trackingPlan2Pos, "event2 should be created before trackingPlan2")
	assert.Less(t, property2Pos, trackingPlan2Pos, "property2 should be created before trackingPlan2")

	// Verify reporter calls
	assert.Len(t, mockReporter.ReportPlanCalls, 1, "ReportPlan should be called once")
	assert.Len(t, mockReporter.SyncStartedCalls, 1, "SyncStarted should be called once")
	assert.Equal(t, 6, mockReporter.SyncStartedCalls[0], "Should report 6 total tasks")
	assert.Equal(t, 1, mockReporter.SyncCompletedCalls, "SyncCompleted should be called once")
	assert.Len(t, mockReporter.TaskStartedCalls, 6, "TaskStarted should be called 6 times for concurrent operations")
	assert.Len(t, mockReporter.TaskCompletedCalls, 6, "TaskCompleted should be called 6 times")

	expectedURNs := []string{
		events[0].URN(),
		events[1].URN(),
		properties[0].URN(),
		properties[1].URN(),
		trackingPlans[0].URN(),
		trackingPlans[1].URN(),
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

	// Verify all tasks completed successfully
	for _, taskCompletion := range mockReporter.TaskCompletedCalls {
		assert.Nil(t, taskCompletion.Err, "All concurrent tasks should complete without error")
	}
}

func TestSyncerConcurrencyDelete(t *testing.T) {
	enableConcurrentSyncs(t)

	events, properties := createBasicResources()
	trackingPlans := createTrackingPlans(events, properties)

	initialState := createInitialStateWithResources(events, properties, trackingPlans)
	reconstructedState := createInitialStateWithResources(events, properties, trackingPlans)

	// Create provider with initial state
	provider := &internalTestutils.DataCatalogProvider{
		InitialState:       initialState,
		ReconstructedState: reconstructedState,
	}

	// Create empty target graph (all resources should be deleted)
	targetGraph := resources.NewGraph()

	mockReporter := testutils.NewMockReporter()
	s, err := syncer.New(provider, mockWorkspace(), syncer.WithConcurrency(3), syncer.WithReporter(mockReporter))
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	require.NoError(t, err)

	// We expect 12 deletes
	assert.Len(t, provider.OperationLog, 6)
	deleteOps := make([]internalTestutils.OperationLogEntry, 0)

	for _, op := range provider.OperationLog {
		if op.Operation == "Delete" {
			deleteOps = append(deleteOps, op)
		}
	}
	assert.Len(t, deleteOps, 6)

	// Verify that tracking plans were deleted before events and properties
	// by checking the order of delete operations
	eventDeleteCount := 0
	propertyDeleteCount := 0
	trackingPlanDeleteCount := 0

	for _, op := range deleteOps {
		resourceType := op.Args[1].(string)
		switch resourceType {
		case "event":
			eventDeleteCount++
		case "property":
			propertyDeleteCount++
		case "tracking-plan":
			trackingPlanDeleteCount++
		}
	}

	assert.Equal(t, 2, eventDeleteCount)
	assert.Equal(t, 2, propertyDeleteCount)
	assert.Equal(t, 2, trackingPlanDeleteCount)

	// Verify that all resources were deleted successfully
	expectedResourceIds := []string{"event1", "event2", "property1", "property2", "trackingPlan1", "trackingPlan2"}
	actualResourceIds := make([]string, 0)

	for _, op := range deleteOps {
		if op.Operation == "Delete" {
			actualResourceIds = append(actualResourceIds, op.Args[0].(string))
		}
	}
	assert.ElementsMatch(t, expectedResourceIds, actualResourceIds)

	// Verify dependency order: each tracking plan should be deleted before its specific dependencies
	event1Pos := -1
	event2Pos := -1
	property1Pos := -1
	property2Pos := -1
	trackingPlan1Pos := -1
	trackingPlan2Pos := -1

	for i, op := range deleteOps {
		resourceID := op.Args[0].(string)
		switch resourceID {
		case "event1":
			event1Pos = i
		case "event2":
			event2Pos = i
		case "property1":
			property1Pos = i
		case "property2":
			property2Pos = i
		case "trackingPlan1":
			trackingPlan1Pos = i
		case "trackingPlan2":
			trackingPlan2Pos = i
		}
	}

	assert.Less(t, trackingPlan1Pos, event1Pos, "trackingPlan1 should be deleted before event1")
	assert.Less(t, trackingPlan1Pos, property1Pos, "trackingPlan1 should be deleted before property1")
	assert.Less(t, trackingPlan2Pos, event2Pos, "trackingPlan2 should be deleted before event2")
	assert.Less(t, trackingPlan2Pos, property2Pos, "trackingPlan2 should be deleted before property2")

	// Verify reporter calls
	assert.Len(t, mockReporter.ReportPlanCalls, 1, "ReportPlan should be called once")
	assert.Len(t, mockReporter.SyncStartedCalls, 1, "SyncStarted should be called once")
	assert.Equal(t, 6, mockReporter.SyncStartedCalls[0], "Should report 6 total deletion tasks")
	assert.Equal(t, 1, mockReporter.SyncCompletedCalls, "SyncCompleted should be called once")
	assert.Len(t, mockReporter.TaskStartedCalls, 6, "TaskStarted should be called 6 times for concurrent deletions")
	assert.Len(t, mockReporter.TaskCompletedCalls, 6, "TaskCompleted should be called 6 times")

	expectedURNs := []string{
		events[0].URN(),
		events[1].URN(),
		properties[0].URN(),
		properties[1].URN(),
		trackingPlans[0].URN(),
		trackingPlans[1].URN(),
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

	// Verify all deletion tasks completed successfully
	for _, taskCompletion := range mockReporter.TaskCompletedCalls {
		assert.Nil(t, taskCompletion.Err, "All concurrent deletion tasks should complete without error")
	}
}

func TestSyncerContinueOnFailBehavior(t *testing.T) {

	t.Run("sync operations stop on first failure", func(t *testing.T) {
		enableConcurrentSyncs(t)

		events, properties := createBasicResources()
		trackingPlans := createTrackingPlans(events, properties)

		provider := &internalTestutils.DataCatalogProvider{
			InitialState:       state.EmptyState(),
			ReconstructedState: state.EmptyState(),
		}

		// Create a custom provider that fails for event2
		failingProvider := &internalTestutils.FailingDataCatalogProvider{
			DataCatalogProvider: provider,
			FailingResources:    []string{"event2"},
		}

		mockReporter := testutils.NewMockReporter()
		s, err := syncer.New(failingProvider, mockWorkspace(), syncer.WithConcurrency(2), syncer.WithReporter(mockReporter))
		require.NoError(t, err)

		targetGraph := createGraphWithResources(events, properties, trackingPlans)
		err = s.Sync(context.Background(), targetGraph)
		assert.Error(t, err)

		// Verify error messages
		expectedError := "simulated failure for event2"
		assert.Contains(t, err.Error(), expectedError)

		// Expected operation count: 12 total operations in success case.
		// When event2 fails, its 2 operations + 2 tracking plan operations are skipped.
		// Therefore, we expect at most 8 operations to be executed.

		// FIXME: This assertion is not very helpful because if there is bug in the
		// continueOnFail logic, the assertion will still not fail.
		assert.LessOrEqual(t, len(failingProvider.OperationLog), 8, "Should have fewer operations due to early failure")

		// Verify reporter recorded the failure
		assert.Len(t, mockReporter.ReportPlanCalls, 1, "ReportPlan should be called once")
		assert.Len(t, mockReporter.SyncStartedCalls, 1, "SyncStarted should be called once")
		assert.Equal(t, 1, mockReporter.SyncCompletedCalls, "SyncCompleted should be called once even after failure")

		// Check that at least one task reported an error
		failureCount := 0
		for _, taskCompletion := range mockReporter.TaskCompletedCalls {
			if taskCompletion.Err != nil {
				failureCount++
			}
		}
		assert.Greater(t, failureCount, 0, "At least one task should have reported an error")
	})

	t.Run("destroy operations continue despite failures", func(t *testing.T) {
		enableConcurrentSyncs(t)

		events, properties := createBasicResources()
		trackingPlans := createTrackingPlans(events, properties)

		provider := &internalTestutils.DataCatalogProvider{
			InitialState:       createInitialStateWithResources(events, properties, trackingPlans),
			ReconstructedState: createInitialStateWithResources(events, properties, trackingPlans),
		}

		// Create a custom provider that fails for event2 and property2
		failingProvider := &internalTestutils.FailingDataCatalogProvider{
			DataCatalogProvider: provider,
			FailingResources:    []string{"event2", "property2"},
		}

		mockReporter := testutils.NewMockReporter()
		s, err := syncer.New(failingProvider, mockWorkspace(), syncer.WithConcurrency(2), syncer.WithReporter(mockReporter))
		require.NoError(t, err)

		errors := s.Destroy(context.Background())

		// Verify error count
		assert.Len(t, errors, 2, "Should have expected number of errors")

		// Verify operation count
		assert.Len(t, failingProvider.OperationLog, 4, "Should have attempted operations for successful resources despite failures")

		// Verify error messages
		expectedErrorContains := []string{
			"simulated delete failure for property2",
			"simulated delete failure for event2",
		}
		errorMessages := make([]string, len(errors))
		for i, err := range errors {
			errorMessages[i] = err.Error()
		}
		for _, expectedError := range expectedErrorContains {
			assert.Contains(t, errorMessages, expectedError)
		}

		// Verify reporter recorded failures but still completed sync
		assert.Len(t, mockReporter.ReportPlanCalls, 1, "ReportPlan should be called once")
		assert.Len(t, mockReporter.SyncStartedCalls, 1, "SyncStarted should be called once")
		assert.Equal(t, 1, mockReporter.SyncCompletedCalls, "SyncCompleted should be called once even with failures in destroy mode")

		// Check that failures were reported for the failing resources
		failureCount := 0
		for _, taskCompletion := range mockReporter.TaskCompletedCalls {
			if taskCompletion.Err != nil {
				failureCount++
			}
		}
		assert.Equal(t, 2, failureCount, "Should have reported 2 task failures")
	})
}

// Helper function
func createBasicResources() ([]*resources.Resource, []*resources.Resource) {
	events := []*resources.Resource{
		internalTestutils.NewMockEvent("event1", resources.ResourceData{
			"name": "Event 1",
		}),
		internalTestutils.NewMockEvent("event2", resources.ResourceData{
			"name": "Event 2",
		}),
	}

	properties := []*resources.Resource{
		internalTestutils.NewMockProperty("property1", resources.ResourceData{
			"name": "Property 1",
		}),
		internalTestutils.NewMockProperty("property2", resources.ResourceData{
			"name": "Property 2",
		}),
	}

	return events, properties
}

// Helper function
func createTrackingPlans(events []*resources.Resource, properties []*resources.Resource) []*resources.Resource {
	trackingPlans := []*resources.Resource{
		internalTestutils.NewMockTrackingPlan("trackingPlan1", resources.ResourceData{
			"name":        "Tracking Plan 1",
			"description": "First tracking plan",
			"event_id":    resources.PropertyRef{URN: events[0].URN(), Property: "id"},
			"rules": []interface{}{
				map[string]interface{}{
					"event":    resources.PropertyRef{URN: events[0].URN(), Property: "id"},
					"property": resources.PropertyRef{URN: properties[0].URN(), Property: "id"},
				},
			},
		}),
		internalTestutils.NewMockTrackingPlan("trackingPlan2", resources.ResourceData{
			"name":        "Tracking Plan 2",
			"description": "Second tracking plan",
			"event_id":    resources.PropertyRef{URN: events[1].URN(), Property: "id"},
			"rules": []interface{}{
				map[string]interface{}{
					"event":    resources.PropertyRef{URN: events[1].URN(), Property: "id"},
					"property": resources.PropertyRef{URN: properties[1].URN(), Property: "id"},
				},
			},
		}),
	}
	return trackingPlans
}

// Helper function
func createGraphWithResources(events []*resources.Resource, properties []*resources.Resource, trackingPlans []*resources.Resource) *resources.Graph {
	graph := resources.NewGraph()

	// Add all events to the graph
	for _, event := range events {
		graph.AddResource(event)
	}

	// Add all properties to the graph
	for _, property := range properties {
		graph.AddResource(property)
	}

	// Add all tracking plans to the graph
	for _, trackingPlan := range trackingPlans {
		graph.AddResource(trackingPlan)
	}

	return graph
}

// Helper function
func createInitialStateWithResources(events []*resources.Resource, properties []*resources.Resource, trackingPlans []*resources.Resource) *state.State {
	initialState := state.EmptyState()

	// Add all events to initial state
	for _, event := range events {
		initialState.AddResource(&state.ResourceState{
			ID:    event.ID(),
			Type:  event.Type(),
			Input: event.Data(),
			Output: resources.ResourceData{
				"id": fmt.Sprintf("generated-event-%s", event.ID()),
			},
		})
	}

	// Add all properties to initial state
	for _, property := range properties {
		initialState.AddResource(&state.ResourceState{
			ID:    property.ID(),
			Type:  property.Type(),
			Input: property.Data(),
			Output: resources.ResourceData{
				"id": fmt.Sprintf("generated-property-%s", property.ID()),
			},
		})
	}

	// Add all tracking plans to initial state
	for _, trackingPlan := range trackingPlans {
		initialState.AddResource(&state.ResourceState{
			ID:    trackingPlan.ID(),
			Type:  trackingPlan.Type(),
			Input: trackingPlan.Data(),
			Output: resources.ResourceData{
				"id": fmt.Sprintf("generated-tracking-plan-%s", trackingPlan.ID()),
			},
		})
	}

	return initialState
}
