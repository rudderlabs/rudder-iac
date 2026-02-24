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

type mockTransformationStore struct {
	batchTestFunc        func(ctx context.Context, req *transformations.BatchTestRequest) (*transformations.BatchTestResponse, error)
	createTransformation func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error)
	createLibraryFunc    func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error)
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
			OutputRaw: &model.TransformationState{VersionID: "ver-2-existing"},
		})

		mockStore.createTransformation = func(ctx context.Context, req *transformations.CreateTransformationRequest, publish bool) (*transformations.Transformation, error) {
			return &transformations.Transformation{VersionID: "ver-1-new"}, nil
		}

		versionMap, err := runner.resolveAllTransformationVersions(ctx, testPlan, remoteState)

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

		remoteState := newTestState()

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

		versionMap, err := runner.resolveAllTransformationVersions(ctx, testPlan, remoteState)

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
			OutputRaw: &model.LibraryState{VersionID: "ver-2-existing"},
		})
		remoteState.AddResource(&state.ResourceState{
			ID:        "lib-3",
			Type:      "transformation-library",
			OutputRaw: &model.LibraryState{VersionID: "ver-3-existing"},
		})

		mockStore.createLibraryFunc = func(ctx context.Context, req *transformations.CreateLibraryRequest, publish bool) (*transformations.TransformationLibrary, error) {
			return &transformations.TransformationLibrary{VersionID: "ver-1-new"}, nil
		}

		graph := resources.NewGraph()
		planner := NewPlanner(graph)
		runner := &Runner{
			store:   mockStore,
			planner: planner,
		}

		versionMap, err := runner.resolveAllLibraryVersions(ctx, testPlan, remoteState)

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
