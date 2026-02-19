package testorchestrator

import (
	"context"
	"fmt"

	"github.com/samber/lo"

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
	ModeAll                  // Test all transformations and standalone libraries
	ModeModified             // Test only new or modified transformations
)

// TestUnit represents a transformation to test along with its library dependencies
type TestUnit struct {
	Transformation           *model.TransformationResource
	Libraries                []*model.LibraryResource
	IsTransformationModified bool
}

// TestPlan contains all test units to execute
type TestPlan struct {
	TestUnits                  []*TestUnit
	StandaloneLibraries        []*model.LibraryResource // Libraries with no dependent transformations
	ModifiedLibraryURNs        map[string]bool
	ModifiedTransformationURNs map[string]bool
}

// IsLibraryModified checks if a library is modified
func (tp *TestPlan) IsLibraryModified(urn string) bool {
	return tp.ModifiedLibraryURNs[urn]
}

// IsTransformationModified checks if a transformation is modified
func (tp *TestPlan) IsTransformationModified(urn string) bool {
	return tp.ModifiedTransformationURNs[urn]
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
	transformations := p.graph.ResourcesByType(transformation.HandlerMetadata.ResourceType)
	libraries := p.graph.ResourcesByType(library.HandlerMetadata.ResourceType)

	// Compute diff to identify new/updated/unmodified resources
	diff := differ.ComputeDiff(remoteGraph, p.graph, differ.DiffOptions{
		WorkspaceID: workspaceID,
	})

	modifiedTransformationURNs, modifiedLibraryURNs := p.buildModifiedResourceSets(diff, remoteGraph)

	var (
		transformationsToTest []*resources.Resource
		standaloneLibs        []*resources.Resource
	)

	switch mode {
	case ModeAll:
		transformationsToTest = transformations

		// Identify standalone libraries (not depended on by any transformation)
		dependedLibURNs := lo.FlatMap(transformations, func(t *resources.Resource, _ int) []string {
			return p.graph.GetDependencies(t.URN())
		})
		standaloneLibs = lo.Filter(libraries, func(lib *resources.Resource, _ int) bool {
			return !lo.Contains(dependedLibURNs, lib.URN())
		})

	case ModeModified:
		transformationsToTest = p.filterByURNs(transformations, lo.Keys(modifiedTransformationURNs))

		// Also add transformations that depend on modified libraries
		for modifiedLibURN := range modifiedLibraryURNs {
			library, _ := p.graph.GetResource(modifiedLibURN)

			dependents := p.findDependentTransformations(library, transformations)
			transformationsToTest = append(transformationsToTest, dependents...)
			// Modified library with no dependents — test it standalone
			if len(dependents) == 0 {
				standaloneLibs = append(standaloneLibs, library)
			}
		}
		transformationsToTest = lo.UniqBy(transformationsToTest, func(r *resources.Resource) string {
			return r.URN()
		})

	case ModeSingle:
		resource := p.findResourceByID(targetID)
		if resource == nil {
			return nil, fmt.Errorf("resource with ID '%s' not found", targetID)
		}

		transformationsToTest = []*resources.Resource{resource}
		if resource.Type() == library.HandlerMetadata.ResourceType {
			dependents := p.findDependentTransformations(resource, transformations)
			transformationsToTest = dependents
			// Add library with no dependents to standalone list
			if len(dependents) == 0 {
				standaloneLibs = []*resources.Resource{resource}
			}
		}
	}

	// Build library lookup once: URN -> LibraryResource
	libByURN, err := p.buildLibraryLookup(libraries)
	if err != nil {
		return nil, err
	}

	standaloneLibraryModels := lo.FilterMap(standaloneLibs, func(r *resources.Resource, _ int) (*model.LibraryResource, bool) {
		lib, ok := libByURN[r.URN()]
		return lib, ok
	})

	testUnits, err := p.buildTestUnits(transformationsToTest, libByURN, modifiedTransformationURNs)
	if err != nil {
		return nil, fmt.Errorf("building test units: %w", err)
	}

	return &TestPlan{
		TestUnits:                  testUnits,
		StandaloneLibraries:        standaloneLibraryModels,
		ModifiedLibraryURNs:        modifiedLibraryURNs,
		ModifiedTransformationURNs: modifiedTransformationURNs,
	}, nil
}

// filterByURNs filters resources to only those with URNs in the provided list
func (p *Planner) filterByURNs(resourceList []*resources.Resource, urns []string) []*resources.Resource {
	return lo.Filter(resourceList, func(r *resources.Resource, _ int) bool {
		return lo.Contains(urns, r.URN())
	})
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


// buildModifiedResourceSets identifies modified transformation and library URNs.
// New resources are always considered modified. Updated and importable resources
// are only considered modified if their code or name has changed.
func (p *Planner) buildModifiedResourceSets(diff *differ.Diff, remoteGraph *resources.Graph) (modifiedTransformationURNs, modifiedLibraryURNs map[string]bool) {
	modifiedTransformationURNs = make(map[string]bool)
	modifiedLibraryURNs = make(map[string]bool)

	addModified := func(urn string, resourceType string) {
		switch resourceType {
		case transformation.HandlerMetadata.ResourceType:
			modifiedTransformationURNs[urn] = true
		case library.HandlerMetadata.ResourceType:
			modifiedLibraryURNs[urn] = true
		}
	}

	// New resources are always modified
	for _, urn := range diff.NewResources {
		r, exists := p.graph.GetResource(urn)
		if !exists {
			continue
		}
		addModified(urn, r.Type())
	}

	// Updated and importable resources are modified only if code or name changed
	candidateURNs := append(lo.Keys(diff.UpdatedResources), diff.ImportableResources...)
	for _, urn := range candidateURNs {
		resource, exists := p.graph.GetResource(urn)
		remote, remoteExists := remoteGraph.GetResource(urn)
		if !exists || !remoteExists {
			continue
		}

		if isModified(resource, remote, resource.Type()) {
			addModified(urn, resource.Type())
		}
	}

	return modifiedTransformationURNs, modifiedLibraryURNs
}

func isModified(resource, remote *resources.Resource, resourceType string) bool {
	switch resourceType {
	case transformation.HandlerMetadata.ResourceType:
		resourceData, lok := resource.RawData().(*model.TransformationResource)
		remoteData, rok := remote.RawData().(*model.TransformationResource)
		if !lok || !rok {
			return true
		}
		return resourceData.Name != remoteData.Name || resourceData.Code != remoteData.Code

	case library.HandlerMetadata.ResourceType:
		resourceData, lok := resource.RawData().(*model.LibraryResource)
		remoteData, rok := remote.RawData().(*model.LibraryResource)
		if !lok || !rok {
			return true
		}
		return resourceData.Name != remoteData.Name || resourceData.Code != remoteData.Code
	}

	return true
}

// findDependentTransformations finds all transformations that depend on a library
func (p *Planner) findDependentTransformations(libraryResource *resources.Resource, transformations []*resources.Resource) []*resources.Resource {
	return lo.Filter(transformations, func(t *resources.Resource, _ int) bool {
		return lo.Contains(p.graph.GetDependencies(t.URN()), libraryResource.URN())
	})
}


// buildLibraryLookup extracts model data from library resources into a URN-keyed map
func (p *Planner) buildLibraryLookup(libraries []*resources.Resource) (map[string]*model.LibraryResource, error) {
	libByURN := make(map[string]*model.LibraryResource, len(libraries))
	for _, libRes := range libraries {
		libData, ok := libRes.RawData().(*model.LibraryResource)
		if !ok {
			return nil, fmt.Errorf("extracting library data for %s", libRes.ID())
		}
		libByURN[libRes.URN()] = libData
	}
	return libByURN, nil
}

// buildTestUnits creates test units with library dependencies from the graph
func (p *Planner) buildTestUnits(transformations []*resources.Resource, libByURN map[string]*model.LibraryResource, modifiedTransformationURNs map[string]bool) ([]*TestUnit, error) {
	testUnits := make([]*TestUnit, 0, len(transformations))
	for _, transformationRes := range transformations {
		transformationData, ok := transformationRes.RawData().(*model.TransformationResource)
		if !ok {
			return nil, fmt.Errorf("extracting transformation data for %s", transformationRes.ID())
		}

		deps := p.graph.GetDependencies(transformationRes.URN())
		libs := lo.FilterMap(deps, func(depURN string, _ int) (*model.LibraryResource, bool) {
			lib, ok := libByURN[depURN]
			return lib, ok
		})

		testUnits = append(testUnits, &TestUnit{
			Transformation:           transformationData,
			Libraries:                libs,
			IsTransformationModified: modifiedTransformationURNs[transformationRes.URN()],
		})
	}

	return testUnits, nil
}
