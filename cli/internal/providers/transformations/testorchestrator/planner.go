package testorchestrator

import (
	"context"
	"fmt"

	"github.com/samber/lo"

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
	Transformation *model.TransformationResource
	Libraries      []*model.LibraryResource
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
	graph              *resources.Graph
	transformationType string
	libraryType        string
}

// NewPlanner creates a new test planner
func NewPlanner(graph *resources.Graph) *Planner {
	return &Planner{
		graph:              graph,
		transformationType: "transformation",
		libraryType:        "transformation-library",
	}
}

// BuildPlan determines which transformations to test based on mode and remote graph
func (p *Planner) BuildPlan(ctx context.Context, remoteGraph *resources.Graph, mode Mode, targetID string, workspaceID string) (*TestPlan, error) {
	var (
		transformations = p.graph.ResourcesByType(p.transformationType)
		libraries       = p.graph.ResourcesByType(p.libraryType)
	)

	// Compute diff to identify new/updated/unmodified resources
	diff := differ.ComputeDiff(remoteGraph, p.graph, differ.DiffOptions{
		WorkspaceID: workspaceID,
	})

	modifiedTransformationURNs, modifiedLibraryURNs := p.buildModifiedResourceSets(diff, remoteGraph)

	var (
		transformationsToTest []*resources.Resource
		standaloneLibs        []*resources.Resource
		err                   error
	)

	switch mode {
	case ModeAll:
		transformationsToTest, standaloneLibs = p.planAll(transformations, libraries)
	case ModeModified:
		transformationsToTest, standaloneLibs = p.planModified(transformations, modifiedTransformationURNs, modifiedLibraryURNs)
	case ModeSingle:
		transformationsToTest, standaloneLibs, err = p.planSingle(targetID, transformations)
		if err != nil {
			return nil, err
		}
	}

	// Build library lookup once: URN -> LibraryResource
	libByURN, err := p.buildLibraryLookup(libraries)
	if err != nil {
		return nil, err
	}

	standaloneLibraries := lo.FilterMap(standaloneLibs, func(r *resources.Resource, _ int) (*model.LibraryResource, bool) {
		lib, ok := libByURN[r.URN()]
		return lib, ok
	})

	testUnits, err := p.buildTestUnits(transformationsToTest, libByURN)
	if err != nil {
		return nil, fmt.Errorf("building test units: %w", err)
	}

	return &TestPlan{
		TestUnits:                  testUnits,
		StandaloneLibraries:        standaloneLibraries,
		ModifiedLibraryURNs:        modifiedLibraryURNs,
		ModifiedTransformationURNs: modifiedTransformationURNs,
	}, nil
}

func (p *Planner) planAll(transformations, libraries []*resources.Resource) (transformationsToTest, standaloneLibs []*resources.Resource) {
	dependedLibURNs := lo.FlatMap(transformations, func(t *resources.Resource, _ int) []string {
		return p.graph.GetDependencies(t.URN())
	})
	standaloneLibs = lo.Filter(libraries, func(lib *resources.Resource, _ int) bool {
		return !lo.Contains(dependedLibURNs, lib.URN())
	})
	return transformations, standaloneLibs
}

func (p *Planner) planModified(transformations []*resources.Resource, modifiedTransformationURNs, modifiedLibraryURNs map[string]bool) (transformationsToTest, standaloneLibs []*resources.Resource) {
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
	return transformationsToTest, standaloneLibs
}

func (p *Planner) planSingle(targetID string, transformations []*resources.Resource) (transformationsToTest, standaloneLibs []*resources.Resource, err error) {
	resource := p.findResourceByID(targetID)
	if resource == nil {
		return nil, nil, fmt.Errorf("resource with ID '%s' not found", targetID)
	}

	transformationsToTest = []*resources.Resource{resource}
	if resource.Type() == p.libraryType {
		dependents := p.findDependentTransformations(resource, transformations)
		transformationsToTest = dependents
		// Add library with no dependents to standalone list
		if len(dependents) == 0 {
			standaloneLibs = []*resources.Resource{resource}
		}
	}
	return transformationsToTest, standaloneLibs, nil
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
	var (
		transformationURNs = make(map[string]bool)
		libraryURNs        = make(map[string]bool)
	)

	for _, urn := range diff.NewResources {
		r, _ := p.graph.GetResource(urn)

		switch r.Type() {
		case p.transformationType:
			transformationURNs[urn] = true

		case p.libraryType:
			libraryURNs[urn] = true
		}
	}

	// Importable resources are considered modified if remote resource do not exist or
	// if they exist in the remote graph and code or name has changed
	for _, urn := range diff.ImportableResources {
		resource, _ := p.graph.GetResource(urn)
		remote, _ := remoteGraph.GetResource(urn)

		modified := remote != nil && p.isModified(resource, remote, resource.Type())
		if remote == nil || modified {
			switch resource.Type() {
			case p.transformationType:
				transformationURNs[urn] = true

			case p.libraryType:
				libraryURNs[urn] = true
			}
		}
	}

	// Updated resources are modified only if code or name changed
	for _, urn := range lo.Keys(diff.UpdatedResources) {
		resource, _ := p.graph.GetResource(urn)
		remote, _ := remoteGraph.GetResource(urn)

		if p.isModified(resource, remote, resource.Type()) {
			switch resource.Type() {
			case p.transformationType:
				transformationURNs[urn] = true

			case p.libraryType:
				libraryURNs[urn] = true
			}
		}
	}

	return transformationURNs, libraryURNs
}

func (p *Planner) isModified(resource, remote *resources.Resource, resourceType string) bool {
	switch resourceType {
	case p.transformationType:
		resourceData, _ := resource.RawData().(*model.TransformationResource)
		remoteData, _ := remote.RawData().(*model.TransformationResource)

		return resourceData.Name != remoteData.Name || resourceData.Code != remoteData.Code

	case p.libraryType:
		resourceData, _ := resource.RawData().(*model.LibraryResource)
		remoteData, _ := remote.RawData().(*model.LibraryResource)

		return resourceData.Name != remoteData.Name || resourceData.Code != remoteData.Code
	}

	return true
}

func (p *Planner) findDependentTransformations(library *resources.Resource, transformations []*resources.Resource) []*resources.Resource {
	return lo.Filter(transformations, func(t *resources.Resource, _ int) bool {
		return lo.Contains(p.graph.GetDependencies(t.URN()), library.URN())
	})
}

// URN to LibraryResource map
func (p *Planner) buildLibraryLookup(libraries []*resources.Resource) (map[string]*model.LibraryResource, error) {
	libByURN := make(map[string]*model.LibraryResource, len(libraries))
	for _, library := range libraries {
		libData, ok := library.RawData().(*model.LibraryResource)
		if !ok {
			return nil, fmt.Errorf("extracting library data for %s", library.ID())
		}
		libByURN[library.URN()] = libData
	}
	return libByURN, nil
}

// buildTestUnits creates test units with library dependencies from the graph
func (p *Planner) buildTestUnits(transformations []*resources.Resource, libByURN map[string]*model.LibraryResource) ([]*TestUnit, error) {
	testUnits := make([]*TestUnit, 0, len(transformations))
	for _, transformation := range transformations {
		transData, ok := transformation.RawData().(*model.TransformationResource)
		if !ok {
			return nil, fmt.Errorf("extracting transformation data for %s", transformation.ID())
		}

		deps := p.graph.GetDependencies(transformation.URN())
		libs := lo.FilterMap(deps, func(depURN string, _ int) (*model.LibraryResource, bool) {
			lib, ok := libByURN[depURN]
			return lib, ok
		})

		testUnits = append(testUnits, &TestUnit{
			Transformation: transData,
			Libraries:      libs,
		})
	}

	return testUnits, nil
}
