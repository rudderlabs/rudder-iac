package planner

import (
	"fmt"
	"slices"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/resources"
)

type Planner struct {
}

type OperationType int

const (
	Create OperationType = iota
	Update
	Delete
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
	default:
		return "Unknown"
	}
}

type Plan struct {
	Diff       *differ.Diff
	Operations []*Operation
}

func New() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(source, target *resources.Graph) *Plan {
	diff := differ.ComputeDiff(source, target)
	plan := &Plan{
		Diff: diff,
	}

	sortedNew := sortByDependencies(diff.NewResources, target)
	for _, urn := range sortedNew {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Create, Resource: resource})
	}

	updatedURNs := make([]string, 0, len(diff.UpdatedResources))
	for r := range diff.UpdatedResources {
		updatedURNs = append(updatedURNs, r)
	}
	sortedUpdated := sortByDependencies(updatedURNs, target)
	for _, urn := range sortedUpdated {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Update, Resource: resource})
	}

	sortedDeleted := sortByDependencies(diff.RemovedResources, source)
	slices.Reverse(sortedDeleted)
	for _, urn := range sortedDeleted {
		resource, _ := source.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Delete, Resource: resource})
	}

	return plan
}

// sortByDependencies returns resources ordered by their dependencies,
// so that resources that depend on others are visited after their dependencies.
// Resources with the same dependencies are sorted alphabetically for consistent ordering.
func sortByDependencies(urns []string, g *resources.Graph) []string {
	// Sort URNs alphabetically first
	sort.Strings(urns)

	visited := make(map[string]bool)
	sorted := make([]string, 0, len(urns))

	var visit func(string)
	visit = func(urn string) {
		if visited[urn] {
			return
		}
		visited[urn] = true

		for _, dep := range g.GetDependencies(urn) {
			// Only visit dependencies that are in our target URN set
			for _, targetURN := range urns {
				if dep == targetURN {
					visit(dep)
					break
				}
			}
		}
		sorted = append(sorted, urn)
	}

	for _, urn := range urns {
		visit(urn)
	}

	return sorted
}
