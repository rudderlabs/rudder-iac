package validations

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// Mode represents the validation mode
type Mode int

const (
	ModeAll      Mode = iota
	ModeModified
	ModeSingle
)

// ValidationUnit represents a single resource to validate
type ValidationUnit struct {
	ResourceType string // "model" or "relationship"
	ID           string // local resource ID
	Resource     any    // *dgModel.ModelResource or *dgModel.RelationshipResource
	AccountID    string // resolved account ID from the parent data graph spec
}

// ValidationPlan holds the units to validate
type ValidationPlan struct {
	Units []*ValidationUnit
}

// Planner determines which resources to validate
type Planner struct {
	graph *resources.Graph
}

// NewPlanner creates a new validation planner
func NewPlanner(graph *resources.Graph) *Planner {
	return &Planner{graph: graph}
}

// BuildPlan determines which resources to validate based on the mode
func (p *Planner) BuildPlan(remoteGraph *resources.Graph, mode Mode, resourceType, targetID string, opts differ.DiffOptions) (*ValidationPlan, error) {
	switch mode {
	case ModeAll:
		return p.planAll()
	case ModeModified:
		return p.planModified(remoteGraph, opts)
	case ModeSingle:
		return p.planSingle(resourceType, targetID)
	default:
		return nil, fmt.Errorf("unknown validation mode: %d", mode)
	}
}

func (p *Planner) planAll() (*ValidationPlan, error) {
	var units []*ValidationUnit

	// Add all models
	for _, r := range p.graph.ResourcesByType(model.HandlerMetadata.ResourceType) {
		modelRes, ok := r.RawData().(*dgModel.ModelResource)
		if !ok {
			continue
		}
		units = append(units, &ValidationUnit{
			ResourceType: "model",
			ID:           r.ID(),
			Resource:     modelRes,
		})
	}

	// Add all relationships
	for _, r := range p.graph.ResourcesByType(relationship.HandlerMetadata.ResourceType) {
		relRes, ok := r.RawData().(*dgModel.RelationshipResource)
		if !ok {
			continue
		}
		units = append(units, &ValidationUnit{
			ResourceType: "relationship",
			ID:           r.ID(),
			Resource:     relRes,
		})
	}

	return &ValidationPlan{Units: units}, nil
}

func (p *Planner) planModified(remoteGraph *resources.Graph, opts differ.DiffOptions) (*ValidationPlan, error) {
	diff := differ.ComputeDiff(remoteGraph, p.graph, opts)

	modifiedURNs := make(map[string]bool)
	for _, urn := range diff.NewResources {
		modifiedURNs[urn] = true
	}
	for _, urn := range diff.ImportableResources {
		modifiedURNs[urn] = true
	}
	for urn := range diff.UpdatedResources {
		modifiedURNs[urn] = true
	}

	var units []*ValidationUnit

	for _, r := range p.graph.ResourcesByType(model.HandlerMetadata.ResourceType) {
		if !modifiedURNs[r.URN()] {
			continue
		}
		modelRes, ok := r.RawData().(*dgModel.ModelResource)
		if !ok {
			continue
		}
		units = append(units, &ValidationUnit{
			ResourceType: "model",
			ID:           r.ID(),
			Resource:     modelRes,
		})
	}

	for _, r := range p.graph.ResourcesByType(relationship.HandlerMetadata.ResourceType) {
		if !modifiedURNs[r.URN()] {
			continue
		}
		relRes, ok := r.RawData().(*dgModel.RelationshipResource)
		if !ok {
			continue
		}
		units = append(units, &ValidationUnit{
			ResourceType: "relationship",
			ID:           r.ID(),
			Resource:     relRes,
		})
	}

	return &ValidationPlan{Units: units}, nil
}

func (p *Planner) planSingle(resourceType, targetID string) (*ValidationPlan, error) {
	var handlerType string
	switch resourceType {
	case "model":
		handlerType = model.HandlerMetadata.ResourceType
	case "relationship":
		handlerType = relationship.HandlerMetadata.ResourceType
	default:
		return nil, fmt.Errorf("unknown resource type: %s, must be 'model' or 'relationship'", resourceType)
	}

	urn := resources.URN(targetID, handlerType)
	r, exists := p.graph.GetResource(urn)
	if !exists {
		return nil, fmt.Errorf("resource %q of type %q not found in project", targetID, resourceType)
	}

	unit := &ValidationUnit{
		ResourceType: resourceType,
		ID:           r.ID(),
		Resource:     r.RawData(),
	}

	return &ValidationPlan{Units: []*ValidationUnit{unit}}, nil
}
