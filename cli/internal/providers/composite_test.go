package providers_test

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
)

func TestNewCompositeProvider(t *testing.T) {
	t.Run("successful creation with multiple providers", func(t *testing.T) {
		p1 := testutils.NewMockProvider()
		p1.SupportedKinds = []string{"kindA"}
		p1.SupportedTypes = []string{"typeA"}
		p2 := testutils.NewMockProvider()
		p2.SupportedKinds = []string{"kindB"}
		p2.SupportedTypes = []string{"typeB"}
		cp, err := providers.NewCompositeProvider(p1, p2)

		assert.NoError(t, err, "NewCompositeProvider returned an error")
		assert.NotNil(t, cp, "NewCompositeProvider returned nil")
		assert.Len(t, cp.Providers, 2, "Expected 2 providers")
		assert.Equal(t, p1, cp.Providers[0], "Provider 1 not set correctly")
		assert.Equal(t, p2, cp.Providers[1], "Provider 2 not set correctly")
	})

	t.Run("error when no providers are given", func(t *testing.T) {
		cp, err := providers.NewCompositeProvider()
		assert.Error(t, err, "NewCompositeProvider should return an error for no providers")
		assert.Nil(t, cp, "NewCompositeProvider should return nil for no providers")
		assert.EqualError(t, err, "at least one provider must be specified")
	})

	t.Run("error when providers support duplicate kinds", func(t *testing.T) {
		p1 := testutils.NewMockProvider()
		p1.SupportedKinds = []string{"kindA", "kindB"}
		p2 := testutils.NewMockProvider()
		p2.SupportedKinds = []string{"kindB", "kindC"} // kindB is duplicate
		cp, err := providers.NewCompositeProvider(p1, p2)
		assert.Error(t, err, "NewCompositeProvider should return an error for duplicate kinds")
		assert.Nil(t, cp, "NewCompositeProvider should return nil for duplicate kinds")
		assert.EqualError(t, err, "duplicate kind 'kindB' supported by multiple providers")
	})

	t.Run("error when providers support duplicate types", func(t *testing.T) {
		p1 := testutils.NewMockProvider()
		p1.SupportedTypes = []string{"typeA", "typeB"}
		p2 := testutils.NewMockProvider()
		p2.SupportedTypes = []string{"typeB", "typeC"} // typeB is duplicate
		cp, err := providers.NewCompositeProvider(p1, p2)
		assert.Error(t, err, "NewCompositeProvider should return an error for duplicate types")
		assert.Nil(t, cp, "NewCompositeProvider should return nil for duplicate types")
		assert.EqualError(t, err, "duplicate type 'typeB' supported by multiple providers")
	})

	t.Run("successful creation with one provider", func(t *testing.T) {
		p1 := testutils.NewMockProvider()
		p1.SupportedKinds = []string{"kindA"}
		p1.SupportedTypes = []string{"typeA"}
		cp, err := providers.NewCompositeProvider(p1)
		assert.NoError(t, err)
		assert.NotNil(t, cp)
		assert.Len(t, cp.Providers, 1)
	})
}

func TestCompositeProvider_GetSupportedKinds(t *testing.T) {
	tests := []struct {
		name      string
		providers []project.Provider
		expected  []string
	}{
		{
			name: "single provider",
			providers: []project.Provider{
				&testutils.MockProvider{SupportedKinds: []string{"kindA", "kindB"}},
			},
			expected: []string{"kindA", "kindB"},
		},
		{
			name: "multiple providers with unique kinds",
			providers: []project.Provider{
				&testutils.MockProvider{SupportedKinds: []string{"kindA"}},
				&testutils.MockProvider{SupportedKinds: []string{"kindB"}},
			},
			expected: []string{"kindA", "kindB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := providers.NewCompositeProvider(tt.providers...)
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
		providers []project.Provider
		expected  []string
	}{
		{
			name: "single provider",
			providers: []project.Provider{
				&testutils.MockProvider{SupportedTypes: []string{"typeA", "typeB"}},
			},
			expected: []string{"typeA", "typeB"},
		},
		{
			name: "multiple providers with unique types",
			providers: []project.Provider{
				&testutils.MockProvider{SupportedTypes: []string{"typeA"}},
				&testutils.MockProvider{SupportedTypes: []string{"typeB"}},
			},
			expected: []string{"typeA", "typeB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := providers.NewCompositeProvider(tt.providers...)
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
	errTest2 := errors.New("test validation error 2")

	tests := []struct {
		name        string
		providers   []*testutils.MockProvider
		expectedErr error
	}{
		{
			name: "single provider, no error",
			providers: []*testutils.MockProvider{
				testutils.NewMockProvider(),
			},
			expectedErr: nil,
		},
		{
			name: "single provider, with error",
			providers: []*testutils.MockProvider{
				{ValidateErr: errTest},
			},
			expectedErr: errTest,
		},
		{
			name: "multiple providers, no error",
			providers: []*testutils.MockProvider{
				testutils.NewMockProvider(),
				testutils.NewMockProvider(),
			},
			expectedErr: nil,
		},
		{
			name: "multiple providers, first errors",
			providers: []*testutils.MockProvider{
				{ValidateErr: errTest},
				{ValidateErr: errTest2}, // This one won't be called
			},
			expectedErr: errTest,
		},
		{
			name: "multiple providers, second errors",
			providers: []*testutils.MockProvider{
				testutils.NewMockProvider(),
				{ValidateErr: errTest},
			},
			expectedErr: errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerInterfaces := make([]project.Provider, len(tt.providers))
			for i, p := range tt.providers {
				p.ResetCallCounters() // Reset for each test run
				providerInterfaces[i] = p
			}
			cp, errCp := providers.NewCompositeProvider(providerInterfaces...)
			assert.NoError(t, errCp)
			assert.NotNil(t, cp)

			graph := resources.NewGraph() // Empty graph for validation
			err := cp.Validate(graph)

			assert.ErrorIs(t, err, tt.expectedErr)

			for i, p := range tt.providers {
				assert.Equal(t, graph, p.ValidateArg, "Provider %d Validate() called with unexpected argument")
				if tt.expectedErr != nil && errors.Is(tt.expectedErr, p.ValidateErr) {
					assert.Equal(t, 1, p.ValidateCalledCount, "Provider %d Validate() not called once when it should have errored", i)
					for j := i + 1; j < len(tt.providers); j++ {
						assert.Equal(t, 0, tt.providers[j].ValidateCalledCount, "Provider %d Validate() called after a previous provider errored", j)
					}
					break
				} else if tt.expectedErr == nil {
					assert.Equal(t, 1, p.ValidateCalledCount, "Provider %d Validate() not called once when no error was expected", i)
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

	pA := testutils.NewMockProvider()
	pA.SupportedKinds = []string{"kindA"}

	pB := testutils.NewMockProvider()
	pB.SupportedKinds = []string{"kindB"}
	pB.LoadSpecErr = errTest

	tests := []struct {
		name         string
		providers    []project.Provider
		path         string
		spec         *specs.Spec
		expectedErr  error
		expectCallOn *testutils.MockProvider // which provider should be called
		expectedPath string
		expectedSpec *specs.Spec
	}{
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
			pA.ResetCallCounters()
			pB.ResetCallCounters()

			cp, errCp := providers.NewCompositeProvider(tt.providers...)
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
	graph2 := resources.NewGraph()
	graph2.AddResource(resources.NewResource("id2", "typeB", resources.ResourceData{"key": "val2"}, nil))
	errTest := errors.New("test getresourcegraph error")

	tests := []struct {
		name         string
		providers    []*testutils.MockProvider
		expectedURNs []string // URNs in the final graph
		expectedErr  error
	}{
		{
			name: "single provider, no error",
			providers: []*testutils.MockProvider{
				{GetResourceGraphVal: graph1},
			},
			expectedURNs: []string{"typeA:id1"},
			expectedErr:  nil,
		},
		{
			name: "single provider, with error",
			providers: []*testutils.MockProvider{
				{GetResourceGraphErr: errTest},
			},
			expectedURNs: nil,
			expectedErr:  errTest,
		},
		{
			name: "multiple providers, no error, merged graph",
			providers: []*testutils.MockProvider{
				{GetResourceGraphVal: graph1},
				{GetResourceGraphVal: graph2},
			},
			expectedURNs: []string{"typeA:id1", "typeB:id2"},
			expectedErr:  nil,
		},
		{
			name: "multiple providers, first errors",
			providers: []*testutils.MockProvider{
				{GetResourceGraphErr: errTest},
				{GetResourceGraphVal: graph2}, // This one won't be called
			},
			expectedURNs: nil,
			expectedErr:  errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providerInterfaces := make([]project.Provider, len(tt.providers))
			for i, p := range tt.providers {
				p.ResetCallCounters()
				providerInterfaces[i] = p
			}
			cp, errCp := providers.NewCompositeProvider(providerInterfaces...)
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
			} else {
				assert.Nil(t, graph, "Expected nil graph on error")
			}

			for i, p := range tt.providers {
				if tt.expectedErr != nil && errors.Is(tt.expectedErr, p.GetResourceGraphErr) {
					assert.Equal(t, 1, p.GetResourceGraphCalledCount, "Provider %d GetResourceGraph() not called once when it should have errored", i)
					for j := i + 1; j < len(tt.providers); j++ {
						assert.Equal(t, 0, tt.providers[j].GetResourceGraphCalledCount, "Provider %d GetResourceGraph() called after a previous provider errored", j)
					}
					break
				} else if tt.expectedErr == nil {
					assert.Equal(t, 1, p.GetResourceGraphCalledCount, "Provider %d GetResourceGraph() not called once when no error was expected", i)
				}
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
		providers      []project.Provider
		urn            string
		resourceType   string
		data           resources.ResourceData
		stateData      resources.ResourceData // for Update
		expectedErr    error
		expectCallOn   *testutils.MockProvider // which provider should be called
		expectedReturn any                     // for Create/Update
	}{
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

			cp, errCp := providers.NewCompositeProvider(tt.providers...)
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
