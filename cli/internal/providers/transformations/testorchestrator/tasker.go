package testorchestrator

import (
	"context"
	"errors"
	"fmt"

	transformations "github.com/rudderlabs/rudder-iac/api/client/transformations"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

const concurrency = 2

type libraryVersionTask struct {
	lib        *model.LibraryResource
	isModified bool
	remoteID   string
}

func (t *libraryVersionTask) Id() string {
	return t.lib.ID
}

func (t *libraryVersionTask) Dependencies() []string {
	return nil
}

type transformationVersionTask struct {
	transformation *model.TransformationResource
	isModified     bool
	remoteID       string
}

func (t *transformationVersionTask) Id() string {
	return t.transformation.ID
}

func (t *transformationVersionTask) Dependencies() []string {
	return nil
}

// testUnitResult holds the API response for a single test unit execution
type testUnitResult struct {
	Transformations []*TransformationTestWithDefinitions
	Libraries       []transformations.LibraryTestResult
}

type testUnitTask struct {
	ID                    string
	Name                  string
	testDefs              []*transformations.TestDefinition
	transformationVersion string
	libraryVersionIDs     []string
}

func (t *testUnitTask) Id() string {
	return t.ID
}

func (t *testUnitTask) Dependencies() []string {
	return nil
}

func runTransformationVersionTasks(
	ctx context.Context,
	store transformations.TransformationStore,
	tasks []*transformationVersionTask,
) (map[string]string, error) {
	taskerTasks := make([]tasker.Task, 0, len(tasks))
	for _, t := range tasks {
		taskerTasks = append(taskerTasks, t)
	}

	results := tasker.NewResults[string]()
	errs := tasker.RunTasks(
		ctx,
		taskerTasks,
		concurrency,
		false,
		runTransformationVersionTask(ctx, store, results),
	)
	if len(errs) > 0 {
		return nil, fmt.Errorf("running transformation version tasks: %w", errors.Join(errs...))
	}

	versionMap := make(map[string]string, len(tasks))
	for _, key := range results.GetKeys() {
		versionID, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("transformation %s version not found in task results", key)
		}
		versionMap[key] = versionID
	}

	return versionMap, nil
}

func runTransformationVersionTask(
	ctx context.Context,
	store transformations.TransformationStore,
	results *tasker.Results[string],
) func(task tasker.Task) error {
	return func(task tasker.Task) error {
		transformationTask, ok := task.(*transformationVersionTask)
		if !ok {
			return fmt.Errorf("task is not a transformation version task")
		}

		versionID, err := getTransformationVersionID(
			ctx,
			store,
			transformationTask.transformation,
			transformationTask.isModified,
			transformationTask.remoteID,
		)
		if err != nil {
			return fmt.Errorf("resolving transformation version for %s: %w", transformationTask.transformation.ID, err)
		}

		results.Store(transformationTask.Id(), versionID)
		return nil
	}
}

func getTransformationVersionID(
	ctx context.Context,
	store transformations.TransformationStore,
	transformation *model.TransformationResource,
	isModified bool,
	remoteID string,
) (string, error) {
	if !isModified && remoteID == "" {
		return "", fmt.Errorf("unmodified transformation %s not found in remote state", transformation.ID)
	}

	testLogger.Debug("Staging transformation", "transformation", transformation.ID, "isModified", isModified)
	versionID, err := StageTransformation(ctx, store, transformation, remoteID)
	if err != nil {
		return "", fmt.Errorf("staging transformation %s: %w", transformation.ID, err)
	}

	return versionID, nil
}

func runLibraryVersionTasks(
	ctx context.Context,
	store transformations.TransformationStore,
	libraryTasks []*libraryVersionTask,
) (map[string]string, error) {
	tasks := make([]tasker.Task, 0, len(libraryTasks))
	for _, libraryTask := range libraryTasks {
		tasks = append(tasks, libraryTask)
	}

	results := tasker.NewResults[string]()
	errs := tasker.RunTasks(
		ctx,
		tasks,
		concurrency,
		false,
		runLibraryVersionTask(ctx, store, results),
	)
	if len(errs) > 0 {
		return nil, fmt.Errorf("running library version tasks: %w", errors.Join(errs...))
	}

	versionMap := make(map[string]string, len(libraryTasks))
	for _, key := range results.GetKeys() {
		versionID, ok := results.Get(key)
		if !ok {
			return nil, fmt.Errorf("library %s version not found in task results", key)
		}
		versionMap[key] = versionID
	}

	return versionMap, nil
}

func runLibraryVersionTask(
	ctx context.Context,
	store transformations.TransformationStore,
	results *tasker.Results[string],
) func(task tasker.Task) error {
	return func(task tasker.Task) error {
		libraryTask, ok := task.(*libraryVersionTask)
		if !ok {
			return fmt.Errorf("task is not a library version task")
		}

		versionID, err := getLibraryVersionID(
			ctx,
			store,
			libraryTask.lib,
			libraryTask.isModified,
			libraryTask.remoteID,
		)
		if err != nil {
			return fmt.Errorf("resolving library version for %s: %w", libraryTask.lib.ID, err)
		}

		results.Store(libraryTask.Id(), versionID)
		return nil
	}
}

func getLibraryVersionID(
	ctx context.Context,
	store transformations.TransformationStore,
	library *model.LibraryResource,
	isModified bool,
	remoteID string,
) (string, error) {
	if !isModified && remoteID == "" {
		return "", fmt.Errorf("unmodified library %s not found in remote state", library.ID)
	}

	testLogger.Debug("Staging library", "library", library.ID, "isModified", isModified)
	versionID, err := StageLibrary(ctx, store, library, remoteID)
	if err != nil {
		return "", fmt.Errorf("staging library %s: %w", library.ID, err)
	}

	return versionID, nil
}
