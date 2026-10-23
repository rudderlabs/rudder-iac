package planner_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
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
	targetToBeImported := newResource("res5", "target resource 5", nil, resources.WithResourceImportMetadata("remote-res5", "workspace-id"))

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
		{
			name:   "import single resource",
			source: newGraph(),
			target: newGraphWithResources(targetToBeImported),
			expected: []*planner.Operation{
				{Type: planner.Import, Resource: targetToBeImported},
			},
		},
		{
			name: "combined import, create, update and delete",
			source: newGraphWithResources(
				sourceToBeDeleted,
				sourceUnmodified,
				sourceToBeUpdated,
			),
			target: newGraphWithResources(
				targetUnmodified,
				targetToBeUpdated,
				targetToBeCreated,
				targetToBeImported,
			),
			expected: []*planner.Operation{
				{Type: planner.Import, Resource: targetToBeImported},
				{Type: planner.Create, Resource: targetToBeCreated},
				{Type: planner.Update, Resource: targetToBeUpdated},
				{Type: planner.Delete, Resource: sourceToBeDeleted},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := planner.New("workspace-id")
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

func newResource(id string, name string, dependency *resources.PropertyRef, opts ...resources.ResourceOpts) *resources.Resource {
	data := resources.ResourceData{
		"name": name,
	}
	if dependency != nil {
		data["dependency"] = *dependency
	}
	return resources.NewResource(id, "some-type", data, []string{}, opts...)
}

func newResourceWithType(id string, name string, resourceType string) *resources.Resource {
	return resources.NewResource(id, resourceType, resources.ResourceData{"name": name}, []string{})
}

func TestPlanner_SkipDeletes(t *testing.T) {
	sourceToBeDeleted := newResource("res0", "source resource 0", nil)
	sourceToBeUpdated := newResource("res1", "source resource 1", nil)
	targetToBeUpdated := newResource("res1", "target resource 1", nil)
	targetToBeCreated := newResource("res2", "target resource 2", nil)

	tests := []struct {
		name        string
		skipDeletes bool
		source      *resources.Graph
		target      *resources.Graph
		expected    []*planner.Operation
	}{
		{
			name:        "with skip deletes enabled, delete operations are excluded",
			skipDeletes: true,
			source: newGraphWithResources(
				sourceToBeDeleted,
				sourceToBeUpdated,
			),
			target: newGraphWithResources(
				targetToBeUpdated,
				targetToBeCreated,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetToBeCreated},
				{Type: planner.Update, Resource: targetToBeUpdated},
			},
		},
		{
			name:        "with skip deletes disabled, delete operations are included",
			skipDeletes: false,
			source: newGraphWithResources(
				sourceToBeDeleted,
				sourceToBeUpdated,
			),
			target: newGraphWithResources(
				targetToBeUpdated,
				targetToBeCreated,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetToBeCreated},
				{Type: planner.Update, Resource: targetToBeUpdated},
				{Type: planner.Delete, Resource: sourceToBeDeleted},
			},
		},
		{
			name:        "skip deletes with only deletes results in empty plan",
			skipDeletes: true,
			source: newGraphWithResources(
				sourceToBeDeleted,
			),
			target:   newGraph(),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := planner.New("workspace-id", planner.WithSkipDeletes(tt.skipDeletes))
			plan := p.Plan(tt.source, tt.target)
			assert.Equal(t, tt.expected, plan.Operations)
		})
	}
}

func TestPlanner_ResourceTypeFilter(t *testing.T) {
	// Use resources with identical names in source and target to avoid update operations
	sourceTypeA := newResourceWithType("res0", "resource 0", "type-a")
	sourceTypeB := newResourceWithType("res1", "resource 1", "type-b")
	targetTypeA := newResourceWithType("res0", "resource 0", "type-a")   // Same as source - no update
	targetTypeB := newResourceWithType("res1", "resource 1", "type-b")   // Same as source - no update
	targetTypeANew := newResourceWithType("res2", "resource 2", "type-a")
	targetTypeBNew := newResourceWithType("res3", "resource 3", "type-b")

	tests := []struct {
		name          string
		resourceTypes []string
		source        *resources.Graph
		target        *resources.Graph
		expected      []*planner.Operation
	}{
		{
			name:          "filter to single type includes only that type",
			resourceTypes: []string{"type-a"},
			source: newGraphWithResources(
				sourceTypeA,
				sourceTypeB,
			),
			target: newGraphWithResources(
				targetTypeA,
				targetTypeANew,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetTypeANew},
			},
		},
		{
			name:          "filter to multiple types includes all specified types",
			resourceTypes: []string{"type-a", "type-b"},
			source: newGraphWithResources(
				sourceTypeA,
				sourceTypeB,
			),
			target: newGraphWithResources(
				targetTypeA,
				targetTypeB,
				targetTypeANew,
				targetTypeBNew,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetTypeANew},
				{Type: planner.Create, Resource: targetTypeBNew},
			},
		},
		{
			name:          "empty filter includes all types",
			resourceTypes: []string{},
			source: newGraphWithResources(
				sourceTypeA,
				sourceTypeB,
			),
			target: newGraphWithResources(
				targetTypeA,
				targetTypeB,
				targetTypeANew,
				targetTypeBNew,
			),
			expected: []*planner.Operation{
				{Type: planner.Create, Resource: targetTypeANew},
				{Type: planner.Create, Resource: targetTypeBNew},
			},
		},
		{
			name:          "filter excludes deletes for non-matching types",
			resourceTypes: []string{"type-a"},
			source: newGraphWithResources(
				sourceTypeA,
				sourceTypeB,
			),
			target: newGraph(),
			expected: []*planner.Operation{
				{Type: planner.Delete, Resource: sourceTypeA},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := planner.New("workspace-id", planner.WithResourceTypes(tt.resourceTypes))
			plan := p.Plan(tt.source, tt.target)
			assert.Equal(t, tt.expected, plan.Operations)
		})
	}
}

func TestPlanner_CombinedOptions(t *testing.T) {
	sourceTypeA := newResourceWithType("res0", "source type-a", "type-a")
	sourceTypeB := newResourceWithType("res1", "source type-b", "type-b")
	targetTypeANew := newResourceWithType("res2", "target type-a new", "type-a")

	t.Run("skip deletes with resource type filter", func(t *testing.T) {
		p := planner.New("workspace-id",
			planner.WithSkipDeletes(true),
			planner.WithResourceTypes([]string{"type-a"}),
		)

		source := newGraphWithResources(sourceTypeA, sourceTypeB)
		target := newGraphWithResources(targetTypeANew)

		plan := p.Plan(source, target)

		// Should only create type-a, no deletes even though sourceTypeA is removed
		expected := []*planner.Operation{
			{Type: planner.Create, Resource: targetTypeANew},
		}
		assert.Equal(t, expected, plan.Operations)
	})
}
