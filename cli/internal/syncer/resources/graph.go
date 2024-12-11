package resources

import "fmt"

type Graph struct {
	resources    map[string]*Resource
	dependencies map[string][]string
}

func NewGraph() *Graph {
	return &Graph{
		resources:    map[string]*Resource{},
		dependencies: map[string][]string{},
	}
}

func (s *Graph) Resources() map[string]*Resource {
	return s.resources
}

func (s *Graph) AddResource(r *Resource) error {
	refs := CollectReferences(r.Data())
	for _, ref := range refs {
		if _, exists := s.resources[ref.URN]; !exists {
			return fmt.Errorf("referred resource '%s' does not exist", ref.URN)
		}
		s.dependencies[r.URN()] = append(s.dependencies[r.URN()], ref.URN)
	}

	s.resources[r.URN()] = r

	return nil
}

func (s *Graph) GetResource(urn string) (*Resource, bool) {
	r, exists := s.resources[urn]
	return r, exists
}

func (s *Graph) GetDependencies(urn string) []string {
	return s.dependencies[urn]
}

func (s *Graph) AddDependencies(addedTo string, dependencies []string) {
	s.dependencies[addedTo] = append(s.dependencies[addedTo], dependencies...)
}
