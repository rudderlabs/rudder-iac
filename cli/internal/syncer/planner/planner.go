package planner

import (
	"fmt"
	"slices"
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

type Planner struct {
	workspaceId string
	options     PlanOptions
}

// PlanOptions configures the planning behavior
type PlanOptions struct {
	// SkipDeletes excludes delete operations from the plan
	SkipDeletes bool
	// ResourceTypes filters operations to only include these resource types.
	// If empty, all resource types are included.
	ResourceTypes []string
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

type PlanOption func(*PlanOptions)

// WithSkipDeletes configures the planner to exclude delete operations
func WithSkipDeletes(skip bool) PlanOption {
	return func(o *PlanOptions) {
		o.SkipDeletes = skip
	}
}

// WithResourceTypes configures the planner to only include operations
// for the specified resource types
func WithResourceTypes(types []string) PlanOption {
	return func(o *PlanOptions) {
		o.ResourceTypes = types
	}
}

func New(workspaceId string, opts ...PlanOption) *Planner {
	options := PlanOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	return &Planner{
		workspaceId: workspaceId,
		options:     options,
	}
}

func (p *Planner) Plan(source, target *resources.Graph) *Plan {
	diff := differ.ComputeDiff(source, target, differ.DiffOptions{WorkspaceID: p.workspaceId})
	plan := &Plan{
		Diff: diff,
	}

	// Handle importable resources (will be imported from remote)
	sortedImportable := sortByDependencies(diff.ImportableResources, target)
	for _, urn := range sortedImportable {
		resource, _ := target.GetResource(urn)
		if p.shouldIncludeResource(resource) {
			plan.Operations = append(plan.Operations, &Operation{Type: Import, Resource: resource})
		}
	}

	// Handle new resources (will be created)
	sortedNew := sortByDependencies(diff.NewResources, target)
	for _, urn := range sortedNew {
		resource, _ := target.GetResource(urn)
		if p.shouldIncludeResource(resource) {
			plan.Operations = append(plan.Operations, &Operation{Type: Create, Resource: resource})
		}
	}

	// Handle updated resources
	updatedURNs := make([]string, 0, len(diff.UpdatedResources))
	for r := range diff.UpdatedResources {
		updatedURNs = append(updatedURNs, r)
	}
	sortedUpdated := sortByDependencies(updatedURNs, target)
	for _, urn := range sortedUpdated {
		resource, _ := target.GetResource(urn)
		if p.shouldIncludeResource(resource) {
			plan.Operations = append(plan.Operations, &Operation{Type: Update, Resource: resource})
		}
	}

	// Handle deleted resources (skip if SkipDeletes is enabled)
	if !p.options.SkipDeletes {
		sortedDeleted := sortByDependencies(diff.RemovedResources, source)
		slices.Reverse(sortedDeleted)
		for _, urn := range sortedDeleted {
			resource, _ := source.GetResource(urn)
			if p.shouldIncludeResource(resource) {
				plan.Operations = append(plan.Operations, &Operation{Type: Delete, Resource: resource})
			}
		}
	}

	return plan
}

// shouldIncludeResource checks if a resource should be included based on the
// configured resource type filter
func (p *Planner) shouldIncludeResource(resource *resources.Resource) bool {
	if len(p.options.ResourceTypes) == 0 {
		return true
	}

	for _, t := range p.options.ResourceTypes {
		if resource.Type() == t {
			return true
		}
	}
	return false
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
