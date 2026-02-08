package testorchestrator

import (
	"context"
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/library"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/handlers/transformation"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/model"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
)

// Mode represents the test execution mode
type Mode int

const (
	ModeSingle   Mode = iota // Test a single transformation or library by ID
	ModeAll                  // Test all transformations
	ModeModified             // Test only new or modified transformations
)

// TestUnit represents a transformation to test along with its library dependencies
type TestUnit struct {
	Transformation           *model.TransformationResource // The transformation to test
	TransformationResource   *resources.Resource           // The graph resource for the transformation
	Libraries                []*model.LibraryResource      // All libraries this transformation depends on
	ModifiedLibraryURNs      map[string]bool               // URNs of modified libraries (for staging)
	IsTransformationModified bool                          // Whether the transformation itself is modified
}

// IsLibraryModified checks if a library is modified
func (tu *TestUnit) IsLibraryModified(urn string) bool {
	return tu.ModifiedLibraryURNs[urn]
}

// TestPlan contains all test units to execute
type TestPlan struct {
	TestUnits []*TestUnit
}

// Planner determines what resources to test based on mode and diff
type Planner struct {
	graph *resources.Graph
}

// NewPlanner creates a new test planner
func NewPlanner(graph *resources.Graph) *Planner {
	return &Planner{
		graph: graph,
	}
}

// BuildPlan determines which transformations to test based on mode and remote graph
func (p *Planner) BuildPlan(ctx context.Context, remoteGraph *resources.Graph, mode Mode, targetID string, workspaceID string) (*TestPlan, error) {
	// Filter by resource type
	transformations := p.filterByType(transformation.HandlerMetadata.ResourceType)
	libraries := p.filterByType(library.HandlerMetadata.ResourceType)

	// Compute diff to identify new/updated/unmodified resources
	// source = remote graph, target = local graph
	diff := differ.ComputeDiff(remoteGraph, p.graph, differ.DiffOptions{
		WorkspaceID: workspaceID,
	})

	// Build set of modified library URNs
	modifiedLibraryURNs := p.buildModifiedResourceSet(libraries, diff)

	// Build set of modified transformation URNs
	modifiedTransformationURNs := p.buildModifiedResourceSet(transformations, diff)

	// Determine which transformations to test based on mode
	var transformationsToTest []*resources.Resource
	var err error

	switch mode {
	case ModeAll:
		transformationsToTest = transformations

	case ModeModified:
		// Filter transformations that are new or updated
		modifiedURNs := append([]string{}, diff.NewResources...)
		for urn := range diff.UpdatedResources {
			modifiedURNs = append(modifiedURNs, urn)
		}
		transformationsToTest = p.filterByURNs(transformations, modifiedURNs)

		// Also add transformations that depend on modified libraries
		for modifiedLibURN := range modifiedLibraryURNs {
			// Find the library resource
			var modifiedLibResource *resources.Resource
			for _, lib := range libraries {
				if lib.URN() == modifiedLibURN {
					modifiedLibResource = lib
					break
				}
			}

			if modifiedLibResource != nil {
				// Find all transformations that depend on this library
				dependents := p.findDependentTransformations(modifiedLibResource, transformations)
				// Add dependents if not already in the list
				for _, dep := range dependents {
					alreadyIncluded := false
					for _, existing := range transformationsToTest {
						if existing.URN() == dep.URN() {
							alreadyIncluded = true
							break
						}
					}
					if !alreadyIncluded {
						transformationsToTest = append(transformationsToTest, dep)
					}
				}
			}
		}

	case ModeSingle:
		// Find resource by ID - could be transformation or library
		resource := p.findResourceByID(targetID)
		if resource == nil {
			return nil, fmt.Errorf("resource with ID '%s' not found", targetID)
		}

		// Check if it's a library
		if resource.Type() == library.HandlerMetadata.ResourceType {
			// Find dependent transformations
			dependents := p.findDependentTransformations(resource, transformations)
			if len(dependents) == 0 {
				return nil, fmt.Errorf("library '%s' has no dependent transformations to test", targetID)
			}
			transformationsToTest = dependents
		} else {
			// It's a transformation
			transformationsToTest = []*resources.Resource{resource}
		}

	default:
		return nil, fmt.Errorf("invalid test mode: %d", mode)
	}

	if len(transformationsToTest) == 0 {
		return &TestPlan{TestUnits: []*TestUnit{}}, nil
	}

	// Build test units with library dependencies
	testUnits, err := p.buildTestUnits(transformationsToTest, libraries, modifiedTransformationURNs, modifiedLibraryURNs)
	if err != nil {
		return nil, fmt.Errorf("building test units: %w", err)
	}

	return &TestPlan{TestUnits: testUnits}, nil
}

// filterByType filters resources from the graph by resource type
func (p *Planner) filterByType(resourceType string) []*resources.Resource {
	filtered := make([]*resources.Resource, 0)
	for _, r := range p.graph.Resources() {
		if r.Type() == resourceType {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// filterByURNs filters resources to only those with URNs in the provided list
func (p *Planner) filterByURNs(resourceList []*resources.Resource, urns []string) []*resources.Resource {
	urnSet := make(map[string]bool)
	for _, urn := range urns {
		urnSet[urn] = true
	}

	filtered := make([]*resources.Resource, 0)
	for _, r := range resourceList {
		if urnSet[r.URN()] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// findResourceByID finds a resource in the graph by its ID
func (p *Planner) findResourceByID(id string) *resources.Resource {
	for _, r := range p.graph.Resources() {
		if r.ID() == id {
			return r
		}
	}
	return nil
}

// buildModifiedResourceSet creates a map of resource URNs that are new or modified
func (p *Planner) buildModifiedResourceSet(resourceList []*resources.Resource, diff *differ.Diff) map[string]bool {
	// Build URN set for O(1) lookup
	urnSet := make(map[string]bool)
	for _, r := range resourceList {
		urnSet[r.URN()] = true
	}

	modifiedSet := make(map[string]bool)

	// Add new resources that match this resource type
	for _, urn := range diff.NewResources {
		if urnSet[urn] {
			modifiedSet[urn] = true
		}
	}

	// Add updated resources that match this resource type
	for urn := range diff.UpdatedResources {
		if urnSet[urn] {
			modifiedSet[urn] = true
		}
	}

	return modifiedSet
}

// findDependentTransformations finds all transformations that depend on a library
func (p *Planner) findDependentTransformations(libraryResource *resources.Resource, transformations []*resources.Resource) []*resources.Resource {
	var dependents []*resources.Resource

	// Check each transformation's dependencies
	for _, t := range transformations {
		deps := p.graph.GetDependencies(t.URN())
		for _, dep := range deps {
			if dep == libraryResource.URN() {
				dependents = append(dependents, t)
				break
			}
		}
	}

	return dependents
}

// buildTestUnits creates test units with library dependencies from the graph
func (p *Planner) buildTestUnits(transformations []*resources.Resource, allLibraries []*resources.Resource, modifiedTransformationURNs map[string]bool, modifiedLibraryURNs map[string]bool) ([]*TestUnit, error) {
	var testUnits []*TestUnit

	for _, transformationRes := range transformations {
		// Extract transformation resource data from RawData (set by BaseHandler via WithRawData)
		transformationData, ok := transformationRes.RawData().(*model.TransformationResource)
		if !ok {
			return nil, fmt.Errorf("failed to extract transformation data for %s", transformationRes.ID())
		}

		// Get library dependencies from graph
		dependencyURNs := p.graph.GetDependencies(transformationRes.URN())

		// Filter to only library dependencies
		var libraryResources []*model.LibraryResource
		for _, depURN := range dependencyURNs {
			// Find the library resource
			for _, libRes := range allLibraries {
				if libRes.URN() == depURN {
					libData, ok := libRes.RawData().(*model.LibraryResource)
					if !ok {
						return nil, fmt.Errorf("failed to extract library data for %s", libRes.ID())
					}
					libraryResources = append(libraryResources, libData)
					break
				}
			}
		}

		// Check if transformation is modified
		isTransformationModified := modifiedTransformationURNs[transformationRes.URN()]

		// Create test unit
		testUnit := &TestUnit{
			Transformation:           transformationData,
			TransformationResource:   transformationRes,
			Libraries:                libraryResources,
			ModifiedLibraryURNs:      modifiedLibraryURNs,
			IsTransformationModified: isTransformationModified,
		}
		testUnits = append(testUnits, testUnit)
	}

	return testUnits, nil
}
