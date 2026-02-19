package testorchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// --- BuildPlan: ModeAll ---

func TestBuildPlan_ModeAll(t *testing.T) {
	t.Run("includes all transformations", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans 1", "code-v1"))
		local.AddResource(newTransResource("t2", "Trans 2", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeAll, "", "ws-1")

		require.NoError(t, err)
		assert.Len(t, plan.TestUnits, 2)
	})

	t.Run("empty graph produces empty plan", func(t *testing.T) {
		plan, err := NewPlanner(resources.NewGraph()).BuildPlan(context.Background(), resources.NewGraph(), ModeAll, "", "ws-1")

		require.NoError(t, err)
		assert.Empty(t, plan.TestUnits)
		assert.Empty(t, plan.StandaloneLibraries)
	})

	t.Run("library depended on by transformation is not standalone", func(t *testing.T) {
		local := resources.NewGraph()
		lib := newLibResource("lib-1", "Lib", "code-v1")
		trans := newTransResource("t1", "Trans", "code-v1")
		local.AddResource(lib)
		local.AddResource(trans)
		local.AddDependency(trans.URN(), lib.URN())

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeAll, "", "ws-1")

		require.NoError(t, err)
		assert.Len(t, plan.TestUnits, 1)
		assert.Empty(t, plan.StandaloneLibraries)
	})

	t.Run("library with no dependent transformations is standalone", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newLibResource("lib-1", "Lib", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeAll, "", "ws-1")

		require.NoError(t, err)
		assert.Empty(t, plan.TestUnits)
		require.Len(t, plan.StandaloneLibraries, 1)
		assert.Equal(t, "lib-1", plan.StandaloneLibraries[0].ID)
	})

	t.Run("test unit carries library dependencies", func(t *testing.T) {
		local := resources.NewGraph()
		lib1 := newLibResource("lib-1", "Lib1", "code-v1")
		lib2 := newLibResource("lib-2", "Lib2", "code-v1")
		trans := newTransResource("t1", "Trans", "code-v1")
		local.AddResource(lib1)
		local.AddResource(lib2)
		local.AddResource(trans)
		local.AddDependency(trans.URN(), lib1.URN())
		local.AddDependency(trans.URN(), lib2.URN())

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.Len(t, plan.TestUnits[0].Libraries, 2)
	})

	t.Run("new transformation is marked modified", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.True(t, plan.TestUnits[0].IsTransformationModified)
	})

	t.Run("unchanged transformation is not marked modified", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans", "same-code"))
		remote := resources.NewGraph()
		remote.AddResource(newTransResource("t1", "Trans", "same-code"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.False(t, plan.TestUnits[0].IsTransformationModified)
	})

	t.Run("transformation with code change is marked modified", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans", "code-v2"))
		remote := resources.NewGraph()
		remote.AddResource(newTransResource("t1", "Trans", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeAll, "", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.True(t, plan.TestUnits[0].IsTransformationModified)
	})
}

// --- BuildPlan: ModeModified ---

func TestBuildPlan_ModeModified(t *testing.T) {
	t.Run("new transformation is included", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans 1", "code-v1"))
		local.AddResource(newTransResource("t2", "Trans 2", "code-v1"))
		// remote only has t1 — t2 is new
		remote := resources.NewGraph()
		remote.AddResource(newTransResource("t1", "Trans 1", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeModified, "", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "t2", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("unchanged transformation is excluded", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans", "code-v1"))
		remote := resources.NewGraph()
		remote.AddResource(newTransResource("t1", "Trans", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeModified, "", "ws-1")

		require.NoError(t, err)
		assert.Empty(t, plan.TestUnits)
	})

	t.Run("transformation depending on modified library is included", func(t *testing.T) {
		local := resources.NewGraph()
		lib := newLibResource("lib-1", "Lib", "code-v2")
		trans := newTransResource("t1", "Trans", "code-v1")
		local.AddResource(lib)
		local.AddResource(trans)
		local.AddDependency(trans.URN(), lib.URN())

		remote := resources.NewGraph()
		remoteLib := newLibResource("lib-1", "Lib", "code-v1")
		remoteTrans := newTransResource("t1", "Trans", "code-v1")
		remote.AddResource(remoteLib)
		remote.AddResource(remoteTrans)
		remote.AddDependency(remoteTrans.URN(), remoteLib.URN())

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeModified, "", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "t1", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("modified standalone library appears in StandaloneLibraries", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newLibResource("lib-1", "Lib", "code-v2"))
		remote := resources.NewGraph()
		remote.AddResource(newLibResource("lib-1", "Lib", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeModified, "", "ws-1")

		require.NoError(t, err)
		assert.Empty(t, plan.TestUnits)
		require.Len(t, plan.StandaloneLibraries, 1)
		assert.Equal(t, "lib-1", plan.StandaloneLibraries[0].ID)
	})

	t.Run("no modifications produces empty plan", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans", "same-code"))
		remote := resources.NewGraph()
		remote.AddResource(newTransResource("t1", "Trans", "same-code"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), remote, ModeModified, "", "ws-1")

		require.NoError(t, err)
		assert.Empty(t, plan.TestUnits)
		assert.Empty(t, plan.StandaloneLibraries)
	})
}

// --- BuildPlan: ModeSingle ---

func TestBuildPlan_ModeSingle(t *testing.T) {
	t.Run("targets specific transformation by ID", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newTransResource("t1", "Trans 1", "code-v1"))
		local.AddResource(newTransResource("t2", "Trans 2", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeSingle, "t1", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "t1", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("targeting a library tests its dependent transformations", func(t *testing.T) {
		local := resources.NewGraph()
		lib := newLibResource("lib-1", "Lib", "code-v1")
		t1 := newTransResource("t1", "Trans 1", "code-v1")
		t2 := newTransResource("t2", "Trans 2", "code-v1")
		local.AddResource(lib)
		local.AddResource(t1)
		local.AddResource(t2)
		local.AddDependency(t1.URN(), lib.URN())
		// t2 does NOT depend on lib

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeSingle, "lib-1", "ws-1")

		require.NoError(t, err)
		require.Len(t, plan.TestUnits, 1)
		assert.Equal(t, "t1", plan.TestUnits[0].Transformation.ID)
	})

	t.Run("targeting a library with no dependents adds it as standalone", func(t *testing.T) {
		local := resources.NewGraph()
		local.AddResource(newLibResource("lib-1", "Lib", "code-v1"))

		plan, err := NewPlanner(local).BuildPlan(context.Background(), resources.NewGraph(), ModeSingle, "lib-1", "ws-1")

		require.NoError(t, err)
		assert.Empty(t, plan.TestUnits)
		require.Len(t, plan.StandaloneLibraries, 1)
		assert.Equal(t, "lib-1", plan.StandaloneLibraries[0].ID)
	})

	t.Run("unknown ID returns error", func(t *testing.T) {
		plan, err := NewPlanner(resources.NewGraph()).BuildPlan(context.Background(), resources.NewGraph(), ModeSingle, "no-such-id", "ws-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no-such-id")
		assert.Nil(t, plan)
	})
}

// --- TestPlan helpers ---

func TestTestPlan_IsLibraryModified(t *testing.T) {
	plan := &TestPlan{
		ModifiedLibraryURNs: map[string]bool{
			"transformation-library:lib-1": true,
		},
	}

	assert.True(t, plan.IsLibraryModified("transformation-library:lib-1"))
	assert.False(t, plan.IsLibraryModified("transformation-library:lib-2"))
}

func TestTestPlan_IsTransformationModified(t *testing.T) {
	plan := &TestPlan{
		ModifiedTransformationURNs: map[string]bool{
			"transformation:t1": true,
		},
	}

	assert.True(t, plan.IsTransformationModified("transformation:t1"))
	assert.False(t, plan.IsTransformationModified("transformation:t2"))
}

// --- resource helpers shared across test files ---

// newTransResource creates a transformation *resources.Resource with RawData set.
func newTransResource(id, name, code string) *resources.Resource {
	return resources.NewResource(id, "transformation", resources.ResourceData{}, nil,
		resources.WithRawData(&model.TransformationResource{ID: id, Name: name, Code: code}),
	)
}

// newLibResource creates a library *resources.Resource with RawData set.
func newLibResource(id, name, code string) *resources.Resource {
	return resources.NewResource(id, "transformation-library", resources.ResourceData{}, nil,
		resources.WithRawData(&model.LibraryResource{ID: id, Name: name, Code: code}),
	)
}
