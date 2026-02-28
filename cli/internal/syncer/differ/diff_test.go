package differ_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/stretchr/testify/assert"
)

func TestCompareData(t *testing.T) {
	data1 := resources.ResourceData{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}

	data2 := resources.ResourceData{
		"key1": "value1",
		"key2": "value3",
		"key4": "value4",
	}

	diffs := differ.CompareData(data1, data2)

	assert.Len(t, diffs, 3)

	assert.Contains(t, diffs, "key2")
	assert.Contains(t, diffs, "key3")
	assert.Contains(t, diffs, "key4")

	assert.Equal(t, diffs["key2"].SourceValue, "value2")
	assert.Equal(t, diffs["key2"].TargetValue, "value3")

	assert.Equal(t, diffs["key3"].SourceValue, "value3")
	assert.Nil(t, diffs["key3"].TargetValue)

	assert.Nil(t, diffs["key4"].SourceValue)
	assert.Equal(t, diffs["key4"].TargetValue, "value4")
}

func TestComputeDiff(t *testing.T) {
	g1 := resources.NewGraph()
	g2 := resources.NewGraph()

	g1.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g1.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g1.AddResource(resources.NewResource("r2", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))

	g2.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}, []string{}))
	g2.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}, []string{}))
	g2.AddResource(resources.NewResource("r3", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}, []string{}))
	g2.AddResource(resources.NewResource("r4", "some-type", resources.ResourceData{"key1": "value1", "key2": "value4"}, []string{}, resources.WithResourceImportMetadata("remote-id-r4", "workspace-id")))

	diff := differ.ComputeDiff(g1, g2, differ.DiffOptions{WorkspaceID: "workspace-id"})

	assert.Len(t, diff.NewResources, 1)
	assert.Len(t, diff.ImportableResources, 1)
	assert.Len(t, diff.NameMatchedResources, 0)
	assert.Len(t, diff.UpdatedResources, 1)
	assert.Len(t, diff.RemovedResources, 1)
	assert.Len(t, diff.UnmodifiedResources, 1)

	assert.Contains(t, diff.NewResources, "some-type:r3")
	assert.Contains(t, diff.ImportableResources, "some-type:r4")
	assert.Equal(t, diff.UpdatedResources["some-type:r1"], differ.ResourceDiff{URN: "some-type:r1", Diffs: map[string]differ.PropertyDiff{"key2": {Property: "key2", SourceValue: "value2", TargetValue: "value3"}}})
	assert.Contains(t, diff.RemovedResources, "some-type:r2")
	assert.Contains(t, diff.UnmodifiedResources, "some-type:r0")
}

func TestComputeDiff_MatchByName(t *testing.T) {
	t.Run("matches local resource to unmanaged remote by name", func(t *testing.T) {
		source := resources.NewGraph()
		target := resources.NewGraph()

		// Local resource with name "Canvas" that doesn't exist in managed remote state
		target.AddResource(resources.NewResource("canvas", "category", resources.ResourceData{
			"name":        "Canvas",
			"description": "Canvas events",
		}, []string{}))

		// Another local resource with no matching unmanaged remote
		target.AddResource(resources.NewResource("other", "category", resources.ResourceData{
			"name":        "Other",
			"description": "Other events",
		}, []string{}))

		// Unmanaged remote resources index
		unmanagedByName := map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas": {RemoteID: "cat_abc123", Name: "Canvas"},
			},
		}

		diff := differ.ComputeDiff(source, target, differ.DiffOptions{
			WorkspaceID:     "workspace-id",
			MatchByName:     true,
			UnmanagedByName: unmanagedByName,
		})

		assert.Len(t, diff.NameMatchedResources, 1)
		assert.Len(t, diff.NewResources, 1)

		assert.Equal(t, differ.NameMatchCandidate{
			LocalURN:     "category:canvas",
			RemoteID:     "cat_abc123",
			RemoteName:   "Canvas",
			ResourceType: "category",
		}, diff.NameMatchedResources[0])

		assert.Contains(t, diff.NewResources, "category:other")
	})

	t.Run("does not match when MatchByName is false", func(t *testing.T) {
		source := resources.NewGraph()
		target := resources.NewGraph()

		target.AddResource(resources.NewResource("canvas", "category", resources.ResourceData{
			"name": "Canvas",
		}, []string{}))

		unmanagedByName := map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas": {RemoteID: "cat_abc123", Name: "Canvas"},
			},
		}

		diff := differ.ComputeDiff(source, target, differ.DiffOptions{
			WorkspaceID:     "workspace-id",
			MatchByName:     false, // Disabled
			UnmanagedByName: unmanagedByName,
		})

		assert.Len(t, diff.NameMatchedResources, 0)
		assert.Len(t, diff.NewResources, 1)
		assert.Contains(t, diff.NewResources, "category:canvas")
	})

	t.Run("ImportMetadata takes precedence over name matching", func(t *testing.T) {
		source := resources.NewGraph()
		target := resources.NewGraph()

		// Resource with explicit ImportMetadata should use that, not name matching
		target.AddResource(resources.NewResource("canvas", "category", resources.ResourceData{
			"name": "Canvas",
		}, []string{}, resources.WithResourceImportMetadata("explicit-remote-id", "workspace-id")))

		unmanagedByName := map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas": {RemoteID: "cat_abc123", Name: "Canvas"},
			},
		}

		diff := differ.ComputeDiff(source, target, differ.DiffOptions{
			WorkspaceID:     "workspace-id",
			MatchByName:     true,
			UnmanagedByName: unmanagedByName,
		})

		assert.Len(t, diff.NameMatchedResources, 0)
		assert.Len(t, diff.ImportableResources, 1)
		assert.Contains(t, diff.ImportableResources, "category:canvas")
	})

	t.Run("does not match when resource has no name field", func(t *testing.T) {
		source := resources.NewGraph()
		target := resources.NewGraph()

		target.AddResource(resources.NewResource("canvas", "category", resources.ResourceData{
			"description": "No name field",
		}, []string{}))

		unmanagedByName := map[string]map[string]differ.UnmanagedResource{
			"category": {
				"Canvas": {RemoteID: "cat_abc123", Name: "Canvas"},
			},
		}

		diff := differ.ComputeDiff(source, target, differ.DiffOptions{
			WorkspaceID:     "workspace-id",
			MatchByName:     true,
			UnmanagedByName: unmanagedByName,
		})

		assert.Len(t, diff.NameMatchedResources, 0)
		assert.Len(t, diff.NewResources, 1)
	})

	t.Run("does not match when resource type not in unmanaged index", func(t *testing.T) {
		source := resources.NewGraph()
		target := resources.NewGraph()

		target.AddResource(resources.NewResource("canvas", "event", resources.ResourceData{
			"name": "Canvas",
		}, []string{}))

		unmanagedByName := map[string]map[string]differ.UnmanagedResource{
			"category": { // Different type
				"Canvas": {RemoteID: "cat_abc123", Name: "Canvas"},
			},
		}

		diff := differ.ComputeDiff(source, target, differ.DiffOptions{
			WorkspaceID:     "workspace-id",
			MatchByName:     true,
			UnmanagedByName: unmanagedByName,
		})

		assert.Len(t, diff.NameMatchedResources, 0)
		assert.Len(t, diff.NewResources, 1)
	})
}

func TestHasDiff_IncludesNameMatchedResources(t *testing.T) {
	diff := &differ.Diff{
		NameMatchedResources: []differ.NameMatchCandidate{
			{LocalURN: "category:canvas", RemoteID: "cat_123"},
		},
	}

	assert.True(t, diff.HasDiff())
}
