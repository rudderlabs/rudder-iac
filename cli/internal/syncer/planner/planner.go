package planner

import (
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

type Plan struct {
	Operations []*Operation
}

func New() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(source, target *resources.Graph) *Plan {
	diff := differ.ComputeDiff(source, target)

	plan := &Plan{}

	for _, urn := range diff.NewResources {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Create, Resource: resource})
	}

	for _, urn := range diff.UpdatedResources {
		resource, _ := target.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Update, Resource: resource})
	}

	for _, urn := range diff.RemovedResources {
		resource, _ := source.GetResource(urn)
		plan.Operations = append(plan.Operations, &Operation{Type: Delete, Resource: resource})
	}

	return plan
}
