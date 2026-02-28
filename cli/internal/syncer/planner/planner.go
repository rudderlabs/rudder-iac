package planner

import (
	"fmt"
	"slices"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

type Planner struct {
	workspaceID string
	diffOptions differ.DiffOptions
}

type PlannerOption func(*Planner)

// WithDiffOptions sets additional diff options for name matching and other features
func WithDiffOptions(opts differ.DiffOptions) PlannerOption {
	return func(p *Planner) {
		// Preserve workspaceID, merge other options
		opts.WorkspaceID = p.workspaceID
		p.diffOptions = opts
	}
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

func New(workspaceID string, opts ...PlannerOption) *Planner {
	p := &Planner{
		workspaceID: workspaceID,
		diffOptions: differ.DiffOptions{WorkspaceID: workspaceID},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Planner) Plan(source, target *resources.Graph) *Plan {
	diff := differ.ComputeDiff(source, target, p.diffOptions)
	plan := &Plan{
		Diff: diff,
	}

	// Handle importable resources (will be imported from remote)
	sortedImportable := sortByDependencies(diff.ImportableResources, target)
	for _, urn := range sortedImportable {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Import, Resource: resource})
	}

	// Handle new resources (will be created)
	sortedNew := sortByDependencies(diff.NewResources, target)
	for _, urn := range sortedNew {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Create, Resource: resource})
	}

	// Handle updated resources
	updatedURNs := make([]string, 0, len(diff.UpdatedResources))
	for r := range diff.UpdatedResources {
		updatedURNs = append(updatedURNs, r)
	}
	sortedUpdated := sortByDependencies(updatedURNs, target)
	for _, urn := range sortedUpdated {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Update, Resource: resource})
	}

	// Handle deleted resources
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
