package providers

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/state"
)

// mockProvider is a mock implementation of the project.Provider interface for testing.
type mockProvider struct {
	supportedKinds         []string
	supportedTypes         []string
	validateErr            error
	loadSpecErr            error
	getResourceGraphVal    *resources.Graph
	getResourceGraphErr    error
	loadStateVal           *state.State
	loadStateErr           error
	putResourceStateErr    error
	deleteResourceStateErr error
	createVal              *resources.ResourceData
	createErr              error
	updateVal              *resources.ResourceData
	updateErr              error
	deleteErr              error

	// Tracking calls
	validateCalled     bool
	loadSpecCalledWith struct {
		path string
		spec *specs.Spec
	}
	getResourceGraphCalled     bool
	loadStateCalled            bool
	putResourceStateCalledWith struct {
		urn   string
		state *state.ResourceState
	}
	deleteResourceStateCalledWith *state.ResourceState
	createCalledWith              struct {
		id           string
		resourceType string
		data         resources.ResourceData
	}
	updateCalledWith struct {
		id           string
		resourceType string
		data         resources.ResourceData
		state        resources.ResourceData
	}
	deleteCalledWith struct {
		id           string
		resourceType string
		state        resources.ResourceData
	}
}

func (m *mockProvider) GetSupportedKinds() []string {
	return m.supportedKinds
}

func (m *mockProvider) GetSupportedTypes() []string {
	return m.supportedTypes
}

func (m *mockProvider) Validate() error {
	m.validateCalled = true
	return m.validateErr
}

func (m *mockProvider) LoadSpec(path string, s *specs.Spec) error {
	m.loadSpecCalledWith.path = path
	m.loadSpecCalledWith.spec = s
	return m.loadSpecErr
}

func (m *mockProvider) GetResourceGraph() (*resources.Graph, error) {
	m.getResourceGraphCalled = true
	return m.getResourceGraphVal, m.getResourceGraphErr
}

func (m *mockProvider) LoadState(ctx context.Context) (*state.State, error) {
	m.loadStateCalled = true
	return m.loadStateVal, m.loadStateErr
}

func (m *mockProvider) PutResourceState(ctx context.Context, URN string, s *state.ResourceState) error {
	m.putResourceStateCalledWith.urn = URN
	m.putResourceStateCalledWith.state = s
	return m.putResourceStateErr
}

func (m *mockProvider) DeleteResourceState(ctx context.Context, s *state.ResourceState) error {
	m.deleteResourceStateCalledWith = s
	return m.deleteResourceStateErr
}

func (m *mockProvider) Create(ctx context.Context, ID string, resourceType string, data resources.ResourceData) (*resources.ResourceData, error) {
	m.createCalledWith.id = ID
	m.createCalledWith.resourceType = resourceType
	m.createCalledWith.data = data
	return m.createVal, m.createErr
}

func (m *mockProvider) Update(ctx context.Context, ID string, resourceType string, data resources.ResourceData, s resources.ResourceData) (*resources.ResourceData, error) {
	m.updateCalledWith.id = ID
	m.updateCalledWith.resourceType = resourceType
	m.updateCalledWith.data = data
	m.updateCalledWith.state = s
	return m.updateVal, m.updateErr
}

func (m *mockProvider) Delete(ctx context.Context, ID string, resourceType string, s resources.ResourceData) error {
	m.deleteCalledWith.id = ID
	m.deleteCalledWith.resourceType = resourceType
	m.deleteCalledWith.state = s
	return m.deleteErr
}

func TestNewCompositeProvider(t *testing.T) {
	p1 := &mockProvider{}
	p2 := &mockProvider{}
	cp := NewCompositeProvider(p1, p2)

	assert.NotNil(t, cp, "NewCompositeProvider returned nil")
	assert.Len(t, cp.Providers, 2, "Expected 2 providers")
	assert.Equal(t, p1, cp.Providers[0], "Provider 1 not set correctly")
	assert.Equal(t, p2, cp.Providers[1], "Provider 2 not set correctly")

	cpEmpty := NewCompositeProvider()
	assert.NotNil(t, cpEmpty, "NewCompositeProvider with no args returned nil")
	assert.Len(t, cpEmpty.Providers, 0, "Expected 0 providers for empty NewCompositeProvider")
}

func TestCompositeProvider_GetSupportedKinds(t *testing.T) {
	tests := []struct {
		name      string
		providers []project.Provider
		expected  []string
	}{
		{
			name:      "no providers",
			providers: []project.Provider{},
			expected:  []string{},
		},
		{
			name: "single provider",
			providers: []project.Provider{
				&mockProvider{supportedKinds: []string{"kindA", "kindB"}},
			},
			expected: []string{"kindA", "kindB"},
		},
		{
			name: "multiple providers with unique kinds",
			providers: []project.Provider{
				&mockProvider{supportedKinds: []string{"kindA"}},
				&mockProvider{supportedKinds: []string{"kindB"}},
			},
			expected: []string{"kindA", "kindB"},
		},
		{
			name: "multiple providers with overlapping kinds",
			providers: []project.Provider{
				&mockProvider{supportedKinds: []string{"kindA", "kindB"}},
				&mockProvider{supportedKinds: []string{"kindB", "kindC"}},
			},
			expected: []string{"kindA", "kindB", "kindC"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := NewCompositeProvider(tt.providers...)
			actual := cp.GetSupportedKinds()
			sort.Strings(actual)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, actual, "Expected kinds do not match")
		})
	}
}

func TestCompositeProvider_GetSupportedTypes(t *testing.T) {
	tests := []struct {
		name      string
		providers []project.Provider
		expected  []string
	}{
		{
			name:      "no providers",
			providers: []project.Provider{},
			expected:  []string{},
		},
		{
			name: "single provider",
			providers: []project.Provider{
				&mockProvider{supportedTypes: []string{"typeA", "typeB"}},
			},
			expected: []string{"typeA", "typeB"},
		},
		{
			name: "multiple providers with unique types",
			providers: []project.Provider{
				&mockProvider{supportedTypes: []string{"typeA"}},
				&mockProvider{supportedTypes: []string{"typeB"}},
			},
			expected: []string{"typeA", "typeB"},
		},
		{
			name: "multiple providers with overlapping types",
			providers: []project.Provider{
				&mockProvider{supportedTypes: []string{"typeA", "typeB"}},
				&mockProvider{supportedTypes: []string{"typeB", "typeC"}},
			},
			expected: []string{"typeA", "typeB", "typeC"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := NewCompositeProvider(tt.providers...)
			actual := cp.GetSupportedTypes()
			sort.Strings(actual)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, actual, "Expected types do not match")
		})
	}
}

func TestCompositeProvider_Validate(t *testing.T) {
	errTest := errors.New("test validation error")
	errTest2 := errors.New("test validation error 2")

	tests := []struct {
		name        string
		providers   []*mockProvider
		expectedErr error
	}{
		{
			name:        "no providers",
			providers:   []*mockProvider{},
			expectedErr: nil,
		},
		{
			name: "single provider, no error",
			providers: []*mockProvider{
				{validateErr: nil},
			},
			expectedErr: nil,
		},
		{
			name: "single provider, with error",
			providers: []*mockProvider{
				{validateErr: errTest},
			},
			expectedErr: errTest,
		},
		{
			name: "multiple providers, no error",
			providers: []*mockProvider{
				{validateErr: nil},
				{validateErr: nil},
			},
			expectedErr: nil,
		},
		{
			name: "multiple providers, first errors",
			providers: []*mockProvider{
				{validateErr: errTest},
				{validateErr: errTest2}, // This one won't be called
			},
			expectedErr: errTest,
		},
		{
			name: "multiple providers, second errors",
			providers: []*mockProvider{
				{validateErr: nil},
				{validateErr: errTest},
			},
			expectedErr: errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerInterfaces := make([]project.Provider, len(tt.providers))
			for i, p := range tt.providers {
				providerInterfaces[i] = p
			}
			cp := NewCompositeProvider(providerInterfaces...)
			err := cp.Validate()

			assert.ErrorIs(t, err, tt.expectedErr)

			for i, p := range tt.providers {
				if tt.expectedErr != nil && errors.Is(tt.expectedErr, p.validateErr) {
					// If this provider was the one that errored, it should have been called
					assert.True(t, p.validateCalled, "Provider %d Validate() not called when it should have errored", i)
					// Subsequent providers should not be called if a previous one errored
					for j := i + 1; j < len(tt.providers); j++ {
						assert.False(t, tt.providers[j].validateCalled, "Provider %d Validate() called after a previous provider errored", j)
					}
					break // Stop checking further providers if one errored as expected
				} else if tt.expectedErr == nil {
					// If no error was expected, all providers should have been called
					assert.True(t, p.validateCalled, "Provider %d Validate() not called when no error was expected", i)
				}
			}
		})
	}
}

func TestCompositeProvider_LoadSpec(t *testing.T) {
	specKindA := &specs.Spec{Kind: "kindA"}
	specKindB := &specs.Spec{Kind: "kindB"}
	specUnknown := &specs.Spec{Kind: "unknownKind"}
	errTest := errors.New("test loadspec error")

	pA := &mockProvider{supportedKinds: []string{"kindA"}}
	pB := &mockProvider{supportedKinds: []string{"kindB"}, loadSpecErr: errTest}

	tests := []struct {
		name         string
		providers    []project.Provider
		path         string
		spec         *specs.Spec
		expectedErr  error
		expectCallOn *mockProvider // which provider should be called
		expectedPath string
		expectedSpec *specs.Spec
	}{
		{
			name:        "no providers",
			providers:   []project.Provider{},
			path:        "path/to/spec.yaml",
			spec:        specKindA,
			expectedErr: fmt.Errorf("no provider found for kind %s", specKindA.Kind),
		},
		{
			name:         "provider found, no error",
			providers:    []project.Provider{pA, pB},
			path:         "pathA.yaml",
			spec:         specKindA,
			expectedErr:  nil,
			expectCallOn: pA,
			expectedPath: "pathA.yaml",
			expectedSpec: specKindA,
		},
		{
			name:         "provider found, with error",
			providers:    []project.Provider{pA, pB},
			path:         "pathB.yaml",
			spec:         specKindB,
			expectedErr:  errTest,
			expectCallOn: pB,
			expectedPath: "pathB.yaml",
			expectedSpec: specKindB,
		},
		{
			name:        "provider not found for kind",
			providers:   []project.Provider{pA, pB},
			path:        "pathUnknown.yaml",
			spec:        specUnknown,
			expectedErr: fmt.Errorf("no provider found for kind %s", specUnknown.Kind),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset call trackers for mocks
			pA.loadSpecCalledWith.path = ""
			pA.loadSpecCalledWith.spec = nil
			pB.loadSpecCalledWith.path = ""
			pB.loadSpecCalledWith.spec = nil

			cp := NewCompositeProvider(tt.providers...)
			err := cp.LoadSpec(tt.path, tt.spec)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.expectCallOn != nil {
				assert.Equal(t, tt.expectedPath, tt.expectCallOn.loadSpecCalledWith.path)
				assert.Equal(t, tt.expectedSpec, tt.expectCallOn.loadSpecCalledWith.spec)
			} else {
				// Ensure no provider was called if none was expected
				assert.Nil(t, pA.loadSpecCalledWith.spec, "pA.LoadSpec called unexpectedly")
				assert.Nil(t, pB.loadSpecCalledWith.spec, "pB.LoadSpec called unexpectedly")
			}
		})
	}
}

func TestCompositeProvider_GetResourceGraph(t *testing.T) {
	graph1 := resources.NewGraph()
	graph1.AddResource(resources.NewResource("id1", "typeA", resources.ResourceData{"key": "val1"}, nil))
	graph2 := resources.NewGraph()
	graph2.AddResource(resources.NewResource("id2", "typeB", resources.ResourceData{"key": "val2"}, nil))
	errTest := errors.New("test getresourcegraph error")

	tests := []struct {
		name         string
		providers    []*mockProvider
		expectedURNs []string // URNs in the final graph
		expectedErr  error
	}{
		{
			name:         "no providers",
			providers:    []*mockProvider{},
			expectedURNs: []string{},
			expectedErr:  nil,
		},
		{
			name: "single provider, no error",
			providers: []*mockProvider{
				{getResourceGraphVal: graph1},
			},
			expectedURNs: []string{"typeA:id1"},
			expectedErr:  nil,
		},
		{
			name: "single provider, with error",
			providers: []*mockProvider{
				{getResourceGraphErr: errTest},
			},
			expectedURNs: nil,
			expectedErr:  errTest,
		},
		{
			name: "multiple providers, no error, merged graph",
			providers: []*mockProvider{
				{getResourceGraphVal: graph1},
				{getResourceGraphVal: graph2},
			},
			expectedURNs: []string{"typeA:id1", "typeB:id2"},
			expectedErr:  nil,
		},
		{
			name: "multiple providers, first errors",
			providers: []*mockProvider{
				{getResourceGraphErr: errTest},
				{getResourceGraphVal: graph2}, // This one won't be called
			},
			expectedURNs: nil,
			expectedErr:  errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerInterfaces := make([]project.Provider, len(tt.providers))
			for i, p := range tt.providers {
				p.getResourceGraphCalled = false // Reset tracker
				providerInterfaces[i] = p
			}
			cp := NewCompositeProvider(providerInterfaces...)
			graph, err := cp.GetResourceGraph()

			assert.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr == nil {
				assert.NotNil(t, graph, "Expected graph, got nil")
				actualURNs := []string{}
				for urn := range graph.Resources() {
					actualURNs = append(actualURNs, urn)
				}
				sort.Strings(actualURNs)
				sort.Strings(tt.expectedURNs)
				assert.Equal(t, tt.expectedURNs, actualURNs)
			} else {
				assert.Nil(t, graph, "Expected nil graph on error")
			}

			for i, p := range tt.providers {
				if tt.expectedErr != nil && errors.Is(tt.expectedErr, p.getResourceGraphErr) {
					assert.True(t, p.getResourceGraphCalled, "Provider %d GetResourceGraph() not called when it should have errored", i)
					for j := i + 1; j < len(tt.providers); j++ {
						assert.False(t, tt.providers[j].getResourceGraphCalled, "Provider %d GetResourceGraph() called after a previous provider errored", j)
					}
					break
				} else if tt.expectedErr == nil {
					assert.True(t, p.getResourceGraphCalled, "Provider %d GetResourceGraph() not called when no error was expected", i)
				}
			}
		})
	}
}

func TestCompositeProvider_LoadState(t *testing.T) {
	state1 := state.EmptyState()
	state1.AddResource(&state.ResourceState{ID: "id1", Type: "typeA"})
	state2 := state.EmptyState()
	state2.AddResource(&state.ResourceState{ID: "id2", Type: "typeB"})
	errTest := errors.New("test loadstate error")

	tests := []struct {
		name         string
		providers    []*mockProvider
		expectedURNs []string // URNs in the final state
		expectedErr  error
	}{
		{
			name:         "no providers",
			providers:    []*mockProvider{},
			expectedURNs: nil, // NewCompositeProvider.LoadState returns nil state if no providers
			expectedErr:  nil,
		},
		{
			name: "single provider, no error",
			providers: []*mockProvider{
				{loadStateVal: state1},
			},
			expectedURNs: []string{"typeA:id1"},
			expectedErr:  nil,
		},
		{
			name: "single provider, with error",
			providers: []*mockProvider{
				{loadStateErr: errTest},
			},
			expectedURNs: nil,
			expectedErr:  errTest,
		},
		{
			name: "multiple providers, no error, merged state",
			providers: []*mockProvider{
				{loadStateVal: state1},
				{loadStateVal: state2},
			},
			expectedURNs: []string{"typeA:id1", "typeB:id2"},
			expectedErr:  nil,
		},
		{
			name: "multiple providers, first errors",
			providers: []*mockProvider{
				{loadStateErr: errTest},
				{loadStateVal: state2}, // This one won't be called
			},
			expectedURNs: nil,
			expectedErr:  errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerInterfaces := make([]project.Provider, len(tt.providers))
			for i, p := range tt.providers {
				p.loadStateCalled = false // Reset tracker
				providerInterfaces[i] = p
			}
			cp := NewCompositeProvider(providerInterfaces...)
			s, err := cp.LoadState(context.Background())

			assert.ErrorIs(t, err, tt.expectedErr)

			if tt.expectedErr == nil {
				if len(tt.providers) == 0 { // Special case for no providers
					assert.Nil(t, s, "Expected nil state for no providers")
				} else {
					assert.NotNil(t, s, "Expected state, got nil")
					actualURNs := []string{}
					for urn := range s.Resources {
						actualURNs = append(actualURNs, urn)
					}
					sort.Strings(actualURNs)
					sort.Strings(tt.expectedURNs)
					assert.Equal(t, tt.expectedURNs, actualURNs)
				}
			} else {
				assert.Nil(t, s, "Expected nil state on error")
			}

			for i, p := range tt.providers {
				if tt.expectedErr != nil && errors.Is(tt.expectedErr, p.loadStateErr) {
					assert.True(t, p.loadStateCalled, "Provider %d LoadState() not called when it should have errored", i)
					for j := i + 1; j < len(tt.providers); j++ {
						assert.False(t, tt.providers[j].loadStateCalled, "Provider %d LoadState() called after a previous provider errored", j)
					}
					break
				} else if tt.expectedErr == nil && len(tt.providers) > 0 {
					assert.True(t, p.loadStateCalled, "Provider %d LoadState() not called when no error was expected", i)
				}
			}
		})
	}
}

func TestCompositeProvider_ResourceOperations(t *testing.T) {
	ctx := context.Background()
	resStateA := &state.ResourceState{ID: "idA", Type: "typeA"}
	resStateB := &state.ResourceState{ID: "idB", Type: "typeB"}
	resDataA := resources.ResourceData{"key": "valA"}
	resDataB := resources.ResourceData{"key": "valB"}
	errTest := errors.New("test resource op error")

	pA := &mockProvider{supportedTypes: []string{"typeA"}}
	pB := &mockProvider{supportedTypes: []string{"typeB"}, createErr: errTest, updateErr: errTest, deleteErr: errTest, putResourceStateErr: errTest, deleteResourceStateErr: errTest}

	tests := []struct {
		name           string
		op             string // "PutResourceState", "DeleteResourceState", "Create", "Update", "Delete"
		providers      []project.Provider
		urn            string
		resourceType   string
		data           resources.ResourceData
		stateData      resources.ResourceData // for Update
		resourceState  *state.ResourceState   // for Put/DeleteResourceState
		expectedErr    error
		expectCallOn   *mockProvider // which provider should be called
		expectedReturn any           // for Create/Update
	}{
		// PutResourceState
		{name: "PutResourceState no provider for type", op: "PutResourceState", providers: []project.Provider{pA}, urn: resources.URN(resStateB.ID, resStateB.Type), resourceState: resStateB, expectedErr: fmt.Errorf("no provider found for resource type %s", resStateB.Type)},
		{name: "PutResourceState success", op: "PutResourceState", providers: []project.Provider{pA, pB}, urn: resources.URN(resStateA.ID, resStateA.Type), resourceState: resStateA, expectCallOn: pA},
		{name: "PutResourceState error", op: "PutResourceState", providers: []project.Provider{pA, pB}, urn: resources.URN(resStateB.ID, resStateB.Type), resourceState: resStateB, expectedErr: errTest, expectCallOn: pB},
		// DeleteResourceState
		{name: "DeleteResourceState no provider for type", op: "DeleteResourceState", providers: []project.Provider{pA}, resourceState: resStateB, expectedErr: fmt.Errorf("no provider found for resource type %s", resStateB.Type)},
		{name: "DeleteResourceState success", op: "DeleteResourceState", providers: []project.Provider{pA, pB}, resourceState: resStateA, expectCallOn: pA},
		{name: "DeleteResourceState error", op: "DeleteResourceState", providers: []project.Provider{pA, pB}, resourceState: resStateB, expectedErr: errTest, expectCallOn: pB},
		// Create
		{name: "Create no provider for type", op: "Create", providers: []project.Provider{pA}, resourceType: "typeUnknown", data: resDataA, expectedErr: fmt.Errorf("no provider found for resource type typeUnknown")},
		{name: "Create success", op: "Create", providers: []project.Provider{pA, pB}, resourceType: "typeA", data: resDataA, expectCallOn: pA, expectedReturn: &resDataA},
		{name: "Create error", op: "Create", providers: []project.Provider{pA, pB}, resourceType: "typeB", data: resDataB, expectedErr: errTest, expectCallOn: pB},
		// Update
		{name: "Update no provider for type", op: "Update", providers: []project.Provider{pA}, resourceType: "typeUnknown", data: resDataA, stateData: resDataA, expectedErr: fmt.Errorf("no provider found for resource type typeUnknown")},
		{name: "Update success", op: "Update", providers: []project.Provider{pA, pB}, resourceType: "typeA", data: resDataA, stateData: resDataA, expectCallOn: pA, expectedReturn: &resDataA},
		{name: "Update error", op: "Update", providers: []project.Provider{pA, pB}, resourceType: "typeB", data: resDataB, stateData: resDataB, expectedErr: errTest, expectCallOn: pB},
		// Delete
		{name: "Delete no provider for type", op: "Delete", providers: []project.Provider{pA}, resourceType: "typeUnknown", stateData: resDataA, expectedErr: fmt.Errorf("no provider found for resource type typeUnknown")},
		{name: "Delete success", op: "Delete", providers: []project.Provider{pA, pB}, resourceType: "typeA", stateData: resDataA, expectCallOn: pA},
		{name: "Delete error", op: "Delete", providers: []project.Provider{pA, pB}, resourceType: "typeB", stateData: resDataB, expectedErr: errTest, expectCallOn: pB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset call trackers for mocks
			pA.putResourceStateCalledWith.urn = ""
			pA.putResourceStateCalledWith.state = nil
			pA.deleteResourceStateCalledWith = nil
			pA.createCalledWith.id = ""
			pA.updateCalledWith.id = ""
			pA.deleteCalledWith.id = ""
			pB.putResourceStateCalledWith.urn = ""
			pB.putResourceStateCalledWith.state = nil
			pB.deleteResourceStateCalledWith = nil
			pB.createCalledWith.id = ""
			pB.updateCalledWith.id = ""
			pB.deleteCalledWith.id = ""

			// Set return values for successful calls on pA
			pA.createVal = &resDataA
			pA.updateVal = &resDataA

			cp := NewCompositeProvider(tt.providers...)
			var actualReturn any
			var err error

			switch tt.op {
			case "PutResourceState":
				err = cp.PutResourceState(ctx, tt.urn, tt.resourceState)
			case "DeleteResourceState":
				err = cp.DeleteResourceState(ctx, tt.resourceState)
			case "Create":
				actualReturn, err = cp.Create(ctx, "id1", tt.resourceType, tt.data)
			case "Update":
				actualReturn, err = cp.Update(ctx, "id1", tt.resourceType, tt.data, tt.stateData)
			case "Delete":
				err = cp.Delete(ctx, "id1", tt.resourceType, tt.stateData)
			default:
				t.Fatalf("Unknown operation: %s", tt.op)
			}

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedReturn != nil {
				assert.Equal(t, tt.expectedReturn, actualReturn)
			}

			if tt.expectCallOn != nil {
				switch tt.op {
				case "PutResourceState":
					assert.Equal(t, tt.urn, tt.expectCallOn.putResourceStateCalledWith.urn)
					assert.Equal(t, tt.resourceState, tt.expectCallOn.putResourceStateCalledWith.state)
				case "DeleteResourceState":
					assert.Equal(t, tt.resourceState, tt.expectCallOn.deleteResourceStateCalledWith)
				case "Create":
					assert.Equal(t, tt.resourceType, tt.expectCallOn.createCalledWith.resourceType)
					assert.Equal(t, tt.data, tt.expectCallOn.createCalledWith.data)
				case "Update":
					assert.Equal(t, tt.resourceType, tt.expectCallOn.updateCalledWith.resourceType)
					assert.Equal(t, tt.data, tt.expectCallOn.updateCalledWith.data)
					assert.Equal(t, tt.stateData, tt.expectCallOn.updateCalledWith.state)
				case "Delete":
					assert.Equal(t, tt.resourceType, tt.expectCallOn.deleteCalledWith.resourceType)
					assert.Equal(t, tt.stateData, tt.expectCallOn.deleteCalledWith.state)
				}
			} else {
				// Ensure no provider was called if none was expected
				assert.Nil(t, pA.putResourceStateCalledWith.state, "pA.PutResourceState called unexpectedly for op %s", tt.op)
				assert.Nil(t, pA.deleteResourceStateCalledWith, "pA.DeleteResourceState called unexpectedly for op %s", tt.op)
				assert.Empty(t, pA.createCalledWith.id, "pA.Create called unexpectedly for op %s", tt.op)
				assert.Empty(t, pA.updateCalledWith.id, "pA.Update called unexpectedly for op %s", tt.op)
				assert.Empty(t, pA.deleteCalledWith.id, "pA.Delete called unexpectedly for op %s", tt.op)

				assert.Nil(t, pB.putResourceStateCalledWith.state, "pB.PutResourceState called unexpectedly for op %s", tt.op)
				assert.Nil(t, pB.deleteResourceStateCalledWith, "pB.DeleteResourceState called unexpectedly for op %s", tt.op)
				assert.Empty(t, pB.createCalledWith.id, "pB.Create called unexpectedly for op %s", tt.op)
				assert.Empty(t, pB.updateCalledWith.id, "pB.Update called unexpectedly for op %s", tt.op)
				assert.Empty(t, pB.deleteCalledWith.id, "pB.Delete called unexpectedly for op %s", tt.op)
			}
		})
	}
}
