package funcs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// GraphWith is a test helper function that builds a resource graph from alternating
// (id, resourceType) pairs. It is intended for use in tests only.
//
// Example: GraphWith("user_id", "property", "signup", "event")
func GraphWith(pairs ...string) *resources.Graph {
	g := resources.NewGraph()
	for i := 0; i+1 < len(pairs); i += 2 {
		g.AddResource(resources.NewResource(pairs[i], pairs[i+1], resources.ResourceData{}, nil))
	}
	return g
}
