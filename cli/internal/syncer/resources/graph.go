package resources

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

func (s *Graph) AddResource(r *Resource) {
	s.resources[r.URN()] = r
}

func (s *Graph) GetResource(urn string) (*Resource, bool) {
	r, exists := s.resources[urn]
	return r, exists
}
