package testorchestrator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

// --- Mock Implementations ---

type mockRemoteStateLoader struct {
	loadResourcesFromRemoteFunc func(ctx context.Context) (*resources.RemoteResources, error)
	mapRemoteToStateFunc        func(collection *resources.RemoteResources) (*state.State, error)
}

func (m *mockRemoteStateLoader) LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error) {
	if m.loadResourcesFromRemoteFunc != nil {
		return m.loadResourcesFromRemoteFunc(ctx)
	}
	return resources.NewRemoteResources(), nil
}

func (m *mockRemoteStateLoader) MapRemoteToState(collection *resources.RemoteResources) (*state.State, error) {
	if m.mapRemoteToStateFunc != nil {
		return m.mapRemoteToStateFunc(collection)
	}
	return state.EmptyState(), nil
}

type mockTransformationStore struct {
	batchTestFunc        func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error)
	createTransformation func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error)
	updateTransformation func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error)
	createLibraryFunc    func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
	updateLibraryFunc    func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
}

func newMockStore() *mockTransformationStore {
	return &mockTransformationStore{}
}

func (m *mockTransformationStore) BatchTest(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
	if m.batchTestFunc != nil {
		return m.batchTestFunc(ctx, req)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) CreateTransformation(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	if m.createTransformation != nil {
		return m.createTransformation(ctx, req, publish)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) CreateLibrary(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	if m.createLibraryFunc != nil {
		return m.createLibraryFunc(ctx, req, publish)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) UpdateTransformation(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
	if m.updateTransformation != nil {
		return m.updateTransformation(ctx, id, req, publish)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) GetTransformation(ctx context.Context, id string) (*transformations.Transformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) ListTransformations(ctx context.Context) ([]*transformations.Transformation, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) DeleteTransformation(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (m *mockTransformationStore) SetTransformationExternalID(ctx context.Context, id string, externalID string) error {
	return errors.New("not implemented")
}

func (m *mockTransformationStore) UpdateLibrary(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
	if m.updateLibraryFunc != nil {
		return m.updateLibraryFunc(ctx, id, req, publish)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) GetLibrary(ctx context.Context, id string) (*transformations.TransformationLibrary, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) ListLibraries(ctx context.Context) ([]*transformations.TransformationLibrary, error) {
	return nil, errors.New("not implemented")
}

func (m *mockTransformationStore) DeleteLibrary(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func (m *mockTransformationStore) SetLibraryExternalID(ctx context.Context, id string, externalID string) error {
	return errors.New("not implemented")
}

func (m *mockTransformationStore) BatchPublish(ctx context.Context, req *transformations.BatchPublishRequest) (*transformations.BatchPublishResponse, error) {
	return nil, errors.New("not implemented")
}

// --- Helper functions ---

func newTestState() *state.State {
	return state.EmptyState()
}

// --- Runner.Run ---

func TestRunnerRun(t *testing.T) {
	t.Run("returns empty results when no resources to test", func(t *testing.T) {
		ctx := context.Background()
		mockLoader := &mockRemoteStateLoader{
			loadResourcesFromRemoteFunc: func(ctx context.Context) (*resources.RemoteResources, error) {
				return resources.NewRemoteResources(), nil
			},
			mapRemoteToStateFunc: func(collection *resources.RemoteResources) (*state.State, error) {
				return state.EmptyState(), nil
			},
		}

		graph := resources.NewGraph()
		runner := &Runner{
			loader:      mockLoader,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.NoError(t, err)
		assert.True(t, results.Pass)
		assert.Empty(t, results.Transformations)
		assert.Empty(t, results.Libraries)
	})

	t.Run("returns error when loading remote resources fails", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("network error")
		mockLoader := &mockRemoteStateLoader{
			loadResourcesFromRemoteFunc: func(ctx context.Context) (*resources.RemoteResources, error) {
				return nil, expectedErr
			},
		}

		graph := resources.NewGraph()
		runner := &Runner{
			loader:  mockLoader,
			planner: NewPlanner(graph),
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "loading remote resources")
	})

	t.Run("returns error when mapping remote to state fails", func(t *testing.T) {
		ctx := context.Background()
		expectedErr := errors.New("mapping error")
		mockLoader := &mockRemoteStateLoader{
			loadResourcesFromRemoteFunc: func(ctx context.Context) (*resources.RemoteResources, error) {
				return resources.NewRemoteResources(), nil
			},
			mapRemoteToStateFunc: func(collection *resources.RemoteResources) (*state.State, error) {
				return nil, expectedErr
			},
		}

		graph := resources.NewGraph()
		runner := &Runner{
			loader:  mockLoader,
			planner: NewPlanner(graph),
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "building remote state")
	})

	t.Run("returns error when transformation version resolution fails", func(t *testing.T) {
		ctx := context.Background()
		mockLoader := &mockRemoteStateLoader{}
		mockStore := newMockStore()

		graph := resources.NewGraph()
		trans := resources.NewResource(
			"trans-1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				ID:   "trans-1",
				Name: "Test Transformation",
				Code: "function transformEvent(event) { return event; }",
			}),
		)
		graph.AddResource(trans)

		runner := &Runner{
			loader:      mockLoader,
			store:       mockStore,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		expectedErr := errors.New("version creation failed")
		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return nil, expectedErr
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "resolving transformation versions")
	})

	t.Run("successfully runs tests for transformation without dependencies", func(t *testing.T) {
		ctx := context.Background()
		mockLoader := &mockRemoteStateLoader{}
		mockStore := newMockStore()

		graph := resources.NewGraph()
		trans := resources.NewResource(
			"trans-1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				ID:   "trans-1",
				Name: "Test Transformation",
				Code: "function transformEvent(event) { return event; }",
			}),
		)
		graph.AddResource(trans)

		runner := &Runner{
			loader:      mockLoader,
			store:       mockStore,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return &transformations.Transformation{ID: "remote-trans-1", VersionID: "trans-ver-1"}, nil
		}

		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			require.Len(t, req.Transformations, 1)
			assert.Equal(t, "trans-ver-1", req.Transformations[0].VersionID)
			assert.Empty(t, req.Libraries)

			return &transformations.BatchTestResponse{
				Pass: true,
				ValidationOutput: transformations.ValidationOutput{
					Transformations: []transformations.TransformationTestResult{
						{
							ID:        "remote-trans-1",
							VersionID: "trans-ver-1",
							Pass:      true,
						},
					},
				},
			}, nil
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.NoError(t, err)
		assert.Len(t, results.Transformations, 1)
		assert.Empty(t, results.Libraries)
		assert.Equal(t, "trans-ver-1", results.Transformations[0].Result.VersionID)
	})

	t.Run("successfully runs tests for standalone libraries", func(t *testing.T) {
		ctx := context.Background()
		mockLoader := &mockRemoteStateLoader{}
		mockStore := newMockStore()

		graph := resources.NewGraph()
		lib := resources.NewResource(
			"lib-standalone",
			"transformation-library",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.LibraryResource{
				ID:         "lib-standalone",
				Name:       "Standalone Library",
				Code:       "function helper() { return true; }",
				ImportName: "standaloneLib",
			}),
		)
		graph.AddResource(lib)

		runner := &Runner{
			loader:      mockLoader,
			store:       mockStore,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		mockStore.createLibraryFunc = func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			return &transformations.TransformationLibrary{ID: "remote-lib-standalone", VersionID: "lib-ver-standalone"}, nil
		}

		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			assert.Len(t, req.Libraries, 1)
			assert.Equal(t, "lib-ver-standalone", req.Libraries[0].VersionID)
			assert.Empty(t, req.Transformations)

			return &transformations.BatchTestResponse{
				Pass: true,
				ValidationOutput: transformations.ValidationOutput{
					Libraries: []transformations.LibraryTestResult{
						{VersionID: "lib-ver-standalone", Pass: true},
					},
				},
			}, nil
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.NoError(t, err)
		assert.Empty(t, results.Transformations)
		assert.Len(t, results.Libraries, 1)
		assert.Equal(t, "lib-ver-standalone", results.Libraries[0].VersionID)
	})

	t.Run("aggregates results from multiple test units", func(t *testing.T) {
		ctx := context.Background()
		mockLoader := &mockRemoteStateLoader{}
		mockStore := newMockStore()

		graph := resources.NewGraph()

		trans1 := resources.NewResource(
			"trans-1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				ID:   "trans-1",
				Name: "Transformation 1",
				Code: "function transformEvent(event) { return event; }",
			}),
		)
		graph.AddResource(trans1)

		trans2 := resources.NewResource(
			"trans-2",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				ID:   "trans-2",
				Name: "Transformation 2",
				Code: "function transformEvent(event) { return event; }",
			}),
		)
		graph.AddResource(trans2)

		runner := &Runner{
			loader:      mockLoader,
			store:       mockStore,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return &transformations.Transformation{ID: "remote-" + req.Name, VersionID: "ver-" + req.Name}, nil
		}

		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			require.Len(t, req.Transformations, 1)

			return &transformations.BatchTestResponse{
				Pass: true,
				ValidationOutput: transformations.ValidationOutput{
					Transformations: []transformations.TransformationTestResult{
						{
							ID:        req.Transformations[0].VersionID,
							VersionID: req.Transformations[0].VersionID,
							Pass:      true,
						},
					},
				},
			}, nil
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.NoError(t, err)
		assert.Len(t, results.Transformations, 2, "should have results from 2 transformations")
	})

	t.Run("returns error when batch test fails", func(t *testing.T) {
		ctx := context.Background()
		mockLoader := &mockRemoteStateLoader{}
		mockStore := newMockStore()

		graph := resources.NewGraph()
		trans := resources.NewResource(
			"trans-1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				ID:   "trans-1",
				Name: "Test Transformation",
				Code: "function transformEvent(event) { return event; }",
			}),
		)
		graph.AddResource(trans)

		runner := &Runner{
			loader:      mockLoader,
			store:       mockStore,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return &transformations.Transformation{ID: "remote-trans-1", VersionID: "ver-1"}, nil
		}

		expectedErr := errors.New("batch test API failure")
		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			return nil, expectedErr
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.Error(t, err)
		assert.Nil(t, results)
		assert.Contains(t, err.Error(), "running tests for trans-1")
	})

	t.Run("uses remote ID from importable resources for version resolution", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		// Create graph with importable transformation (has import metadata)
		graph := resources.NewGraph()
		trans := resources.NewResource(
			"trans-imported",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{
				ID:   "trans-imported",
				Name: "Imported Transformation",
				Code: "function transformEvent(event) { return event; }",
			}),
			resources.WithResourceImportMetadata("remote-imported-id", ""),
		)
		graph.AddResource(trans)

		// Mock loader that returns remote resources with different ID
		mockLoader := &mockRemoteStateLoader{
			loadResourcesFromRemoteFunc: func(ctx context.Context) (*resources.RemoteResources, error) {
				remoteResources := resources.NewRemoteResources()
				remoteResources.Set("transformation", map[string]*resources.RemoteResource{
					"remote-imported-id": {
						ID:         "remote-imported-id",
						ExternalID: "trans-imported",
						Data: &model.RemoteTransformation{
							Transformation: &transformations.Transformation{
								ID:         "remote-imported-id",
								ExternalID: "trans-imported",
								Name:       "Imported Transformation",
							},
						},
					},
				})
				return remoteResources, nil
			},
			mapRemoteToStateFunc: func(collection *resources.RemoteResources) (*state.State, error) {
				st := state.EmptyState()
				st.AddResource(&state.ResourceState{
					ID:   "trans-imported",
					Type: "transformation",
					OutputRaw: &model.TransformationState{
						ID:        "remote-different-id", // Different ID to verify import metadata takes precedence
						VersionID: "ver-existing",
					},
				})
				return st, nil
			},
		}

		runner := &Runner{
			loader:      mockLoader,
			store:       mockStore,
			planner:     NewPlanner(graph),
			workspaceID: "ws-123",
		}

		// Track which remote ID is used for update
		var usedRemoteID string
		mockStore.updateTransformation = func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			usedRemoteID = id
			return &transformations.Transformation{
				ID:         id,
				VersionID:  "ver-updated",
				ExternalID: "trans-imported",
			}, nil
		}

		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			require.Len(t, req.Transformations, 1)
			assert.Equal(t, "ver-updated", req.Transformations[0].VersionID)

			return &transformations.BatchTestResponse{
				Pass: true,
				ValidationOutput: transformations.ValidationOutput{
					Transformations: []transformations.TransformationTestResult{
						{
							ID:        "remote-imported-id",
							VersionID: "ver-updated",
							Pass:      true,
						},
					},
				},
			}, nil
		}

		results, err := runner.Run(ctx, ModeAll, "")

		require.NoError(t, err)
		assert.Len(t, results.Transformations, 1)
		assert.Equal(t, "ver-updated", results.Transformations[0].Result.VersionID)
		// Verify that import metadata's remote ID was used (not the one from remote state)
		assert.Equal(t, "remote-imported-id", usedRemoteID, "should use remote ID from import metadata, not from remote state")
	})
}

// --- buildTestRequest ---

func TestBuildTestRequest(t *testing.T) {
	t.Run("produces request with correct transformation versionID and test suite", func(t *testing.T) {
		testDefs := []*transformations.TestDefinition{
			{Name: "test-a", Input: []any{map[string]any{"type": "track"}}},
			{Name: "test-b", Input: []any{map[string]any{"type": "identify"}}},
		}

		req := buildTestRequest("trans-ver-1", testDefs, nil)

		require.Len(t, req.Transformations, 1)
		assert.Equal(t, "trans-ver-1", req.Transformations[0].VersionID)
		require.Len(t, req.Transformations[0].TestSuite, 2)
		assert.Equal(t, "test-a", req.Transformations[0].TestSuite[0].Name)
		assert.Equal(t, "test-b", req.Transformations[0].TestSuite[1].Name)
	})

	t.Run("includes library inputs for each provided versionID", func(t *testing.T) {
		req := buildTestRequest("trans-ver-1", nil, []string{"lib-ver-1", "lib-ver-2"})

		require.Len(t, req.Libraries, 2)
		assert.Equal(t, "lib-ver-1", req.Libraries[0].VersionID)
		assert.Equal(t, "lib-ver-2", req.Libraries[1].VersionID)
	})

	t.Run("empty library list produces no library inputs", func(t *testing.T) {
		req := buildTestRequest("trans-ver-1", nil, []string{})

		assert.Empty(t, req.Libraries)
	})

	t.Run("test suite preserves expectedOutput", func(t *testing.T) {
		expected := []any{map[string]any{"type": "track", "processed": true}}
		testDefs := []*transformations.TestDefinition{
			{
				Name:           "with-output",
				Input:          []any{map[string]any{"type": "track"}},
				ExpectedOutput: expected,
			},
		}

		req := buildTestRequest("ver-1", testDefs, nil)

		require.Len(t, req.Transformations[0].TestSuite, 1)
		assert.Equal(t, expected, req.Transformations[0].TestSuite[0].ExpectedOutput)
	})

	t.Run("preserves test definition order", func(t *testing.T) {
		testDefs := []*transformations.TestDefinition{
			{Name: "first"},
			{Name: "second"},
			{Name: "third"},
		}

		req := buildTestRequest("ver-1", testDefs, nil)

		suite := req.Transformations[0].TestSuite
		assert.Equal(t, "first", suite[0].Name)
		assert.Equal(t, "second", suite[1].Name)
		assert.Equal(t, "third", suite[2].Name)
	})
}

// --- getVersionIDsForUnitLibraries ---

func TestGetVersionIDsForUnitLibraries(t *testing.T) {
	t.Run("extracts versionIDs for all libraries", func(t *testing.T) {
		libraries := []*model.LibraryResource{
			{ID: "lib-1"},
			{ID: "lib-2"},
		}
		versionMap := map[string]string{
			"lib-1": "ver-1",
			"lib-2": "ver-2",
			"lib-3": "ver-3", // not in libraries
		}

		versionIDs := getVersionIDsForUnitLibraries(libraries, versionMap)

		require.Len(t, versionIDs, 2)
		assert.Contains(t, versionIDs, "ver-1")
		assert.Contains(t, versionIDs, "ver-2")
		assert.NotContains(t, versionIDs, "ver-3")
	})

	t.Run("returns empty slice when no libraries provided", func(t *testing.T) {
		libraries := []*model.LibraryResource{}
		versionIDs := getVersionIDsForUnitLibraries(libraries, map[string]string{"lib-1": "ver-1"})

		assert.Empty(t, versionIDs)
	})

	t.Run("skips libraries missing from the version map", func(t *testing.T) {
		libraries := []*model.LibraryResource{
			{ID: "lib-1"},
			{ID: "lib-missing"},
		}
		versionMap := map[string]string{"lib-1": "ver-1"}

		versionIDs := getVersionIDsForUnitLibraries(libraries, versionMap)

		require.Len(t, versionIDs, 1)
		assert.Equal(t, "ver-1", versionIDs[0])
	})
}

// --- runTestUnitTask ---

func TestRunTestUnitTask(t *testing.T) {
	t.Run("executes test for a single unit and stores result", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		testDefs := []*transformations.TestDefinition{
			{Name: "test1", Input: []any{map[string]any{"type": "track"}}},
		}

		unitTask := &testUnitTask{
			ID:                    "trans-1",
			Name:                  "Test Transformation",
			testDefs:              testDefs,
			transformationVersion: "ver-1",
			libraryVersionIDs:     []string{"lib-ver-1"},
		}

		expectedResponse := &transformations.BatchTestResponse{
			Pass: true,
			ValidationOutput: transformations.ValidationOutput{
				Libraries: []transformations.LibraryTestResult{
					{VersionID: "lib-ver-1", Pass: true},
				},
				Transformations: []transformations.TransformationTestResult{
					{
						ID:        "trans-1",
						VersionID: "ver-1",
						Pass:      true,
						TestSuiteResult: transformations.TestSuiteRunResult{
							Status: transformations.TestRunStatusPass,
							Results: []transformations.TestResult{
								{Name: "test1", Status: transformations.TestRunStatusPass},
							},
						},
					},
				},
			},
		}

		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			assert.Len(t, req.Transformations, 1)
			assert.Equal(t, "ver-1", req.Transformations[0].VersionID)
			assert.Len(t, req.Libraries, 1)
			assert.Equal(t, "lib-ver-1", req.Libraries[0].VersionID)
			return expectedResponse, nil
		}

		runner := &Runner{store: mockStore}
		results := tasker.NewResults[*testUnitResult]()

		err := runner.runTestUnitTask(ctx, results)(unitTask)

		require.NoError(t, err)
	})

	t.Run("returns error when batch test fails", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		unitTask := &testUnitTask{
			ID:                    "trans-1",
			Name:                  "Test Transformation",
			testDefs:              []*transformations.TestDefinition{},
			transformationVersion: "ver-1",
		}

		expectedErr := errors.New("API error")
		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			return nil, expectedErr
		}

		runner := &Runner{store: mockStore}
		results := tasker.NewResults[*testUnitResult]()

		err := runner.runTestUnitTask(ctx, results)(unitTask)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "running tests for trans-1")
	})

	t.Run("returns error for invalid task type", func(t *testing.T) {
		ctx := context.Background()
		runner := &Runner{}
		results := tasker.NewResults[*testUnitResult]()

		invalidTask := &libraryVersionTask{}

		err := runner.runTestUnitTask(ctx, results)(invalidTask)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "task is not a test unit task")
	})
}

// --- resolveAllTransformationVersions ---

func TestResolveAllTransformationVersions(t *testing.T) {
	t.Run("resolves versions for all transformations in test plan", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		graph := resources.NewGraph()
		transformation1 := &model.TransformationResource{
			ID:   "trans-1",
			Name: "Transformation 1",
			Code: "code1",
		}
		transformation2 := &model.TransformationResource{
			ID:   "trans-2",
			Name: "Transformation 2",
			Code: "code2",
		}

		planner := NewPlanner(graph)
		runner := &Runner{
			store:       mockStore,
			planner:     planner,
			workspaceID: "ws-123",
		}

		testPlan := &TestPlan{
			TestUnits: []*TestUnit{
				{Transformation: transformation1},
				{Transformation: transformation2},
			},
			ModifiedTransformationURNs: map[string]bool{
				resources.URN("trans-1", "transformation"): true,
			},
		}

		remoteState := newTestState()
		remoteState.AddResource(&state.ResourceState{
			ID:        "trans-2",
			Type:      "transformation",
			OutputRaw: &model.TransformationState{ID: "remote-trans-2", VersionID: "ver-2-existing"},
		})

		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return &transformations.Transformation{VersionID: "ver-1-new"}, nil
		}

		mockStore.updateTransformation = func(ctx context.Context, id string, req *transformations.UpdateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return &transformations.Transformation{VersionID: "ver-2-existing"}, nil
		}

		urnToRemoteID := map[string]string{
			resources.URN("trans-2", "transformation"): "remote-trans-2",
		}

		versionMap, err := runner.resolveAllTransformationVersions(ctx, testPlan, urnToRemoteID)

		require.NoError(t, err)
		assert.Equal(t, "ver-1-new", versionMap["trans-1"])
		assert.Equal(t, "ver-2-existing", versionMap["trans-2"])
	})

	t.Run("deduplicates transformations by ID", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		transformation := &model.TransformationResource{
			ID:   "trans-1",
			Name: "Transformation",
			Code: "code",
		}

		testPlan := &TestPlan{
			TestUnits: []*TestUnit{
				{Transformation: transformation},
				{Transformation: transformation}, // Same transformation twice
			},
			ModifiedTransformationURNs: map[string]bool{
				resources.URN("trans-1", "transformation"): true,
			},
		}

		callCount := 0
		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			callCount++
			return &transformations.Transformation{VersionID: "ver-1"}, nil
		}

		graph := resources.NewGraph()
		planner := NewPlanner(graph)
		runner := &Runner{
			store:   mockStore,
			planner: planner,
		}

		urnToRemoteID := map[string]string{}

		versionMap, err := runner.resolveAllTransformationVersions(ctx, testPlan, urnToRemoteID)

		require.NoError(t, err)
		assert.Len(t, versionMap, 1)
		assert.Equal(t, "ver-1", versionMap["trans-1"])
		assert.Equal(t, 1, callCount, "CreateTransformation should be called only once for duplicate transformation")
	})
}

// --- resolveAllLibraryVersions ---

func TestResolveAllLibraryVersions(t *testing.T) {
	t.Run("resolves versions for all unique libraries", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		lib1 := &model.LibraryResource{ID: "lib-1", Name: "Lib 1", Code: "code1"}
		lib2 := &model.LibraryResource{ID: "lib-2", Name: "Lib 2", Code: "code2"}
		lib3 := &model.LibraryResource{ID: "lib-3", Name: "Lib 3", Code: "code3"}

		testPlan := &TestPlan{
			TestUnits: []*TestUnit{
				{Libraries: []*model.LibraryResource{lib1, lib2}},
				{Libraries: []*model.LibraryResource{lib2, lib3}}, // lib2 duplicated
			},
			StandaloneLibraries: []*model.LibraryResource{lib3}, // lib3 duplicated
			ModifiedLibraryURNs: map[string]bool{
				resources.URN("lib-1", "transformation-library"): true,
			},
		}

		remoteState := newTestState()
		remoteState.AddResource(&state.ResourceState{
			ID:        "lib-2",
			Type:      "transformation-library",
			OutputRaw: &model.LibraryState{ID: "remote-lib-2", VersionID: "ver-2-existing"},
		})
		remoteState.AddResource(&state.ResourceState{
			ID:        "lib-3",
			Type:      "transformation-library",
			OutputRaw: &model.LibraryState{ID: "remote-lib-3", VersionID: "ver-3-existing"},
		})

		mockStore.createLibraryFunc = func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			return &transformations.TransformationLibrary{VersionID: "ver-1-new"}, nil
		}

		mockStore.updateLibraryFunc = func(ctx context.Context, id string, req *transformations.UpdateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			if id == "remote-lib-2" {
				return &transformations.TransformationLibrary{VersionID: "ver-2-existing"}, nil
			}
			if id == "remote-lib-3" {
				return &transformations.TransformationLibrary{VersionID: "ver-3-existing"}, nil
			}
			return nil, errors.New("unexpected library ID")
		}

		graph := resources.NewGraph()
		planner := NewPlanner(graph)
		runner := &Runner{
			store:   mockStore,
			planner: planner,
		}

		urnToRemoteID := map[string]string{
			resources.URN("lib-2", "transformation-library"): "remote-lib-2",
			resources.URN("lib-3", "transformation-library"): "remote-lib-3",
		}

		versionMap, err := runner.resolveAllLibraryVersions(ctx, testPlan, urnToRemoteID)

		require.NoError(t, err)
		assert.Len(t, versionMap, 3)
		assert.Equal(t, "ver-1-new", versionMap["lib-1"])
		assert.Equal(t, "ver-2-existing", versionMap["lib-2"])
		assert.Equal(t, "ver-3-existing", versionMap["lib-3"])
	})
}

// --- testStandaloneLibraries ---

func TestTestStandaloneLibraries(t *testing.T) {
	t.Run("returns nil when no libraries provided", func(t *testing.T) {
		ctx := context.Background()
		runner := &Runner{}

		libResults, trResults, err := runner.testStandaloneLibraries(ctx, nil, nil)

		require.NoError(t, err)
		assert.Nil(t, libResults)
		assert.Nil(t, trResults)
	})

	t.Run("returns nil when library version map is empty", func(t *testing.T) {
		ctx := context.Background()
		runner := &Runner{}

		libs := []*model.LibraryResource{{ID: "lib-1"}}
		libResults, trResults, err := runner.testStandaloneLibraries(ctx, libs, map[string]string{})

		require.NoError(t, err)
		assert.Nil(t, libResults)
		assert.Nil(t, trResults)
	})

	t.Run("executes batch test for standalone libraries", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		runner := &Runner{
			store: mockStore,
		}

		libs := []*model.LibraryResource{
			{ID: "lib-1", Name: "Library 1"},
			{ID: "lib-2", Name: "Library 2"},
		}

		versionMap := map[string]string{
			"lib-1": "ver-1",
			"lib-2": "ver-2",
		}

		expectedResponse := &transformations.BatchTestResponse{
			Pass: true,
			ValidationOutput: transformations.ValidationOutput{
				Libraries: []transformations.LibraryTestResult{
					{VersionID: "ver-1", Pass: true},
					{VersionID: "ver-2", Pass: true},
				},
				Transformations: []transformations.TransformationTestResult{
					{ID: "trans-remote", VersionID: "ver-remote", Pass: true},
				},
			},
		}

		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			assert.Len(t, req.Libraries, 2)
			assert.Equal(t, "ver-1", req.Libraries[0].VersionID)
			assert.Equal(t, "ver-2", req.Libraries[1].VersionID)
			return expectedResponse, nil
		}

		libResults, trResults, err := runner.testStandaloneLibraries(ctx, libs, versionMap)

		require.NoError(t, err)
		assert.Len(t, libResults, 2)
		assert.Len(t, trResults, 1)
		assert.Equal(t, "ver-1", libResults[0].VersionID)
		assert.Equal(t, "trans-remote", trResults[0].Result.ID)
	})

	t.Run("returns error when batch test fails", func(t *testing.T) {
		ctx := context.Background()
		mockStore := newMockStore()

		runner := &Runner{
			store: mockStore,
		}

		libs := []*model.LibraryResource{{ID: "lib-1"}}
		versionMap := map[string]string{"lib-1": "ver-1"}

		expectedErr := errors.New("batch test failed")
		mockStore.batchTestFunc = func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error) {
			return nil, expectedErr
		}

		libResults, trResults, err := runner.testStandaloneLibraries(ctx, libs, versionMap)

		require.Error(t, err)
		assert.Nil(t, libResults)
		assert.Nil(t, trResults)
		assert.Contains(t, err.Error(), "running standalone library tests")
	})
}

// --- buildRemoteIDByURN ---

func TestBuildRemoteIDByURN(t *testing.T) {
	runner := &Runner{}

	t.Run("extracts remoteID from source graph import metadata", func(t *testing.T) {
		sourceGraph := resources.NewGraph()
		transResource := resources.NewResource(
			"t1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{ID: "t1"}),
			resources.WithResourceImportMetadata("remote-t1", ""),
		)
		sourceGraph.AddResource(transResource)

		remoteState := &state.State{Resources: map[string]*state.ResourceState{}}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Equal(t, "remote-t1", result["transformation:t1"])
	})

	t.Run("extracts remoteID from remote state transformation", func(t *testing.T) {
		sourceGraph := resources.NewGraph()

		remoteState := &state.State{
			Resources: map[string]*state.ResourceState{
				"transformation:t2": {
					Type:      "transformation",
					ID:        "t2",
					OutputRaw: &model.TransformationState{ID: "remote-t2"},
				},
			},
		}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Equal(t, "remote-t2", result["transformation:t2"])
	})

	t.Run("extracts remoteID from remote state library", func(t *testing.T) {
		sourceGraph := resources.NewGraph()

		remoteState := &state.State{
			Resources: map[string]*state.ResourceState{
				"transformation-library:lib1": {
					Type:      "transformation-library",
					ID:        "lib1",
					OutputRaw: &model.LibraryState{ID: "remote-lib1"},
				},
			},
		}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Equal(t, "remote-lib1", result["transformation-library:lib1"])
	})

	t.Run("skips source graph resources with empty remoteID", func(t *testing.T) {
		sourceGraph := resources.NewGraph()
		transResource := resources.NewResource(
			"t1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{ID: "t1"}),
			resources.WithResourceImportMetadata("", ""),
		)
		sourceGraph.AddResource(transResource)

		remoteState := &state.State{Resources: map[string]*state.ResourceState{}}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Empty(t, result)
	})

	t.Run("includes remote state resources with empty ID", func(t *testing.T) {
		sourceGraph := resources.NewGraph()

		remoteState := &state.State{
			Resources: map[string]*state.ResourceState{
				"transformation:t1": {
					Type:      "transformation",
					ID:        "t1",
					OutputRaw: &model.TransformationState{ID: ""},
				},
			},
		}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Len(t, result, 1)
		assert.Equal(t, "", result["transformation:t1"], "empty ID should be stored in the map")
	})

	t.Run("ignores non-transformation and non-library resources", func(t *testing.T) {
		sourceGraph := resources.NewGraph()
		otherResource := resources.NewResource(
			"other1",
			"some-other-type",
			resources.ResourceData{},
			nil,
			resources.WithResourceImportMetadata("remote-other", ""),
		)
		sourceGraph.AddResource(otherResource)

		remoteState := &state.State{Resources: map[string]*state.ResourceState{}}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Empty(t, result)
	})

	t.Run("handles mixed transformations and libraries", func(t *testing.T) {
		sourceGraph := resources.NewGraph()
		trans1 := resources.NewResource(
			"t1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{ID: "t1"}),
			resources.WithResourceImportMetadata("remote-t1", ""),
		)
		sourceGraph.AddResource(trans1)

		lib1 := resources.NewResource(
			"lib1",
			"transformation-library",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.LibraryResource{ID: "lib1"}),
			resources.WithResourceImportMetadata("remote-lib1", ""),
		)
		sourceGraph.AddResource(lib1)

		remoteState := &state.State{
			Resources: map[string]*state.ResourceState{
				"transformation:t2": {
					Type:      "transformation",
					ID:        "t2",
					OutputRaw: &model.TransformationState{ID: "remote-t2"},
				},
				"transformation-library:lib2": {
					Type:      "transformation-library",
					ID:        "lib2",
					OutputRaw: &model.LibraryState{ID: "remote-lib2"},
				},
			},
		}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Equal(t, map[string]string{
			"transformation:t1":           "remote-t1",
			"transformation-library:lib1": "remote-lib1",
			"transformation:t2":           "remote-t2",
			"transformation-library:lib2": "remote-lib2",
		}, result)
	})

	t.Run("source graph import metadata takes precedence over remote state", func(t *testing.T) {
		sourceGraph := resources.NewGraph()
		trans1 := resources.NewResource(
			"t1",
			"transformation",
			resources.ResourceData{},
			nil,
			resources.WithRawData(&model.TransformationResource{ID: "t1"}),
			resources.WithResourceImportMetadata("import-t1", ""),
		)
		sourceGraph.AddResource(trans1)

		remoteState := &state.State{
			Resources: map[string]*state.ResourceState{
				"transformation:t1": {
					Type:      "transformation",
					ID:        "t1",
					OutputRaw: &model.TransformationState{ID: "remote-t1"},
				},
			},
		}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Equal(t, "import-t1", result["transformation:t1"])
	})

	t.Run("returns empty map when both are empty", func(t *testing.T) {
		sourceGraph := resources.NewGraph()
		remoteState := &state.State{Resources: map[string]*state.ResourceState{}}

		result := runner.buildRemoteIDByURN(sourceGraph, remoteState)

		assert.Empty(t, result)
	})
}
