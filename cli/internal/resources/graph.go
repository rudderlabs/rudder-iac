package resources

import (
	"fmt"
	"strings"
)

type Graph struct {
	resources    map[string]*Resource
	dependencies map[string][]string
	dependents   map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		resources:    map[string]*Resource{},
		dependencies: map[string][]string{},
		dependents:   map[string][]string{},
	}
}

func (s *Graph) Resources() map[string]*Resource {
	return s.resources
}

func (s *Graph) AddResource(r *Resource) {
	refs := CollectReferences(r.Data())
	for _, ref := range refs {
		s.AddDependency(r.URN(), ref.URN)
	}

	s.resources[r.URN()] = r
}

func (s *Graph) GetResource(urn string) (*Resource, bool) {
	r, exists := s.resources[urn]
	return r, exists
}

func (s *Graph) GetDependencies(urn string) []string {
	return s.dependencies[urn]
}

func (s *Graph) GetDependents(urn string) []string {
	return s.dependents[urn]
}

func (s *Graph) AddDependency(addedTo string, dependency string) {
	deps := s.dependencies[addedTo]
	for _, dep := range deps {
		if dep == dependency {
			return
		}
	}
	s.dependencies[addedTo] = append(s.dependencies[addedTo], dependency)
	s.addDependent(dependency, addedTo)
}

func (s *Graph) AddDependencies(addedTo string, dependencies []string) {
	for _, dep := range dependencies {
		s.AddDependency(addedTo, dep)
	}
}

func (s *Graph) addDependent(dependency string, dependent string) {
	deps := s.dependents[dependency]
	for _, dep := range deps {
		if dep == dependent {
			return
		}
	}
	s.dependents[dependency] = append(s.dependents[dependency], dependent)
}

func (s *Graph) Merge(g *Graph) {
	for _, r := range g.resources {
		s.AddResource(r)
	}

	for k, v := range g.dependencies {
		for _, dep := range v {
			s.AddDependency(k, dep)
		}
	}
}

// DetectCycles detects circular dependencies in the resource graph using Depth-First Search (DFS).
// It returns the first cycle found as a slice of URNs showing the circular path, or nil if no cycles exist.
//
// Algorithm: Uses DFS to detect back edges (cycles) by checking if a dependency is already in the current path.
// - alreadyVisited: tracks all nodes we've explored (prevents revisiting completed branches)
// - visitedPath: the current path from root to current node (used to detect and report cycles)
func (g *Graph) DetectCycles() ([]string, error) {
	// Track all nodes we've visited during the entire DFS traversal
	alreadyVisited := make(map[string]bool)

	// Helper function to check if a URN is in the current path
	containsURN := func(path []string, urn string) bool {
		for _, pathURN := range path {
			if pathURN == urn {
				return true
			}
		}
		return false
	}

	// Recursive function to detect cycles starting from a given resource
	var detectCycleFromNode func(resourceURN string, visitedPath []string) []string
	detectCycleFromNode = func(resourceURN string, visitedPath []string) []string {
		// Mark this node as visited and add it to the current path
		alreadyVisited[resourceURN] = true
		visitedPath = append(visitedPath, resourceURN)

		// Check all dependencies of this resource
		for _, dependencyURN := range g.GetDependencies(resourceURN) {
			// Check if this dependency is already in our current path
			// If yes, we've found a cycle!
			if containsURN(visitedPath, dependencyURN) {
				// Example: path = [A, B, C], dependency = A
				// This means C depends on A, creating cycle: A → B → C → A

				// Find where the cycle starts in our path
				cycleStartIndex := -1
				for i, pathURN := range visitedPath {
					if pathURN == dependencyURN {
						cycleStartIndex = i
						break
					}
				}

				// Extract and return the cycle: [A, B, C, A]
				cyclePath := append(visitedPath[cycleStartIndex:], dependencyURN)
				return cyclePath
			}

			// If we haven't visited this dependency yet, recurse into it
			if !alreadyVisited[dependencyURN] {
				if cyclePath := detectCycleFromNode(dependencyURN, visitedPath); cyclePath != nil {
					return cyclePath
				}
			}
			// If already visited but not in current path, it's from a different branch - safe to ignore
		}

		// No cycle found from this node
		return nil
	}

	// Check each resource in the graph (handles disconnected components)
	for resourceURN := range g.resources {
		if !alreadyVisited[resourceURN] {
			if cyclePath := detectCycleFromNode(resourceURN, []string{}); cyclePath != nil {
				return cyclePath, fmt.Errorf("circular dependency detected: %s", strings.Join(cyclePath, " → "))
			}
		}
	}

	// No cycles found
	return nil, nil
}
