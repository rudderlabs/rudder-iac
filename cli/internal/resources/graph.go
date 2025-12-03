package resources

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
	refs := collectReferences(r.Data())

	if r.RawData() != nil {
		refs = append(refs, collectReferencesByReflection(r.RawData())...)
	}

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
