package syncer_test

import (
	"context"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDataWithRef is a minimal RawData struct with a PropertyRef field
type mockDataWithRef struct {
	ID        string
	Name      string
	ParentRef *resources.PropertyRef
}

// mockParentData represents the parent resource being referenced
type mockParentData struct {
	ID   string
	Name string
}

// mockParentState is the output state for parent resources
type mockParentState struct {
	RemoteID string
}

// mockRawProvider captures data received by handlers to verify dereferencing
type mockRawProvider struct {
	provider.EmptyProvider
	capturedCreateData any
	capturedUpdateOld  any
	capturedUpdateNew  any
	capturedDeleteData any
	initialState       *state.State
}

func (p *mockRawProvider) SupportedTypes() []string {
	return []string{"mock-parent", "mock-child"}
}

func (p *mockRawProvider) LoadResourcesFromRemote(_ context.Context) (*resources.RemoteResources, error) {
	return &resources.RemoteResources{}, nil
}

func (p *mockRawProvider) MapRemoteToState(_ *resources.RemoteResources) (*state.State, error) {
	return p.initialState, nil
}

func (p *mockRawProvider) CreateRaw(_ context.Context, r *resources.Resource) (any, error) {
	p.capturedCreateData = r.RawData()

	// Return appropriate state based on resource type
	if r.Type() == "mock-parent" {
		return &mockParentState{RemoteID: "remote-parent-id"}, nil
	}
	return &mockDataWithRef{ID: "remote-child-id"}, nil
}

func (p *mockRawProvider) UpdateRaw(_ context.Context, r *resources.Resource, oldData any, oldState any) (any, error) {
	p.capturedUpdateNew = r.RawData()
	p.capturedUpdateOld = oldData
	return oldState, nil
}

func (p *mockRawProvider) DeleteRaw(_ context.Context, id string, resourceType string, oldData any, oldState any) error {
	p.capturedDeleteData = oldData
	return nil
}

// createParentRef creates a PropertyRef that resolves to parent's remote ID
func createParentRef(parentURN string) *resources.PropertyRef {
	return &resources.PropertyRef{
		URN: parentURN,
		Resolve: func(outputRaw any) (string, error) {
			typed, ok := outputRaw.(*mockParentState)
			if !ok {
				return "", assert.AnError
			}
			return typed.RemoteID, nil
		},
	}
}

func TestDereferenceOnCreate(t *testing.T) {
	// Create parent resource
	parent := resources.NewResource(
		"parent1",
		"mock-parent",
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&mockParentData{ID: "parent1", Name: "Parent"}),
	)

	// Create child resource with PropertyRef to parent
	child := resources.NewResource(
		"child1",
		"mock-child",
		resources.ResourceData{},
		[]string{parent.URN()},
		resources.WithRawData(&mockDataWithRef{
			ID:        "child1",
			Name:      "Child",
			ParentRef: createParentRef(parent.URN()),
		}),
	)

	targetGraph := resources.NewGraph()
	targetGraph.AddResource(parent)
	targetGraph.AddResource(child)

	mockProvider := &mockRawProvider{
		initialState: state.EmptyState(),
	}

	s, err := syncer.New(mockProvider, mockWorkspace())
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	require.NoError(t, err)

	// Verify PropertyRef.Value was populated during Create
	capturedChild, ok := mockProvider.capturedCreateData.(*mockDataWithRef)
	require.True(t, ok, "Expected captured data to be *mockDataWithRef")
	require.NotNil(t, capturedChild.ParentRef, "ParentRef should not be nil")
	assert.True(t, capturedChild.ParentRef.IsResolved, "PropertyRef should be resolved")
	assert.Equal(t, "remote-parent-id", capturedChild.ParentRef.Value, "PropertyRef.Value should be populated with parent remote ID")
}

func TestDereferenceOnUpdate(t *testing.T) {
	// Create parent resource
	parent := resources.NewResource(
		"parent1",
		"mock-parent",
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&mockParentData{ID: "parent1", Name: "Parent"}),
	)

	// Create initial state with existing child
	initialState := state.EmptyState()
	initialState.AddResource(&state.ResourceState{
		ID:   "child1",
		Type: "mock-child",
		InputRaw: &mockDataWithRef{
			ID:        "child1",
			Name:      "Old Child",
			ParentRef: createParentRef(parent.URN()),
		},
		OutputRaw: &mockDataWithRef{ID: "remote-child-id"},
	})
	initialState.AddResource(&state.ResourceState{
		ID:        "parent1",
		Type:      "mock-parent",
		InputRaw:  &mockParentData{ID: "parent1", Name: "Parent"},
		OutputRaw: &mockParentState{RemoteID: "remote-parent-id"},
	})

	// Create updated child resource
	updatedChild := resources.NewResource(
		"child1",
		"mock-child",
		resources.ResourceData{},
		[]string{parent.URN()},
		resources.WithRawData(&mockDataWithRef{
			ID:        "child1",
			Name:      "Updated Child",
			ParentRef: createParentRef(parent.URN()),
		}),
	)

	targetGraph := resources.NewGraph()
	targetGraph.AddResource(parent)
	targetGraph.AddResource(updatedChild)

	mockProvider := &mockRawProvider{
		initialState: initialState,
	}

	s, err := syncer.New(mockProvider, mockWorkspace())
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	require.NoError(t, err)

	// Verify PropertyRef.Value was populated in NEW data
	capturedNew, ok := mockProvider.capturedUpdateNew.(*mockDataWithRef)
	require.True(t, ok, "Expected new data to be *mockDataWithRef")
	require.NotNil(t, capturedNew.ParentRef, "New data ParentRef should not be nil")
	assert.True(t, capturedNew.ParentRef.IsResolved, "New data PropertyRef should be resolved")
	assert.Equal(t, "remote-parent-id", capturedNew.ParentRef.Value, "New data PropertyRef.Value should be populated")

	// Verify PropertyRef.Value was populated in OLD data (THIS IS THE BUG - will fail without fix)
	capturedOld, ok := mockProvider.capturedUpdateOld.(*mockDataWithRef)
	require.True(t, ok, "Expected old data to be *mockDataWithRef")
	require.NotNil(t, capturedOld.ParentRef, "Old data ParentRef should not be nil")
	assert.True(t, capturedOld.ParentRef.IsResolved, "Old data PropertyRef should be resolved")
	assert.Equal(t, "remote-parent-id", capturedOld.ParentRef.Value, "Old data PropertyRef.Value should be populated (THIS IS THE BUG)")
}

func TestDereferenceOnDelete(t *testing.T) {
	// Create parent resource for state
	parent := resources.NewResource(
		"parent1",
		"mock-parent",
		resources.ResourceData{},
		[]string{},
		resources.WithRawData(&mockParentData{ID: "parent1", Name: "Parent"}),
	)

	// Create initial state with existing child
	initialState := state.EmptyState()
	initialState.AddResource(&state.ResourceState{
		ID:   "child1",
		Type: "mock-child",
		InputRaw: &mockDataWithRef{
			ID:        "child1",
			Name:      "Child",
			ParentRef: createParentRef(parent.URN()),
		},
		OutputRaw: &mockDataWithRef{ID: "remote-child-id"},
	})
	initialState.AddResource(&state.ResourceState{
		ID:        "parent1",
		Type:      "mock-parent",
		InputRaw:  &mockParentData{ID: "parent1", Name: "Parent"},
		OutputRaw: &mockParentState{RemoteID: "remote-parent-id"},
	})

	// Create target graph with only parent (will trigger child delete)
	targetGraph := resources.NewGraph()
	targetGraph.AddResource(parent)

	mockProvider := &mockRawProvider{
		initialState: initialState,
	}

	s, err := syncer.New(mockProvider, mockWorkspace())
	require.NoError(t, err)

	err = s.Sync(context.Background(), targetGraph)
	require.NoError(t, err)

	// Verify PropertyRef.Value was populated in oldData during Delete (THIS IS THE BUG - will fail without fix)
	capturedOld, ok := mockProvider.capturedDeleteData.(*mockDataWithRef)
	if !ok {
		t.Logf("capturedDeleteData type: %T, value: %+v", mockProvider.capturedDeleteData, mockProvider.capturedDeleteData)
	}
	require.True(t, ok, "Expected delete data to be *mockDataWithRef")
	require.NotNil(t, capturedOld.ParentRef, "Delete data ParentRef should not be nil")
	assert.True(t, capturedOld.ParentRef.IsResolved, "Delete data PropertyRef should be resolved")
	assert.Equal(t, "remote-parent-id", capturedOld.ParentRef.Value, "Delete data PropertyRef.Value should be populated (THIS IS THE BUG)")
}
