package planner

import (
	"fmt"
	"maps"
	"slices"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

type Planner struct {
	workspaceId string
}

type OperationType int

const (
	Create OperationType = iota
	Update
	Delete
	Import
)

type Operation struct {
	Type     OperationType
	Resource *resources.Resource
}

func (o *Operation) String() string {
	return fmt.Sprintf("%s %s", o.Type.String(), o.Resource.URN())
}

func (t *OperationType) String() string {
	switch *t {
	case Create:
		return "Create"
	case Update:
		return "Update"
	case Delete:
		return "Delete"
	case Import:
		return "Import"
	default:
		return "Unknown"
	}
}

type Plan struct {
	Diff       *differ.Diff
	Operations []*Operation
}

func New(workspaceId string) *Planner {
	return &Planner{
		workspaceId: workspaceId,
	}
}

func (p *Planner) Plan(source, target *resources.Graph) *Plan {
	diff := differ.ComputeDiff(source, target, differ.DiffOptions{WorkspaceID: p.workspaceId})
	plan := &Plan{
		Diff: diff,
	}

	// Non-delete operations are ordered across types: an import or create can
	// reference a resource created or updated in the same apply, and the
	// sequential executor runs operations in plan order. Grouping by type
	// remains the tie-break where dependencies allow.
	opTypes := make(map[string]OperationType)
	var ordered []string
	for _, group := range []struct {
		opType OperationType
		urns   []string
	}{
		{Import, diff.ImportableResources},
		{Create, diff.NewResources},
		{Update, slices.Collect(maps.Keys(diff.UpdatedResources))},
	} {
		for _, urn := range slices.Sorted(slices.Values(group.urns)) {
			opTypes[urn] = group.opType
			ordered = append(ordered, urn)
		}
	}
	for _, urn := range sortByDependencies(ordered, target) {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: opTypes[urn], Resource: resource})
	}

	// Handle deleted resources
	sortedDeleted := sortByDependencies(slices.Sorted(slices.Values(diff.RemovedResources)), source)
	slices.Reverse(sortedDeleted)
	for _, urn := range sortedDeleted {
		resource, _ := source.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Delete, Resource: resource})
	}

	return plan
}

// sortByDependencies returns urns ordered so that each resource comes after
// its dependencies within the set, otherwise keeping the input order.
func sortByDependencies(urns []string, g *resources.Graph) []string {
	inSet := make(map[string]bool, len(urns))
	for _, urn := range urns {
		inSet[urn] = true
	}

	visited := make(map[string]bool)
	sorted := make([]string, 0, len(urns))

	var visit func(string)
	visit = func(urn string) {
		if visited[urn] {
			return
		}
		visited[urn] = true

		for _, dep := range g.GetDependencies(urn) {
			if inSet[dep] {
				visit(dep)
			}
		}
		sorted = append(sorted, urn)
	}

	for _, urn := range urns {
		visit(urn)
	}

	return sorted
}
