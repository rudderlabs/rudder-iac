package provider_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestNewCompositeProvider(t *testing.T) {
	t.Run("error when no providers are given", func(t *testing.T) {
		cp, err := provider.NewCompositeProvider(map[string]provider.Provider{})
		assert.Error(t, err, "NewCompositeProvider should return an error for no providers")
		assert.Nil(t, cp, "NewCompositeProvider should return nil for no providers")
		assert.EqualError(t, err, "at least one provider must be specified")
	})

	t.Run("error when providers support duplicate kinds", func(t *testing.T) {
		p1 := testutils.NewMockProvider()
		p1.SupportedKinds = []string{"kindA", "kindB"}
		p2 := testutils.NewMockProvider()
		p2.SupportedKinds = []string{"kindB", "kindC"} // kindB is duplicate
		cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
			"p1": p1,
			"p2": p2,
		})
		assert.Error(t, err, "NewCompositeProvider should return an error for duplicate kinds")
		assert.Nil(t, cp, "NewCompositeProvider should return nil for duplicate kinds")
		assert.EqualError(t, err, "duplicate kind 'kindB' supported by multiple providers")
	})

	t.Run("error when providers support duplicate types", func(t *testing.T) {
		p1 := testutils.NewMockProvider()
		p1.SupportedTypes = []string{"typeA", "typeB"}
		p2 := testutils.NewMockProvider()
		p2.SupportedTypes = []string{"typeB", "typeC"} // typeB is duplicate
		cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
			"p1": p1,
			"p2": p2,
		})
		assert.Error(t, err, "NewCompositeProvider should return an error for duplicate types")
		assert.Nil(t, cp, "NewCompositeProvider should return nil for duplicate types")
		assert.EqualError(t, err, "duplicate type 'typeB' supported by multiple providers")
	})
}

func TestCompositeProvider_GetSupportedKinds(t *testing.T) {
	tests := []struct {
		name      string
		providers map[string]provider.Provider
		expected  []string
	}{
		{
			name: "single provider",
			providers: map[string]provider.Provider{
				"p1": &testutils.MockProvider{SupportedKinds: []string{"kindA", "kindB"}},
			},
			expected: []string{"kindA", "kindB"},
		},
		{
			name: "multiple providers with unique kinds",
			providers: map[string]provider.Provider{
				"p1": &testutils.MockProvider{SupportedKinds: []string{"kindA"}},
				"p2": &testutils.MockProvider{SupportedKinds: []string{"kindB"}},
			},
			expected: []string{"kindA", "kindB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := provider.NewCompositeProvider(tt.providers)
			assert.NoError(t, err)
			assert.NotNil(t, cp)
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
		providers map[string]provider.Provider
		expected  []string
	}{
		{
			name: "single provider",
			providers: map[string]provider.Provider{
				"p1": &testutils.MockProvider{SupportedTypes: []string{"typeA", "typeB"}},
			},
			expected: []string{"typeA", "typeB"},
		},
		{
			name: "multiple providers with unique types",
			providers: map[string]provider.Provider{
				"p1": &testutils.MockProvider{SupportedTypes: []string{"typeA"}},
				"p2": &testutils.MockProvider{SupportedTypes: []string{"typeB"}},
			},
			expected: []string{"typeA", "typeB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := provider.NewCompositeProvider(tt.providers)
			assert.NoError(t, err)
			assert.NotNil(t, cp)
			actual := cp.GetSupportedTypes()
			sort.Strings(actual)
			sort.Strings(tt.expected)
			assert.Equal(t, tt.expected, actual, "Expected types do not match")
		})
	}
}

func TestCompositeProvider_Validate(t *testing.T) {
	errTest := errors.New("test validation error")

	tests := []struct {
		name          string
		providerCount int
		validateErr   error
		expectedErr   error
	}{
		{
			name:          "single provider, no error",
			providerCount: 1,
			validateErr:   nil,
			expectedErr:   nil,
		},
		{
			name:          "single provider, with error",
			providerCount: 1,
			validateErr:   errTest,
			expectedErr:   errTest,
		},
		{
			name:          "multiple providers, no error",
			providerCount: 3,
			validateErr:   nil,
			expectedErr:   nil,
		},
		{
			name:          "multiple providers, with error",
			providerCount: 3,
			validateErr:   errTest,
			expectedErr:   errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a single mock provider instance shared across all map entries
			mockProvider := testutils.NewMockProvider()
			mockProvider.ValidateErr = tt.validateErr

			providerInterfaces := make(map[string]provider.Provider, tt.providerCount)
			for i := 0; i < tt.providerCount; i++ {
				providerInterfaces[fmt.Sprintf("provider-%d", i)] = mockProvider
			}

			cp, errCp := provider.NewCompositeProvider(providerInterfaces)
			assert.NoError(t, errCp)
			assert.NotNil(t, cp)

			graph := resources.NewGraph() // Empty graph for validation
			err := cp.Validate(graph)

			// Verify the error behavior
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
				// Should return exactly one error (stops on first error)
				assert.Equal(t, 1, mockProvider.ValidateErrorReturnedCount, "Should return error exactly once (stop on first error)")
			} else {
				assert.NoError(t, err)
				// All providers should have been called (same mock instance, so count = providerCount)
				assert.Equal(t, tt.providerCount, mockProvider.ValidateCalledCount, "Provider should have been called once per entry")
				assert.Equal(t, 0, mockProvider.ValidateErrorReturnedCount, "Should not return any errors")
			}
			assert.Equal(t, graph, mockProvider.ValidateArg, "Provider Validate() called with unexpected argument")
		})
	}
}

func TestCompositeProvider_LoadSpec(t *testing.T) {
	specKindA := &specs.Spec{Kind: "kindA"}
	specKindB := &specs.Spec{Kind: "kindB"}
	specUnknown := &specs.Spec{Kind: "unknownKind"}
	errTest := errors.New("test loadspec error")

	pA := testutils.NewMockProvider()
	pA.SupportedKinds = []string{"kindA"}

	pB := testutils.NewMockProvider()
	pB.SupportedKinds = []string{"kindB"}
	pB.LoadSpecErr = errTest

	tests := []struct {
		name         string
		providers    map[string]provider.Provider
		path         string
		spec         *specs.Spec
		expectedErr  error
		expectCallOn *testutils.MockProvider // which provider should be called
		expectedPath string
		expectedSpec *specs.Spec
	}{
		{
			name:         "provider found, no error",
			providers:    map[string]provider.Provider{"pA": pA, "pB": pB},
			path:         "pathA.yaml",
			spec:         specKindA,
			expectedErr:  nil,
			expectCallOn: pA,
			expectedPath: "pathA.yaml",
			expectedSpec: specKindA,
		},
		{
			name:         "provider found, with error",
			providers:    map[string]provider.Provider{"pA": pA, "pB": pB},
			path:         "pathB.yaml",
			spec:         specKindB,
			expectedErr:  errTest,
			expectCallOn: pB,
			expectedPath: "pathB.yaml",
			expectedSpec: specKindB,
		},
		{
			name:        "provider not found for kind",
			providers:   map[string]provider.Provider{"pA": pA, "pB": pB},
			path:        "pathUnknown.yaml",
			spec:        specUnknown,
			expectedErr: fmt.Errorf("no provider found for kind %s", specUnknown.Kind),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pA.ResetCallCounters()
			pB.ResetCallCounters()

			cp, errCp := provider.NewCompositeProvider(tt.providers)
			assert.NoError(t, errCp)
			assert.NotNil(t, cp)

			err := cp.LoadSpec(tt.path, tt.spec)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.expectCallOn != nil {
				assert.Len(t, tt.expectCallOn.LoadSpecCalledWithArgs, 1, "Expected LoadSpec to be called once on the target provider")
				if len(tt.expectCallOn.LoadSpecCalledWithArgs) > 0 {
					assert.Equal(t, tt.expectedPath, tt.expectCallOn.LoadSpecCalledWithArgs[0].Path)
					assert.Equal(t, tt.expectedSpec, tt.expectCallOn.LoadSpecCalledWithArgs[0].Spec)
				}
			} else {
				assert.Empty(t, pA.LoadSpecCalledWithArgs, "pA.LoadSpec called unexpectedly")
				assert.Empty(t, pB.LoadSpecCalledWithArgs, "pB.LoadSpec called unexpectedly")
			}
		})
	}
}

func TestCompositeProvider_GetResourceGraph(t *testing.T) {
	graph1 := resources.NewGraph()
	graph1.AddResource(resources.NewResource("id1", "typeA", resources.ResourceData{"key": "val1"}, nil))
	errTest := errors.New("test getresourcegraph error")

	tests := []struct {
		name          string
		providerCount int
		returnGraph   *resources.Graph
		returnErr     error
		expectedURNs  []string // URNs in the final graph
		expectedErr   error
	}{
		{
			name:          "single provider, no error",
			providerCount: 1,
			returnGraph:   graph1,
			returnErr:     nil,
			expectedURNs:  []string{"typeA:id1"},
			expectedErr:   nil,
		},
		{
			name:          "single provider, with error",
			providerCount: 1,
			returnGraph:   nil,
			returnErr:     errTest,
			expectedURNs:  nil,
			expectedErr:   errTest,
		},
		{
			name:          "multiple providers, no error, merged graph",
			providerCount: 3,
			returnGraph:   graph1,
			returnErr:     nil,
			expectedURNs:  []string{"typeA:id1"}, // Same graph returned 3 times, merges into one
			expectedErr:   nil,
		},
		{
			name:          "multiple providers, with error",
			providerCount: 3,
			returnGraph:   nil,
			returnErr:     errTest,
			expectedURNs:  nil,
			expectedErr:   errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a single mock provider instance shared across all map entries
			mockProvider := testutils.NewMockProvider()
			mockProvider.GetResourceGraphVal = tt.returnGraph
			mockProvider.GetResourceGraphErr = tt.returnErr

			providerInterfaces := make(map[string]provider.Provider, tt.providerCount)
			for i := 0; i < tt.providerCount; i++ {
				providerInterfaces[fmt.Sprintf("provider-%d", i)] = mockProvider
			}

			cp, errCp := provider.NewCompositeProvider(providerInterfaces)
			assert.NoError(t, errCp)
			assert.NotNil(t, cp)
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
				assert.Equal(t, tt.providerCount, mockProvider.GetResourceGraphCalledCount, "Provider should have been called once per entry")
				assert.Equal(t, 0, mockProvider.GetResourceGraphErrorReturnedCount, "Should not return any errors")
			} else {
				assert.Nil(t, graph, "Expected nil graph on error")
				// Should return exactly one error (stops on first error)
				assert.Equal(t, 1, mockProvider.GetResourceGraphErrorReturnedCount, "Should return error exactly once (stop on first error)")
			}
		})
	}
}

func TestCompositeProvider_ResourceOperations(t *testing.T) {
	ctx := context.Background()
	resDataA := resources.ResourceData{"key": "valA"}
	resDataB := resources.ResourceData{"key": "valB"}
	errTest := errors.New("test resource op error")

	pA := testutils.NewMockProvider()
	pA.SupportedTypes = []string{"typeA"}

	pB := testutils.NewMockProvider()
	pB.SupportedTypes = []string{"typeB"}
	pB.CreateErr = errTest
	pB.UpdateErr = errTest
	pB.DeleteErr = errTest

	tests := []struct {
		name           string
		op             string // "Create", "Update", "Delete"
		providers      map[string]provider.Provider
		urn            string
		resourceType   string
		data           resources.ResourceData
		stateData      resources.ResourceData // for Update
		expectedErr    error
		expectCallOn   *testutils.MockProvider // which provider should be called
		expectedReturn any                     // for Create/Update
	}{
		// Create
		{name: "Create no provider for type", op: "Create", providers: map[string]provider.Provider{"pA": pA}, resourceType: "typeUnknown", data: resDataA, expectedErr: fmt.Errorf("no provider found for resource type typeUnknown")},
		{name: "Create success", op: "Create", providers: map[string]provider.Provider{"pA": pA, "pB": pB}, resourceType: "typeA", data: resDataA, expectCallOn: pA, expectedReturn: &resDataA},
		{name: "Create error", op: "Create", providers: map[string]provider.Provider{"pA": pA, "pB": pB}, resourceType: "typeB", data: resDataB, expectedErr: errTest, expectCallOn: pB},
		// Update
		{name: "Update no provider for type", op: "Update", providers: map[string]provider.Provider{"pA": pA}, resourceType: "typeUnknown", data: resDataA, stateData: resDataA, expectedErr: fmt.Errorf("no provider found for resource type typeUnknown")},
		{name: "Update success", op: "Update", providers: map[string]provider.Provider{"pA": pA, "pB": pB}, resourceType: "typeA", data: resDataA, stateData: resDataA, expectCallOn: pA, expectedReturn: &resDataA},
		{name: "Update error", op: "Update", providers: map[string]provider.Provider{"pA": pA, "pB": pB}, resourceType: "typeB", data: resDataB, stateData: resDataB, expectedErr: errTest, expectCallOn: pB},
		// Delete
		{name: "Delete no provider for type", op: "Delete", providers: map[string]provider.Provider{"pA": pA}, resourceType: "typeUnknown", stateData: resDataA, expectedErr: fmt.Errorf("no provider found for resource type typeUnknown")},
		{name: "Delete success", op: "Delete", providers: map[string]provider.Provider{"pA": pA, "pB": pB}, resourceType: "typeA", stateData: resDataA, expectCallOn: pA},
		{name: "Delete error", op: "Delete", providers: map[string]provider.Provider{"pA": pA, "pB": pB}, resourceType: "typeB", stateData: resDataB, expectedErr: errTest, expectCallOn: pB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pA.ResetCallCounters()
			pB.ResetCallCounters()

			// Set return values for successful calls on pA if it's the expected one
			if tt.expectCallOn == pA {
				if tt.expectedReturn != nil {
					pA.CreateVal = tt.expectedReturn.(*resources.ResourceData)
					pA.UpdateVal = tt.expectedReturn.(*resources.ResourceData)
				} else {
					pA.CreateVal = nil
					pA.UpdateVal = nil
				}
			} else {
				pA.CreateVal = nil // Ensure pA doesn't return values if not expected
				pA.UpdateVal = nil
			}
			// pB already has error values set if it's the target for error cases

			cp, errCp := provider.NewCompositeProvider(tt.providers)
			assert.NoError(t, errCp)
			assert.NotNil(t, cp)

			var actualReturn any
			var err error

			switch tt.op {
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
				case "Create":
					assert.Equal(t, tt.resourceType, tt.expectCallOn.CreateCalledWithArg.ResourceType)
					assert.Equal(t, tt.data, tt.expectCallOn.CreateCalledWithArg.Data)
				case "Update":
					assert.Equal(t, tt.resourceType, tt.expectCallOn.UpdateCalledWithArg.ResourceType)
					assert.Equal(t, tt.data, tt.expectCallOn.UpdateCalledWithArg.Data)
					assert.Equal(t, tt.stateData, tt.expectCallOn.UpdateCalledWithArg.State)
				case "Delete":
					assert.Equal(t, tt.resourceType, tt.expectCallOn.DeleteCalledWithArg.ResourceType)
					assert.Equal(t, tt.stateData, tt.expectCallOn.DeleteCalledWithArg.State)
				}
			} else {
				// Ensure no provider was called if none was expected (check one field from each relevant arg struct)
				assert.Empty(t, pA.CreateCalledWithArg.ID, "pA.Create called unexpectedly for op %s", tt.op)
				assert.Empty(t, pA.UpdateCalledWithArg.ID, "pA.Update called unexpectedly for op %s", tt.op)
				assert.Empty(t, pA.DeleteCalledWithArg.ID, "pA.Delete called unexpectedly for op %s", tt.op)

				assert.Empty(t, pB.CreateCalledWithArg.ID, "pB.Create called unexpectedly for op %s", tt.op)
				assert.Empty(t, pB.UpdateCalledWithArg.ID, "pB.Update called unexpectedly for op %s", tt.op)
				assert.Empty(t, pB.DeleteCalledWithArg.ID, "pB.Delete called unexpectedly for op %s", tt.op)
			}
		})
	}
}
