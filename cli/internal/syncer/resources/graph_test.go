package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDependencies(t *testing.T) {
	g := NewGraph()

	r1 := NewResource("test:resource1", "test", map[string]interface{}{}, []string{})
	r2 := NewResource("test:resource2", "test", map[string]interface{}{}, []string{})
	r3 := NewResource("test:resource3", "test", map[string]interface{}{}, []string{})

	g.AddResource(r1)
	g.AddResource(r2)
	g.AddResource(r3)

	g.AddDependency(r2.URN(), r1.URN())
	g.AddDependency(r2.URN(), r1.URN()) // adding twice should not make a difference
	g.AddDependency(r3.URN(), r1.URN())
	g.AddDependencies(r1.URN(), []string{r2.URN(), r3.URN()})

	assert.ElementsMatch(t, []string{r2.URN(), r3.URN()}, g.GetDependencies(r1.URN()))
	assert.Equal(t, []string{r1.URN()}, g.GetDependencies(r2.URN()))
	assert.Equal(t, []string{r1.URN()}, g.GetDependencies(r3.URN()))

	// adding multiple dependencies at once, without duplicating existing ones
	g.AddDependencies(r3.URN(), []string{r2.URN(), r1.URN()})
	assert.ElementsMatch(t, []string{r2.URN(), r1.URN()}, g.GetDependencies(r3.URN()))
}

func TestMerge(t *testing.T) {
	// Graph 1
	g1 := NewGraph()
	r1_1 := NewResource("test:g1_resource1", "test", map[string]interface{}{}, []string{})
	r1_2 := NewResource("test:g1_resource2", "test", map[string]interface{}{}, []string{})

	g1.AddResource(r1_1)
	g1.AddResource(r1_2)
	g1.AddDependency(r1_2.URN(), r1_1.URN())

	g2 := NewGraph()
	r2_1 := NewResource("test:g2_resource1", "test", map[string]interface{}{}, []string{})
	r2_2 := NewResource("test:g2_resource2", "test", map[string]interface{}{}, []string{})

	g2.AddResource(r2_1)
	g2.AddResource(r2_2)
	g2.AddDependency(r1_1.URN(), r2_1.URN()) // New dependency for existing r1_1
	g2.AddDependency(r1_2.URN(), r1_1.URN()) // Duplicate dependency, should not be added twice

	// Merge g2 into g1
	g1.Merge(g2)

	// Assert Resources
	// All resources from g1 and g2 should be present
	_, r1_1_exists := g1.GetResource(r1_1.URN())
	_, r1_2_exists := g1.GetResource(r1_2.URN())
	_, r2_1_exists := g1.GetResource(r2_1.URN())
	_, r2_2_exists := g1.GetResource(r2_2.URN())

	assert.True(t, r1_1_exists, "r1_1 should exist in merged graph")
	assert.True(t, r1_2_exists, "r1_2 should exist in merged graph")
	assert.True(t, r2_1_exists, "r2_1 should exist in merged graph")
	assert.True(t, r2_2_exists, "r2_2 should exist in merged graph")

	expectedResourceURNs := []string{
		r1_1.URN(),
		r1_2.URN(),
		r2_1.URN(),
		r2_2.URN(),
	}
	actualResourceURNs := []string{}
	for urn := range g1.Resources() {
		actualResourceURNs = append(actualResourceURNs, urn)
	}
	assert.ElementsMatch(t, expectedResourceURNs, actualResourceURNs, "Merged graph should contain all unique resources")

	assert.ElementsMatch(t, []string{r1_1.URN()}, g1.GetDependencies(r1_2.URN()), "Dependencies for r1_2 are incorrect")
	assert.ElementsMatch(t, []string{r2_1.URN()}, g1.GetDependencies(r1_1.URN()), "Dependencies for r1_1 are incorrect")
	assert.Empty(t, g1.GetDependencies(r2_2.URN()), "Dependencies for r2_2 should be empty")

	// Test merging an empty graph
	g3 := NewGraph()
	r3_1 := NewResource("test:g3_resource1", "test", map[string]any{}, []string{})
	g3.AddResource(r3_1)
	g3.AddDependency(r3_1.URN(), "dep1")

	gEmpty := NewGraph()
	g3.Merge(gEmpty) // Merge empty graph into g3

	// g3 should remain unchanged
	_, r3_1_exists_after_empty_merge := g3.GetResource(r3_1.URN())
	assert.True(t, r3_1_exists_after_empty_merge)
	assert.Equal(t, 1, len(g3.Resources()))
	assert.ElementsMatch(t, []string{"dep1"}, g3.GetDependencies(r3_1.URN()))

	// Test merging into an empty graph
	gEmpty2 := NewGraph()
	g4 := NewGraph()
	r4_1 := NewResource("test:g4_resource1", "test", map[string]interface{}{}, []string{})
	r4_2 := NewResource("test:g4_resource2", "test", map[string]interface{}{}, []string{})
	g4.AddResource(r4_1)
	g4.AddResource(r4_2)
	g4.AddDependency(r4_1.URN(), r4_2.URN())

	gEmpty2.Merge(g4) // Merge g4 into empty graph gEmpty2

	// gEmpty2 should now be identical to g4
	_, r4_1_exists_in_empty_merge := gEmpty2.GetResource(r4_1.URN())
	_, r4_2_exists_in_empty_merge := gEmpty2.GetResource(r4_2.URN())
	assert.True(t, r4_1_exists_in_empty_merge)
	assert.True(t, r4_2_exists_in_empty_merge)
	assert.Equal(t, 2, len(gEmpty2.Resources()))
	assert.ElementsMatch(t, []string{r4_2.URN()}, gEmpty2.GetDependencies(r4_1.URN()))
	assert.Empty(t, gEmpty2.GetDependencies(r4_2.URN()))
}
