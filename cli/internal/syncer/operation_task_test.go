package syncer

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOperationTasksSerializesTransformationOperations(t *testing.T) {
	sourceGraph := resources.NewGraph()
	targetGraph := resources.NewGraph()

	property := resources.NewResource("email", "property", nil, nil)
	library := resources.NewResource("python_utils_library", transformationLibraryResourceType, nil, nil)
	transformationWithLibrary := resources.NewResource("python_with_imports", transformationResourceType, nil, nil)
	transformationWithoutLibrary := resources.NewResource("simple_python_transform", transformationResourceType, nil, nil)

	targetGraph.AddResource(property)
	targetGraph.AddResource(library)
	targetGraph.AddResource(transformationWithLibrary)
	targetGraph.AddDependency(transformationWithLibrary.URN(), library.URN())
	targetGraph.AddResource(transformationWithoutLibrary)

	tasks := newOperationTasks(&planner.Plan{Operations: []*planner.Operation{
		{Type: planner.Create, Resource: property},
		{Type: planner.Create, Resource: library},
		{Type: planner.Create, Resource: transformationWithLibrary},
		{Type: planner.Create, Resource: transformationWithoutLibrary},
	}}, sourceGraph, targetGraph)

	require.Len(t, tasks, 4)
	assert.Empty(t, tasks[0].Dependencies())
	assert.Empty(t, tasks[1].Dependencies())
	assert.Equal(t, []string{library.URN()}, tasks[2].Dependencies())
	assert.Equal(t, []string{transformationWithLibrary.URN()}, tasks[3].Dependencies())
}

func TestNewOperationTasksSerializesTransformationDeletesWithoutDuplicateDependencies(t *testing.T) {
	sourceGraph := resources.NewGraph()
	targetGraph := resources.NewGraph()

	library := resources.NewResource("python_utils_library", transformationLibraryResourceType, nil, nil)
	transformation := resources.NewResource("python_with_imports", transformationResourceType, nil, nil)

	sourceGraph.AddResource(library)
	sourceGraph.AddResource(transformation)
	sourceGraph.AddDependency(transformation.URN(), library.URN())

	tasks := newOperationTasks(&planner.Plan{Operations: []*planner.Operation{
		{Type: planner.Delete, Resource: transformation},
		{Type: planner.Delete, Resource: library},
	}}, sourceGraph, targetGraph)

	require.Len(t, tasks, 2)
	assert.Empty(t, tasks[0].Dependencies())
	assert.Equal(t, []string{transformation.URN()}, tasks[1].Dependencies())
}
