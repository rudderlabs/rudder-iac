package syncer

import (

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type operationTask struct {
	operation   *planner.Operation
	sourceGraph *resources.Graph
	targetGraph *resources.Graph
}

func newOperationTask(operation *planner.Operation, sourceGraph *resources.Graph, targetGraph *resources.Graph) *operationTask {
	return &operationTask{operation: operation, sourceGraph: sourceGraph, targetGraph: targetGraph}
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
	// For delete operations, we need to invert the dependency order
	// If A depends on B, then for deletion: B should be deleted before A
	if t.operation.Type == planner.Delete {
		// For delete operations, we need to find which resources currently depend on the resource being deleted.
		// This information exists in the source graph. Target graph may not even contain the dependents
		return t.sourceGraph.GetDependents(t.operation.Resource.URN())
	}
	return t.targetGraph.GetDependencies(t.operation.Resource.URN())
}

