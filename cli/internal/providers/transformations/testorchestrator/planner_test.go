package testorchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

func TestBuildPlan(t *testing.T) {
	t.Run("ModeAll - includes all transformations", func(t *testing.T) {
		// Create local graph with two transformations
		localGraph := resources.NewGraph()
		trans1 := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		trans2 := createTestTransformationResource("trans-2", "Transformation 2", "javascript")
		localGraph.AddResource(trans1)
		localGraph.AddResource(trans2)

		// Create empty remote graph
		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Len(t, plan.TestUnits, 2)
	})

	t.Run("ModeAll with no transformations", func(t *testing.T) {
		localGraph := resources.NewGraph()
		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Empty(t, plan.TestUnits)
	})

	t.Run("ModeModified - includes only modified transformations", func(t *testing.T) {
		// Create local graph with two transformations
		localGraph := resources.NewGraph()
		trans1 := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		trans2 := createTestTransformationResource("trans-2", "Transformation 2", "javascript")
		localGraph.AddResource(trans1)
		localGraph.AddResource(trans2)

		// Create remote graph with only trans-1 (trans-2 is new)
		remoteGraph := resources.NewGraph()
		remoteTrans1 := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		remoteGraph.AddResource(remoteTrans1)

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeModified, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "trans-2", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("ModeModified - includes transformations with modified libraries", func(t *testing.T) {
		// Create local graph with library and transformation
		localGraph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Library 1", "javascript")
		trans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		localGraph.AddResource(lib)
		localGraph.AddResource(trans)
		localGraph.AddDependency(trans.URN(), lib.URN())

		// Create remote graph with old version of library
		remoteGraph := resources.NewGraph()
		remoteLib := createTestLibraryResource("lib-1", "Library 1 Old", "javascript")
		remoteTrans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		remoteGraph.AddResource(remoteLib)
		remoteGraph.AddResource(remoteTrans)
		remoteGraph.AddDependency(remoteTrans.URN(), remoteLib.URN())

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeModified, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "trans-1", plan.TestUnits[0].Transformation.ID)
		assert.True(t, plan.TestUnits[0].IsLibraryModified("transformation-library:lib-1"))
	})

	t.Run("ModeSingle - transformation by ID", func(t *testing.T) {
		// Create local graph with two transformations
		localGraph := resources.NewGraph()
		trans1 := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		trans2 := createTestTransformationResource("trans-2", "Transformation 2", "javascript")
		localGraph.AddResource(trans1)
		localGraph.AddResource(trans2)

		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeSingle, "trans-1", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "trans-1", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("ModeSingle - library by ID tests dependent transformations", func(t *testing.T) {
		// Create local graph with library and two transformations
		localGraph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Library 1", "javascript")
		trans1 := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		trans2 := createTestTransformationResource("trans-2", "Transformation 2", "javascript")
		localGraph.AddResource(lib)
		localGraph.AddResource(trans1)
		localGraph.AddResource(trans2)
		localGraph.AddDependency(trans1.URN(), lib.URN())

		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeSingle, "lib-1", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "trans-1", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("ModeSingle - library with no dependents returns error", func(t *testing.T) {
		// Create local graph with library but no transformations
		localGraph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Library 1", "javascript")
		localGraph.AddResource(lib)

		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeSingle, "lib-1", "ws-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "has no dependent transformations")
		assert.Nil(t, plan)
	})

	t.Run("ModeSingle - nonexistent ID returns error", func(t *testing.T) {
		localGraph := resources.NewGraph()
		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeSingle, "nonexistent", "ws-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.Nil(t, plan)
	})

	t.Run("test unit includes library dependencies", func(t *testing.T) {
		// Create local graph with libraries and transformation
		localGraph := resources.NewGraph()
		lib1 := createTestLibraryResource("lib-1", "Library 1", "javascript")
		lib2 := createTestLibraryResource("lib-2", "Library 2", "javascript")
		trans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		localGraph.AddResource(lib1)
		localGraph.AddResource(lib2)
		localGraph.AddResource(trans)
		localGraph.AddDependency(trans.URN(), lib1.URN())
		localGraph.AddDependency(trans.URN(), lib2.URN())

		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.Len(t, plan.TestUnits, 1)
		assert.Len(t, plan.TestUnits[0].Libraries, 2)
	})

	t.Run("test unit marks transformation as modified when new", func(t *testing.T) {
		// Create local graph with transformation
		localGraph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		localGraph.AddResource(trans)

		// Empty remote graph
		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.Len(t, plan.TestUnits, 1)
		assert.True(t, plan.TestUnits[0].IsTransformationModified)
	})

	t.Run("test unit marks transformation as unmodified when unchanged", func(t *testing.T) {
		// Create local graph
		localGraph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		localGraph.AddResource(trans)

		// Create identical remote graph
		remoteGraph := resources.NewGraph()
		remoteTrans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		remoteGraph.AddResource(remoteTrans)

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.Len(t, plan.TestUnits, 1)
		assert.False(t, plan.TestUnits[0].IsTransformationModified)
	})

	t.Run("test unit marks library as modified when updated", func(t *testing.T) {
		// Create local graph
		localGraph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Library 1 Updated", "javascript")
		trans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		localGraph.AddResource(lib)
		localGraph.AddResource(trans)
		localGraph.AddDependency(trans.URN(), lib.URN())

		// Create remote graph with old library
		remoteGraph := resources.NewGraph()
		remoteLib := createTestLibraryResource("lib-1", "Library 1 Old", "javascript")
		remoteTrans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		remoteGraph.AddResource(remoteLib)
		remoteGraph.AddResource(remoteTrans)
		remoteGraph.AddDependency(remoteTrans.URN(), remoteLib.URN())

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.Len(t, plan.TestUnits, 1)
		assert.True(t, plan.TestUnits[0].IsLibraryModified("transformation-library:lib-1"))
	})

	t.Run("transformation without library dependencies has empty libraries list", func(t *testing.T) {
		localGraph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Transformation 1", "javascript")
		localGraph.AddResource(trans)

		remoteGraph := resources.NewGraph()

		planner := NewPlanner(localGraph)
		plan, err := planner.BuildPlan(context.Background(), remoteGraph, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.NotNil(t, plan)
		require.Len(t, plan.TestUnits, 1)
		assert.Empty(t, plan.TestUnits[0].Libraries)
	})
}

func TestFindResourceByID(t *testing.T) {
	t.Run("finds transformation by ID", func(t *testing.T) {
		graph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Test", "javascript")
		graph.AddResource(trans)

		planner := NewPlanner(graph)
		resource := planner.findResourceByID("trans-1")

		require.NotNil(t, resource)
		assert.Equal(t, "trans-1", resource.ID())
	})

	t.Run("finds library by ID", func(t *testing.T) {
		graph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Test", "javascript")
		graph.AddResource(lib)

		planner := NewPlanner(graph)
		resource := planner.findResourceByID("lib-1")

		require.NotNil(t, resource)
		assert.Equal(t, "lib-1", resource.ID())
	})

	t.Run("returns nil for nonexistent ID", func(t *testing.T) {
		graph := resources.NewGraph()

		planner := NewPlanner(graph)
		resource := planner.findResourceByID("nonexistent")

		assert.Nil(t, resource)
	})
}

func TestBuildModifiedResourceSet(t *testing.T) {
	t.Run("includes new resources", func(t *testing.T) {
		graph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Test", "javascript")
		graph.AddResource(trans)

		diff := &differ.Diff{
			NewResources: []string{"transformation:trans-1"},
		}

		planner := NewPlanner(graph)
		resourceList := planner.filterByType("transformation")
		modifiedSet := planner.buildModifiedResourceSet(resourceList, diff)

		assert.Len(t, modifiedSet, 1)
		assert.True(t, modifiedSet["transformation:trans-1"])
	})

	t.Run("includes updated resources", func(t *testing.T) {
		graph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Test", "javascript")
		graph.AddResource(trans)

		diff := &differ.Diff{
			UpdatedResources: map[string]differ.ResourceDiff{
				"transformation:trans-1": {},
			},
		}

		planner := NewPlanner(graph)
		resourceList := planner.filterByType("transformation")
		modifiedSet := planner.buildModifiedResourceSet(resourceList, diff)

		assert.Len(t, modifiedSet, 1)
		assert.True(t, modifiedSet["transformation:trans-1"])
	})

	t.Run("excludes unchanged resources", func(t *testing.T) {
		graph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Test", "javascript")
		graph.AddResource(trans)

		diff := &differ.Diff{}

		planner := NewPlanner(graph)
		resourceList := planner.filterByType("transformation")
		modifiedSet := planner.buildModifiedResourceSet(resourceList, diff)

		assert.Empty(t, modifiedSet)
	})

	t.Run("only includes resources of matching type", func(t *testing.T) {
		graph := resources.NewGraph()
		trans := createTestTransformationResource("trans-1", "Test", "javascript")
		lib := createTestLibraryResource("lib-1", "Test", "javascript")
		graph.AddResource(trans)
		graph.AddResource(lib)

		diff := &differ.Diff{
			NewResources: []string{"transformation:trans-1", "transformation-library:lib-1"},
		}

		planner := NewPlanner(graph)
		transformations := planner.filterByType("transformation")
		modifiedTransformations := planner.buildModifiedResourceSet(transformations, diff)

		assert.Len(t, modifiedTransformations, 1)
		assert.True(t, modifiedTransformations["transformation:trans-1"])
		assert.False(t, modifiedTransformations["transformation-library:lib-1"])
	})
}

func TestFindDependentTransformations(t *testing.T) {
	t.Run("finds transformations that depend on library", func(t *testing.T) {
		graph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Test", "javascript")
		trans1 := createTestTransformationResource("trans-1", "Test 1", "javascript")
		trans2 := createTestTransformationResource("trans-2", "Test 2", "javascript")
		graph.AddResource(lib)
		graph.AddResource(trans1)
		graph.AddResource(trans2)
		graph.AddDependency(trans1.URN(), lib.URN())

		planner := NewPlanner(graph)
		transformations := planner.filterByType("transformation")
		dependents := planner.findDependentTransformations(lib, transformations)

		assert.Len(t, dependents, 1)
		assert.Equal(t, "trans-1", dependents[0].ID())
	})

	t.Run("returns empty when no transformations depend on library", func(t *testing.T) {
		graph := resources.NewGraph()
		lib := createTestLibraryResource("lib-1", "Test", "javascript")
		trans := createTestTransformationResource("trans-1", "Test", "javascript")
		graph.AddResource(lib)
		graph.AddResource(trans)

		planner := NewPlanner(graph)
		transformations := planner.filterByType("transformation")
		dependents := planner.findDependentTransformations(lib, transformations)

		assert.Empty(t, dependents)
	})
}

// Helper functions to create test resources
// These mirror how BaseHandler.Resources() creates resources using WithRawData

func createTestTransformationResource(id, name, language string) *resources.Resource {
	transformation := &model.TransformationResource{
		ID:       id,
		Name:     name,
		Language: language,
		Code:     "export function transformEvent(event) { return event; }",
	}

	resource := resources.NewResource(id, "transformation", resources.ResourceData{}, []string{},
		resources.WithRawData(transformation),
	)
	return resource
}

func createTestLibraryResource(id, name, language string) *resources.Resource {
	library := &model.LibraryResource{
		ID:       id,
		Name:     name,
		Language: language,
		Code:     "export function helper() {}",
	}

	resource := resources.NewResource(id, "transformation-library", resources.ResourceData{}, []string{},
		resources.WithRawData(library),
	)
	return resource
}
