package testorchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/app"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	transformationsprovider "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
)

var log = logger.New("testorchestrator")

// Runner orchestrates the entire test execution flow
type Runner struct {
	deps        app.Deps
	provider    *transformationsprovider.Provider
	graph       *resources.Graph
	store       transformations.TransformationStore
	planner     *Planner
	stagingMgr  *StagingManager
	workspaceID string
}

// NewRunner creates a new test runner
func NewRunner(deps app.Deps, provider *transformationsprovider.Provider, graph *resources.Graph, workspaceID string) *Runner {
	store := transformations.NewRudderTransformationStore(deps.Client())

	return &Runner{
		deps:        deps,
		provider:    provider,
		graph:       graph,
		store:       store,
		planner:     NewPlanner(graph),
		stagingMgr:  NewStagingManager(store),
		workspaceID: workspaceID,
	}
}

// TestResults contains the results of all test executions
type TestResults struct {
	Transformations []*transformations.TransformationTestResult
}

// HasFailures checks if any tests failed or errored
func (r *TestResults) HasFailures() bool {
	for _, tr := range r.Transformations {
		for _, testResult := range tr.TestSuiteResult.Results {
			if testResult.Status == transformations.TestRunStatusFail || testResult.Status == transformations.TestRunStatusError {
				return true
			}
		}
	}
	return false
}

// Run executes tests based on the specified mode and returns results
func (r *Runner) Run(ctx context.Context, mode Mode, targetID string) (*TestResults, error) {
	log.Info("Starting test run", "mode", mode, "targetID", targetID)

	// Load remote resources
	log.Debug("Loading remote resources")
	remoteResources, err := r.provider.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	// Build remote state
	log.Debug("Building remote state")
	remoteState, err := r.provider.MapRemoteToState(remoteResources)
	if err != nil {
		return nil, fmt.Errorf("building remote state: %w", err)
	}

	// Convert remote state to graph for diffing
	remoteGraph := syncer.StateToGraph(remoteState)

	// Build test plan
	log.Debug("Building test plan")
	testPlan, err := r.planner.BuildPlan(ctx, remoteGraph, mode, targetID, r.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("building test plan: %w", err)
	}

	if len(testPlan.TestUnits) == 0 {
		log.Info("No transformations to test")
		return &TestResults{Transformations: []*transformations.TransformationTestResult{}}, nil
	}

	log.Info("Test plan created", "testUnits", len(testPlan.TestUnits))

	// Resolve library versions once for all test units (libraries can be shared)
	libraryVersionMap, err := r.resolveAllLibraryVersions(ctx, testPlan, remoteState)
	if err != nil {
		return nil, fmt.Errorf("resolving library versions: %w", err)
	}

	// Resolve test cases for all units before parallel execution
	unitTestCases := make(map[string][]TestCase)
	for _, unit := range testPlan.TestUnits {
		testCases, err := ResolveTestCases(unit.Transformation)
		if err != nil {
			return nil, fmt.Errorf("resolving test cases for %s: %w", unit.Transformation.ID, err)
		}
		log.Debug("Resolved test cases", "transformation", unit.Transformation.ID, "count", len(testCases))
		unitTestCases[unit.Transformation.ID] = testCases
	}

	// Execute tests in parallel using WaitGroup
	var wg sync.WaitGroup
	resultsMu := sync.Mutex{}
	var allResults []*transformations.TransformationTestResult
	var errs []error
	errorsMutex := sync.Mutex{}

	for _, unit := range testPlan.TestUnits {
		unit := unit // Capture loop variable
		wg.Add(1)
		go func(u *TestUnit) {
			defer wg.Done()

			log.Info("Testing transformation", "id", u.Transformation.ID, "name", u.Transformation.Name)

			// Resolve transformation versionID (upload if modified, reuse existing otherwise)
			transformationVersionID, err := r.resolveTransformationVersion(ctx, u, remoteState)
			if err != nil {
				errorsMutex.Lock()
				errs = append(errs, fmt.Errorf("resolving transformation version for %s: %w", u.Transformation.ID, err))
				errorsMutex.Unlock()
				return
			}

			// Get library versionIDs for this transformation's dependencies
			libraryVersionIDs := r.getLibraryVersionsForUnit(u, libraryVersionMap)

			// Build test request
			testReq := r.buildTestRequest(transformationVersionID, unitTestCases[u.Transformation.ID], libraryVersionIDs)

			// Run tests via API
			log.Debug("Executing tests via API", "transformation", u.Transformation.ID)
			results, err := r.store.BatchTest(ctx, testReq)
			if err != nil {
				errorsMutex.Lock()
				errs = append(errs, fmt.Errorf("running tests for %s: %w", u.Transformation.ID, err))
				errorsMutex.Unlock()
				return
			}

			// Safely append results
			resultsMu.Lock()
			allResults = append(allResults, results...)
			resultsMu.Unlock()
		}(unit)
	}

	// Wait for all tests to complete
	wg.Wait()

	// Check if any errors occurred
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &TestResults{Transformations: allResults}, nil
}

// resolveTransformationVersion resolves the versionID for a transformation
// Modified transformations are uploaded as unpublished versions, unmodified transformations reuse existing versionIDs
func (r *Runner) resolveTransformationVersion(ctx context.Context, unit *TestUnit, remoteState *state.State) (string, error) {
	transformationURN := resources.URN(unit.Transformation.ID, "transformation")

	// Check if transformation is modified
	if unit.IsTransformationModified {
		// Upload transformation (create or update as unpublished)
		log.Debug("Uploading modified transformation", "transformation", unit.Transformation.ID)
		versionID, err := r.stagingMgr.StageTransformation(ctx, unit.Transformation, remoteState)
		if err != nil {
			return "", fmt.Errorf("uploading transformation %s: %w", unit.Transformation.ID, err)
		}
		return versionID, nil
	}

	// Reuse existing versionID from remote state
	remoteResource := remoteState.GetResource(transformationURN)
	if remoteResource == nil {
		return "", fmt.Errorf("unmodified transformation %s not found in remote state", unit.Transformation.ID)
	}

	versionID, ok := remoteResource.Output["versionId"].(string)
	if !ok || versionID == "" {
		return "", fmt.Errorf("transformation %s in remote state has no valid versionId", unit.Transformation.ID)
	}

	log.Debug("Reusing existing transformation version", "transformation", unit.Transformation.ID, "versionId", versionID)
	return versionID, nil
}

type libraryInfo struct {
	lib        *model.LibraryResource
	isModified bool
}

// resolveAllLibraryVersions resolves versionIDs for all unique libraries in the test plan
// Modified libraries are uploaded as unpublished versions, unmodified libraries reuse existing versionIDs
// Returns a map of library ID to versionID
func (r *Runner) resolveAllLibraryVersions(ctx context.Context, testPlan *TestPlan, remoteState *state.State) (map[string]string, error) {
	versionMap := make(map[string]string)

	// Collect all unique libraries from all test units
	uniqueLibraries := make(map[string]*libraryInfo)

	for _, unit := range testPlan.TestUnits {
		for _, lib := range unit.Libraries {
			libURN := resources.URN(lib.ID, "transformation-library")
			if _, exists := uniqueLibraries[lib.ID]; !exists {
				uniqueLibraries[lib.ID] = &libraryInfo{
					lib:        lib,
					isModified: unit.IsLibraryModified(libURN),
				}
			}
		}
	}

	// Resolve each unique library once
	for libID, libInfo := range uniqueLibraries {
		libURN := resources.URN(libID, "transformation-library")

		if libInfo.isModified {
			// Upload library (create or update as unpublished)
			log.Debug("Uploading modified library", "library", libID)
			versionID, err := r.stagingMgr.StageLibrary(ctx, libInfo.lib, remoteState)
			if err != nil {
				return nil, fmt.Errorf("uploading library %s: %w", libID, err)
			}
			versionMap[libID] = versionID
		} else {
			// Reuse existing versionID from remote state
			remoteResource := remoteState.GetResource(libURN)
			if remoteResource == nil {
				return nil, fmt.Errorf("unmodified library %s not found in remote state", libID)
			}

			versionID, ok := remoteResource.Output["versionId"].(string)
			if !ok || versionID == "" {
				return nil, fmt.Errorf("library %s in remote state has no valid versionId", libID)
			}

			log.Debug("Reusing existing library version", "library", libID, "versionId", versionID)
			versionMap[libID] = versionID
		}
	}

	return versionMap, nil
}

// getLibraryVersionsForUnit extracts library versionIDs for a specific test unit
func (r *Runner) getLibraryVersionsForUnit(unit *TestUnit, libraryVersionMap map[string]string) []string {
	var versionIDs []string
	for _, lib := range unit.Libraries {
		if versionID, exists := libraryVersionMap[lib.ID]; exists {
			versionIDs = append(versionIDs, versionID)
		}
	}
	return versionIDs
}

// buildTestRequest constructs the batch test API request using client types
func (r *Runner) buildTestRequest(transformationVersionID string, testCases []TestCase, libraryVersionIDs []string) *transformations.BatchTestRequest {
	// Build test definitions from test cases
	testDefinitions := make([]transformations.TestDefinition, len(testCases))
	for i, tc := range testCases {
		testDefinitions[i] = transformations.TestDefinition{
			Name:           tc.Name,
			Input:          tc.InputEvents,
			ExpectedOutput: tc.ExpectedOutput,
		}
	}

	// Build transformation test input
	transformationInputs := []transformations.TransformationTestInput{
		{
			VersionID: transformationVersionID,
			TestSuite: testDefinitions,
		},
	}

	// Build library inputs from version IDs
	var libraryInputs []transformations.LibraryTestInput
	for _, versionID := range libraryVersionIDs {
		libraryInputs = append(libraryInputs, transformations.LibraryTestInput{
			VersionID: versionID,
		})
	}

	// Build request
	return &transformations.BatchTestRequest{
		Transformations: transformationInputs,
		Libraries:       libraryInputs,
	}
}
