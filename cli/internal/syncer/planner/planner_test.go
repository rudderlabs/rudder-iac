package planner_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

func TestPlanner_Plan(t *testing.T) {
	sourceToBeDeleted := newResource("res0", "source resource 0", nil)
	sourceUnmodified := newResource("res1", "source resource 1", nil)
	sourceWithDependency := newResource("res2", "source resource 2", &resources.PropertyRef{URN: sourceUnmodified.URN(), Property: "name"})
	sourceToBeUpdated := newResource("res3", "source resource 3", nil)
	targetUnmodified := newResource("res1", "source resource 1", nil)
	targetWithDependency := newResource("res2", "target resource 2", &resources.PropertyRef{URN: sourceUnmodified.URN(), Property: "name"})
	targetToBeUpdated := newResource("res3", "target resource 3", nil)
	targetToBeCreated := newResource("res4", "target resource 4", nil)

	tests := []struct {
		name     string
		source   *resources.Graph
		target   *resources.Graph
		expected []*planner.Operation
	}{
		{
			name:   "create single resource",
			source: newGraph(),
			target: newGraphWithResources(targetToBeCreated),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetToBeCreated},
			},
		},
		{
			name:   "create resources with dependencies",
			source: newGraph(),
			target: newGraphWithResources(
				targetUnmodified,
				targetWithDependency,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetUnmodified},
				{Type: planner.Create, Resource: targetWithDependency},
			},
		},
		{
			name: "delete single resource",
			source: newGraphWithResources(
				sourceToBeDeleted,
			),
			target: newGraph(),
			expected: []*planner.Operation{
				{Type: planner.Delete, Resource: sourceToBeDeleted},
			},
		},
		{
			name: "delete resources with dependencies in reverse order",
			source: newGraphWithResources(
				sourceUnmodified,
				sourceWithDependency,
			),
			target: newGraph(),
			expected: []*planner.Operation{
				{Type: planner.Delete, Resource: sourceWithDependency},
				{Type: planner.Delete, Resource: sourceUnmodified},
			},
		},
		{
			name: "update single resource",
			source: newGraphWithResources(
				sourceToBeUpdated,
			),
			target: newGraphWithResources(
				targetToBeUpdated,
			),
			expected: []*planner.Operation{
				{Type: planner.Update, Resource: targetToBeUpdated},
			},
		},
		{
			name: "combined create, update and delete",
			source: newGraphWithResources(
				sourceToBeDeleted,
				sourceUnmodified,
				sourceToBeUpdated,
			),
			target: newGraphWithResources(
				targetUnmodified,
				targetToBeUpdated,
				targetToBeCreated,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetToBeCreated},
				{Type: planner.Update, Resource: targetToBeUpdated},
				{Type: planner.Delete, Resource: sourceToBeDeleted},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := planner.New()
			plan := p.Plan(tt.source, tt.target)
			for i, op := range plan.Operations {
				fmt.Printf("Operation %d: %d %s\n", i, op.Type, op.Resource.ID())
			}
			assert.Equal(t, tt.expected, plan.Operations)
		})
	}
}

func newGraph() *resources.Graph {
	return resources.NewGraph()
}

func newGraphWithResources(rs ...*resources.Resource) *resources.Graph {
	g := resources.NewGraph()
	for _, r := range rs {
		g.AddResource(r)
	}
	return g
}

func newResource(id string, name string, dependency *resources.PropertyRef) *resources.Resource {
	data := resources.ResourceData{
		"name": name,
	}
	if dependency != nil {
		data["dependency"] = *dependency
	}
	return resources.NewResource(id, "some-type", data)
}
