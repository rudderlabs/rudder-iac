package syncer

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/pkg/tasker"
)

type operationTask struct {
	operation        *planner.Operation
	sourceGraph      *resources.Graph
	targetGraph      *resources.Graph
	serialDependency string
}

const (
	transformationResourceType        = "transformation"
	transformationLibraryResourceType = "transformation-library"
)

var serializedOperationTypes = map[string]struct{}{
	transformationResourceType:        {},
	transformationLibraryResourceType: {},
}

func newOperationTasks(plan *planner.Plan, sourceGraph *resources.Graph, targetGraph *resources.Graph) []tasker.Task {
	tasks := make([]tasker.Task, 0, len(plan.Operations))
	var previousSerializedOperation string

	for _, operation := range plan.Operations {
		task := newOperationTask(operation, sourceGraph, targetGraph, previousSerializedOperation)
		tasks = append(tasks, task)

		if _, serialized := serializedOperationTypes[operation.Resource.Type()]; serialized {
			previousSerializedOperation = operation.Resource.URN()
		}
	}

	return tasks
}

func newOperationTask(operation *planner.Operation, sourceGraph *resources.Graph, targetGraph *resources.Graph, serialDependency string) *operationTask {
	return &operationTask{
		operation:        operation,
		sourceGraph:      sourceGraph,
		targetGraph:      targetGraph,
		serialDependency: serialDependency,
	}
}

func (t *operationTask) Id() string {
	return t.operation.Resource.URN()
}

/*
Dependencies are currently defined at the resource level,
which means multiple operations for the same resource
may run concurrently. This behavior may not always be desirable.
*/
func (t *operationTask) Dependencies() []string {
	dependencies := make([]string, 0)

	// For delete operations, we need to invert the dependency order
	// If A depends on B, then for deletion: B should be deleted before A
	if t.operation.Type == planner.Delete {
		// For delete operations, we need to find which resources currently depend on the resource being deleted.
		// This information exists in the source graph. Target graph may not even contain the dependents
		dependencies = append(dependencies, t.sourceGraph.GetDependents(t.operation.Resource.URN())...)
	} else {
		dependencies = append(dependencies, t.targetGraph.GetDependencies(t.operation.Resource.URN())...)
	}

	if t.serialDependency != "" {
		dependencies = appendDependency(dependencies, t.serialDependency)
	}

	return dependencies
}

func appendDependency(dependencies []string, dependency string) []string {
	for _, existing := range dependencies {
		if existing == dependency {
			return dependencies
		}
	}
	return append(dependencies, dependency)
}
