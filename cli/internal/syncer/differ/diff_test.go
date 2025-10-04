package differ_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
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
	assert.Len(t, diff.UpdatedResources, 1)
	assert.Len(t, diff.RemovedResources, 1)
	assert.Len(t, diff.UnmodifiedResources, 1)

	assert.Contains(t, diff.NewResources, "some-type:r3")
	assert.Contains(t, diff.ImportableResources, "some-type:r4")
	assert.Equal(t, diff.UpdatedResources["some-type:r1"], differ.ResourceDiff{URN: "some-type:r1", Diffs: map[string]differ.PropertyDiff{"key2": {Property: "key2", SourceValue: "value2", TargetValue: "value3"}}})
	assert.Contains(t, diff.RemovedResources, "some-type:r2")
	assert.Contains(t, diff.UnmodifiedResources, "some-type:r0")
}
