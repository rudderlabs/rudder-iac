package testorchestrator

import (
	"context"
	"fmt"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	transformationsprovider "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
)

var runnerLog = logger.New("testorchestrator", logger.Attr{
	Key:   "component",
	Value: "runner",
})

// Runner orchestrates the entire test execution flow
type Runner struct {
	deps         app.Deps
	provider     *transformationsprovider.Provider
	graph        *resources.Graph
	store        transformations.TransformationStore
	planner      *Planner
	inputResolver *InputResolver
	stagingMgr   *StagingManager
	apiClient    *APIClient
	workspaceID  string
}

// NewRunner creates a new test runner
func NewRunner(deps app.Deps, provider *transformationsprovider.Provider, graph *resources.Graph, workspaceID string) *Runner {
	store := transformations.NewRudderTransformationStore(deps.Client())

	return &Runner{
		deps:         deps,
		provider:     provider,
		graph:        graph,
		store:        store,
		planner:      NewPlanner(graph),
		inputResolver: NewInputResolver(),
		stagingMgr:   NewStagingManager(store),
		apiClient:    NewAPIClient(store),
		workspaceID:  workspaceID,
	}
}

// TestResults contains the results of all test executions
type TestResults struct {
	Transformations []TransformationTestResult
}

// HasFailures checks if any tests failed or errored
func (r *TestResults) HasFailures() bool {
	for _, tr := range r.Transformations {
		for _, testResult := range tr.Result.Tests {
			if testResult.Status == TestRunStatusFail || testResult.Status == TestRunStatusError {
				return true
			}
		}
	}
	return false
}

// Run executes tests based on the specified mode and returns results
func (r *Runner) Run(ctx context.Context, mode Mode, targetID string) (*TestResults, error) {
	runnerLog.Info("Starting test run", "mode", mode, "targetID", targetID)

	// Load remote resources
	runnerLog.Debug("Loading remote resources")
	remoteResources, err := r.provider.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	// Build remote state
	runnerLog.Debug("Building remote state")
	remoteState, err := r.provider.MapRemoteToState(remoteResources)
	if err != nil {
		return nil, fmt.Errorf("building remote state: %w", err)
	}

	// Convert remote state to graph for diffing
	remoteGraph := syncer.StateToGraph(remoteState)

	// Build test plan
	runnerLog.Debug("Building test plan")
	testPlan, err := r.planner.BuildPlan(ctx, remoteGraph, mode, targetID, r.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("building test plan: %w", err)
	}

	if len(testPlan.TestUnits) == 0 {
		runnerLog.Info("No transformations to test")
		return &TestResults{Transformations: []TransformationTestResult{}}, nil
	}

	runnerLog.Info("Test plan created", "testUnits", len(testPlan.TestUnits))

	// Execute tests for each test unit
	var allResults []TransformationTestResult
	for _, unit := range testPlan.TestUnits {
		runnerLog.Info("Testing transformation", "id", unit.Transformation.ID, "name", unit.Transformation.Name)

		// Resolve test cases
		testCases, err := r.inputResolver.ResolveTestCases(unit.Transformation)
		if err != nil {
			return nil, fmt.Errorf("resolving test cases for %s: %w", unit.Transformation.ID, err)
		}

		runnerLog.Debug("Resolved test cases", "transformation", unit.Transformation.ID, "count", len(testCases))

		// Resolve library versions (upload modified libraries, reuse existing for unmodified)
		libraryVersionMap, err := r.resolveLibraryVersions(ctx, unit, remoteState)
		if err != nil {
			return nil, fmt.Errorf("resolving library versions for %s: %w", unit.Transformation.ID, err)
		}

		// Build test request
		testReq := r.buildTestRequest(unit, testCases, libraryVersionMap)

		// Run tests via API
		runnerLog.Debug("Executing tests via API", "transformation", unit.Transformation.ID)
		result, err := r.apiClient.RunTests(ctx, testReq)
		if err != nil {
			return nil, fmt.Errorf("running tests for %s: %w", unit.Transformation.ID, err)
		}

		allResults = append(allResults, *result)
	}

	return &TestResults{Transformations: allResults}, nil
}

// resolveLibraryVersions resolves library versionIDs for a test unit
// Modified libraries are uploaded as unpublished versions, unmodified libraries reuse existing versionIDs
func (r *Runner) resolveLibraryVersions(ctx context.Context, unit *TestUnit, remoteState *state.State) (map[string]string, error) {
	versionMap := make(map[string]string)

	for _, lib := range unit.Libraries {
		libURN := fmt.Sprintf("%s::%s", lib.ID, "library")

		// Check if library is modified
		if unit.IsLibraryModified(libURN) {
			// Upload library (create or update as unpublished)
			runnerLog.Debug("Uploading modified library", "library", lib.ID)
			versionID, err := r.stagingMgr.UploadLibrary(ctx, lib, remoteState)
			if err != nil {
				return nil, fmt.Errorf("uploading library %s: %w", lib.ID, err)
			}
			versionMap[lib.ImportName] = versionID
		} else {
			// Reuse existing versionID from remote state
			remoteResource := remoteState.GetResource(libURN)
			if remoteResource == nil {
				return nil, fmt.Errorf("unmodified library %s not found in remote state", lib.ID)
			}

			versionID, ok := remoteResource.Output["versionId"].(string)
			if !ok || versionID == "" {
				return nil, fmt.Errorf("library %s in remote state has no valid versionId", lib.ID)
			}

			runnerLog.Debug("Reusing existing library version", "library", lib.ID, "versionId", versionID)
			versionMap[lib.ImportName] = versionID
		}
	}

	return versionMap, nil
}

// buildTestRequest constructs the batch test API request
func (r *Runner) buildTestRequest(unit *TestUnit, testCases []TestCase, libraryVersionMap map[string]string) *BatchTestRequest {
	// Build test definitions from test cases
	testDefinitions := make([]TestDefinition, len(testCases))
	for i, tc := range testCases {
		testDefinitions[i] = TestDefinition{
			Name:          tc.Name,
			Input:         tc.InputEvents,
			ExpectedOutput: tc.ExpectedOutput,
		}
	}

	// Build library inputs from version map
	var libraryInputs []TransformationLibraryInput
	for _, versionID := range libraryVersionMap {
		libraryInputs = append(libraryInputs, TransformationLibraryInput{
			VersionID: versionID,
		})
	}

	// Build request
	return &BatchTestRequest{
		Transformation: MultiTransformationTestInput{
			Code:        unit.Transformation.Code,
			Language:    unit.Transformation.Language,
			Tests:       testDefinitions,
			LibraryTags: getVersionIDs(libraryInputs),
		},
		Libraries: libraryInputs,
	}
}

// getVersionIDs extracts versionIDs from library inputs as an array
func getVersionIDs(libraries []TransformationLibraryInput) []string {
	versionIDs := make([]string, len(libraries))
	for i, lib := range libraries {
		versionIDs[i] = lib.VersionID
	}
	return versionIDs
}
