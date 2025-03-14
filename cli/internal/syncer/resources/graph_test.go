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
