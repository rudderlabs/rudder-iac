package validator

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/handlers/relationship"
	dgModel "github.com/rudderlabs/rudder-iac/cli/internal/providers/datagraph/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// ValidationUnit represents a single resource to validate
type ValidationUnit struct {
	ResourceType string // "model" or "relationship"
	ID           string // local resource ID
	URN          string // resource URN (resourceType:ID)
	Resource     any    // *dgModel.ModelResource or *dgModel.RelationshipResource
	AccountID    string // resolved account ID from the parent data graph spec
}

// ValidationPlan holds the units to validate
type ValidationPlan struct {
	Units []*ValidationUnit
}

// PlanAll builds a plan containing all model and relationship resources in the graph.
func PlanAll(graph *resources.Graph) (*ValidationPlan, error) {
	var allResources []*resources.Resource
	allResources = append(allResources, graph.ResourcesByType(model.HandlerMetadata.ResourceType)...)
	allResources = append(allResources, graph.ResourcesByType(relationship.HandlerMetadata.ResourceType)...)

	units := resourcesToUnits(allResources)
	return &ValidationPlan{Units: units}, nil
}

// PlanModified builds a plan containing only resources that differ from the remote graph.
func PlanModified(graph *resources.Graph, remoteGraph *resources.Graph, opts differ.DiffOptions) (*ValidationPlan, error) {
	diff := differ.ComputeDiff(remoteGraph, graph, opts)

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

	var modified []*resources.Resource
	for _, r := range graph.ResourcesByType(model.HandlerMetadata.ResourceType) {
		if modifiedURNs[r.URN()] {
			modified = append(modified, r)
		}
	}
	for _, r := range graph.ResourcesByType(relationship.HandlerMetadata.ResourceType) {
		if modifiedURNs[r.URN()] {
			modified = append(modified, r)
		}
	}

	units := resourcesToUnits(modified)
	return &ValidationPlan{Units: units}, nil
}

// PlanSingle builds a plan containing a single resource identified by type and ID.
func PlanSingle(graph *resources.Graph, resourceType, targetID string) (*ValidationPlan, error) {
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
	r, exists := graph.GetResource(urn)
	if !exists {
		return nil, fmt.Errorf("resource %q of type %q not found in project", targetID, resourceType)
	}

	units := resourcesToUnits([]*resources.Resource{r})
	return &ValidationPlan{Units: units}, nil
}

// resourcesToUnits converts graph resources to validation units, filtering out
// resources that don't match expected model/relationship types.
func resourcesToUnits(rs []*resources.Resource) []*ValidationUnit {
	var units []*ValidationUnit
	for _, r := range rs {
		var (
			resourceType string
			resource     any
		)

		switch raw := r.RawData().(type) {
		case *dgModel.ModelResource:
			resourceType = "model"
			resource = raw
		case *dgModel.RelationshipResource:
			resourceType = "relationship"
			resource = raw
		default:
			continue
		}

		units = append(units, &ValidationUnit{
			ResourceType: resourceType,
			ID:           r.ID(),
			URN:          r.URN(),
			Resource:     resource,
		})
	}
	return units
}
