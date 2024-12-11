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

	diff := differ.CompareData(data1, data2)

	assert.Len(t, diff.Diffs, 3)

	assert.Contains(t, diff.Diffs, "key2")
	assert.Contains(t, diff.Diffs, "key3")
	assert.Contains(t, diff.Diffs, "key4")

	assert.Equal(t, diff.Diffs["key2"].SourceValue, "value2")
	assert.Equal(t, diff.Diffs["key2"].TargetValue, "value3")

	assert.Equal(t, diff.Diffs["key3"].SourceValue, "value3")
	assert.Nil(t, diff.Diffs["key3"].TargetValue)

	assert.Nil(t, diff.Diffs["key4"].SourceValue)
	assert.Equal(t, diff.Diffs["key4"].TargetValue, "value4")
}

func TestComputeDiff(t *testing.T) {
	g1 := resources.NewGraph()
	g2 := resources.NewGraph()

	g1.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}))
	g1.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}))
	g1.AddResource(resources.NewResource("r2", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}))

	g2.AddResource(resources.NewResource("r0", "some-type", resources.ResourceData{"key1": "value1", "key2": "value2"}))
	g2.AddResource(resources.NewResource("r1", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}))
	g2.AddResource(resources.NewResource("r3", "some-type", resources.ResourceData{"key1": "value1", "key2": "value3"}))

	diff := differ.ComputeDiff(g1, g2)

	assert.Len(t, diff.NewResources, 1)
	assert.Len(t, diff.UpdatedResources, 1)
	assert.Len(t, diff.RemovedResources, 1)
	assert.Len(t, diff.UnmodifiedResources, 1)

	assert.Contains(t, diff.NewResources, "some-type:r3")
	assert.Contains(t, diff.UpdatedResources, "some-type:r1")
	assert.Contains(t, diff.RemovedResources, "some-type:r2")
	assert.Contains(t, diff.UnmodifiedResources, "some-type:r0")
}
