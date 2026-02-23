package testorchestrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/rudderlabs/rudder-iac/api/client"
	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/logger"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

// TransformationTestWithDefinitions combines test results with their original definitions
type TransformationTestWithDefinitions struct {
	Result      *transformations.TransformationTestResult
	Definitions []*transformations.TestDefinition
}

// TestResults contains the results of all test executions with their definitions
type TestResults struct {
	Pass            bool
	Message         string
	Libraries       []transformations.LibraryTestResult
	Transformations []*TransformationTestWithDefinitions
}

var testLogger = logger.New("testorchestrator")

// remoteStateLoader abstracts provider methods needed by the runner
type remoteStateLoader interface {
	LoadResourcesFromRemote(ctx context.Context) (*resources.RemoteResources, error)
	MapRemoteToState(collection *resources.RemoteResources) (*state.State, error)
}

type Runner struct {
	loader      remoteStateLoader
	store       transformations.TransformationStore
	planner     *Planner
	workspaceID string
}

func NewRunner(client *client.Client, loader remoteStateLoader, graph *resources.Graph, workspaceID string) *Runner {
	store := transformations.NewRudderTransformationStore(client)

	return &Runner{
		loader:      loader,
		store:       store,
		planner:     NewPlanner(graph),
		workspaceID: workspaceID,
	}
}

// Run executes tests based on the specified mode and returns results
func (r *Runner) Run(ctx context.Context, mode Mode, targetID string) (*TestResults, error) {
	testLogger.Info("Starting test run", "mode", mode, "targetID", targetID)

	// build remote resources graph
	remoteResources, err := r.loader.LoadResourcesFromRemote(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading remote resources: %w", err)
	}

	remoteState, err := r.loader.MapRemoteToState(remoteResources)
	if err != nil {
		return nil, fmt.Errorf("building remote state: %w", err)
	}

	remoteGraph := syncer.StateToGraph(remoteState)

	testLogger.Debug("Building test plan")
	testPlan, err := r.planner.BuildPlan(ctx, remoteGraph, mode, targetID, r.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("building test plan: %w", err)
	}

	if len(testPlan.TestUnits) == 0 && len(testPlan.StandaloneLibraries) == 0 {
		testLogger.Info("No resources to test")
		return &TestResults{Pass: true}, nil
	}

	testLogger.Info("Test plan created", "testUnits", len(testPlan.TestUnits), "standaloneLibraries", len(testPlan.StandaloneLibraries))

	// Resolve library and transformation versions once before parallel execution
	libraryVersionMap, err := r.resolveAllLibraryVersions(ctx, testPlan, remoteState)
	if err != nil {
		return nil, fmt.Errorf("resolving library versions: %w", err)
	}

	transformationVersionMap, err := r.resolveAllTransformationVersions(ctx, testPlan, remoteState)
	if err != nil {
		return nil, fmt.Errorf("resolving transformation versions: %w", err)
	}

	// Build test unit tasks with pre-resolved versions and test definitions
	unitTasks := make([]*testUnitTask, 0, len(testPlan.TestUnits))
	for _, unit := range testPlan.TestUnits {
		testDefs, err := ResolveTestDefinitions(unit.Transformation)
		if err != nil {
			return nil, fmt.Errorf("resolving test definitions for %s: %w", unit.Transformation.ID, err)
		}
		testLogger.Debug("Resolved test definitions", "transformation", unit.Transformation.ID, "count", len(testDefs))

		transformationID := unit.Transformation.ID
		transformationVersionID, ok := transformationVersionMap[transformationID]
		if !ok {
			return nil, fmt.Errorf("transformation version not resolved for %s", transformationID)
		}

		unitTasks = append(unitTasks, &testUnitTask{
			ID:                    transformationID,
			Name:                  unit.Transformation.Name,
			testDefs:              testDefs,
			transformationVersion: transformationVersionID,
			libraryVersionIDs:     getVersionIDsForUnitLibraries(unit.Libraries, libraryVersionMap),
		})
	}

	unitResults, errs := r.runTestUnitTasks(ctx, unitTasks)
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	var (
		allResults []*TransformationTestWithDefinitions
		allLibs    []transformations.LibraryTestResult
	)
	for _, ur := range unitResults {
		allResults = append(allResults, ur.Transformations...)
		allLibs = append(allLibs, ur.Libraries...)
	}

	// Execute standalone library tests (libraries not connected to any transformation)
	if len(testPlan.StandaloneLibraries) > 0 {
		libResults, trResults, err := r.testStandaloneLibraries(ctx, testPlan.StandaloneLibraries, libraryVersionMap)
		if err != nil {
			return nil, fmt.Errorf("testing standalone libraries: %w", err)
		}
		allLibs = append(allLibs, libResults...)
		allResults = append(allResults, trResults...)
	}

	results := &TestResults{
		Libraries:       allLibs,
		Transformations: allResults,
	}

	return results, nil
}

// resolveAllTransformationVersions resolves versionIDs for all unique transformations in the test plan.
// Modified transformations are staged as unpublished versions; unmodified ones reuse existing versionIDs.
// Returns a map of transformation ID to versionID.
func (r *Runner) resolveAllTransformationVersions(ctx context.Context, testPlan *TestPlan, remoteState *state.State) (map[string]string, error) {
	seen := make(map[string]struct{})
	tasks := make([]*transformationVersionTask, 0, len(testPlan.TestUnits))

	for _, unit := range testPlan.TestUnits {
		if _, exists := seen[unit.Transformation.ID]; exists {
			continue
		}
		seen[unit.Transformation.ID] = struct{}{}

		transformationURN := resources.URN(unit.Transformation.ID, "transformation")
		remoteResource := remoteState.GetResource(transformationURN)
		isModified := testPlan.IsTransformationModified(transformationURN)

		tasks = append(tasks, &transformationVersionTask{
			transformation: unit.Transformation,
			isModified:     isModified,
			remoteResource: remoteResource,
		})
	}

	return runTransformationVersionTasks(ctx, r.store, tasks)
}

// resolveAllLibraryVersions fetches versionIDs for all unique libraries in the test plan,
// including both transformation-dependent and standalone libraries.
// Returns a map of library ID to versionID.
func (r *Runner) resolveAllLibraryVersions(ctx context.Context, testPlan *TestPlan, remoteState *state.State) (map[string]string, error) {
	allLibs := lo.FlatMap(testPlan.TestUnits, func(u *TestUnit, _ int) []*model.LibraryResource {
		return u.Libraries
	})
	allLibs = append(allLibs, testPlan.StandaloneLibraries...)

	uniqueLibraries := make(map[string]struct{})
	libraryTasks := make([]*libraryVersionTask, 0, len(allLibs))

	for _, lib := range allLibs {
		if _, exists := uniqueLibraries[lib.ID]; exists {
			continue
		}
		uniqueLibraries[lib.ID] = struct{}{}

		libURN := resources.URN(lib.ID, "transformation-library")
		remoteResource := remoteState.GetResource(libURN)
		isModified := testPlan.IsLibraryModified(libURN)

		libraryTasks = append(libraryTasks, &libraryVersionTask{
			lib:            lib,
			isModified:     isModified,
			remoteResource: remoteResource,
		})
	}

	return runLibraryVersionTasks(ctx, r.store, libraryTasks)
}

// getLibraryVersionsForUnit extracts library versionIDs for a specific test unit
func getVersionIDsForUnitLibraries(libraries []*model.LibraryResource, libraryVersionMap map[string]string) []string {
	return lo.FilterMap(libraries, func(lib *model.LibraryResource, _ int) (string, bool) {
		versionID, exists := libraryVersionMap[lib.ID]
		return versionID, exists
	})
}

// buildTestRequest constructs the batch test API request
func buildTestRequest(transformationVersionID string, testDefs []*transformations.TestDefinition, libraryVersionIDs []string) *transformations.BatchTestRequest {
	libraryInputs := lo.Map(libraryVersionIDs, func(vid string, _ int) transformations.LibraryTestInput {
		return transformations.LibraryTestInput{VersionID: vid}
	})

	// Convert pointers to values for JSON serialization
	testSuite := lo.Map(testDefs, func(td *transformations.TestDefinition, _ int) transformations.TestDefinition {
		return *td
	})

	return &transformations.BatchTestRequest{
		Transformations: []transformations.TransformationTestInput{
			{
				VersionID: transformationVersionID,
				TestSuite: testSuite,
			},
		},
		Libraries: libraryInputs,
	}
}

// runTestUnitTasks executes all test units concurrently via the tasker framework.
// Returns per-unit results and any errors encountered.
func (r *Runner) runTestUnitTasks(ctx context.Context, unitTasks []*testUnitTask) ([]*testUnitResult, []error) {
	tasks := make([]tasker.Task, 0, len(unitTasks))
	for _, t := range unitTasks {
		tasks = append(tasks, t)
	}

	results := tasker.NewResults[*testUnitResult]()
	errs := tasker.RunTasks(
		ctx,
		tasks,
		concurrency,
		true, // continue on failure so all units run
		r.runTestUnitTask(ctx, results),
	)

	unitResults := make([]*testUnitResult, 0, len(unitTasks))
	for _, key := range results.GetKeys() {
		result, ok := results.Get(key)
		if ok {
			unitResults = append(unitResults, result)
		}
	}

	return unitResults, errs
}

func (r *Runner) runTestUnitTask(ctx context.Context, results *tasker.Results[*testUnitResult]) func(task tasker.Task) error {
	return func(task tasker.Task) error {
		unitTask, ok := task.(*testUnitTask)
		if !ok {
			return fmt.Errorf("task is not a test unit task")
		}

		testLogger.Info("Testing transformation", "id", unitTask.ID, "name", unitTask.Name)

		testReq := buildTestRequest(unitTask.transformationVersion, unitTask.testDefs, unitTask.libraryVersionIDs)

		testLogger.Debug("Executing tests via API", "transformation", unitTask.ID)
		resp, err := r.store.BatchTest(ctx, testReq)
		if err != nil {
			return fmt.Errorf("running tests for %s: %w", unitTask.ID, err)
		}

		result := &testUnitResult{
			Libraries: resp.ValidationOutput.Libraries,
		}
		for i := range resp.ValidationOutput.Transformations {
			result.Transformations = append(result.Transformations, &TransformationTestWithDefinitions{
				Result:      &resp.ValidationOutput.Transformations[i],
				Definitions: unitTask.testDefs,
			})
		}

		results.Store(unitTask.Id(), result)
		return nil
	}
}

// testStandaloneLibraries executes a batch test for libraries not connected to any transformation.
// The API may return transformation results for remote transformations connected to these libraries.
func (r *Runner) testStandaloneLibraries(ctx context.Context, libs []*model.LibraryResource, libraryVersionMap map[string]string) ([]transformations.LibraryTestResult, []*TransformationTestWithDefinitions, error) {
	libInputs := lo.FilterMap(libs, func(lib *model.LibraryResource, _ int) (transformations.LibraryTestInput, bool) {
		versionID, exists := libraryVersionMap[lib.ID]
		return transformations.LibraryTestInput{VersionID: versionID}, exists
	})

	if len(libInputs) == 0 {
		return nil, nil, nil
	}

	testLogger.Info("Testing standalone libraries", "count", len(libInputs))

	resp, err := r.store.BatchTest(ctx, &transformations.BatchTestRequest{
		Libraries: libInputs,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("running standalone library tests: %w", err)
	}

	// Collect transformation results from the API response (remote transformations connected to these libraries)
	var trResults []*TransformationTestWithDefinitions
	for i := range resp.ValidationOutput.Transformations {
		trResults = append(trResults, &TransformationTestWithDefinitions{
			Result: &resp.ValidationOutput.Transformations[i],
		})
	}

	return resp.ValidationOutput.Libraries, trResults, nil
}
