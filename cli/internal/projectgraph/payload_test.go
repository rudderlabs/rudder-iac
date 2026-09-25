package projectgraph_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/projectgraph"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/pathindex"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_NilGraphStillReportsDiagnostics(t *testing.T) {
	diagnostics := validation.Diagnostics{{
		RuleID:   "project/spec-syntax-parse-valid",
		Severity: rules.Error,
		Message:  "failed to parse spec",
		File:     "events.yaml",
		Position: pathindex.Position{Line: 3, Column: 1},
	}}

	payload := projectgraph.Build(nil, diagnostics, nil, "1.2.3", "./specs")

	assert.Equal(t, projectgraph.Payload{
		SchemaVersion: projectgraph.SchemaVersion,
		CLIVersion:    "1.2.3",
		Location:      "./specs",
		Nodes:         []projectgraph.Node{},
		Edges:         []projectgraph.Edge{},
		Diagnostics: []projectgraph.Diagnostic{{
			RuleID:   "project/spec-syntax-parse-valid",
			Severity: "error",
			Message:  "failed to parse spec",
			File:     "events.yaml",
			Line:     3,
			Column:   1,
		}},
	}, payload)
}

func TestBuild_NodesEdgesAndLocations(t *testing.T) {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("signup", "event", resources.ResourceData{"name": "Signup"}, []string{}))
	graph.AddResource(resources.NewResource("email", "property", resources.ResourceData{}, []string{}))
	graph.AddDependency("event:signup", "property:email")

	locations := map[string]projectgraph.Location{
		"event:signup": {File: "events.yaml", Line: 7, Column: 5},
	}

	payload := projectgraph.Build(graph, nil, locations, "1.2.3", ".")

	assert.Equal(t, []projectgraph.Node{
		{
			URN:         "event:signup",
			ID:          "signup",
			Type:        "event",
			DisplayName: "Signup",
			File:        "events.yaml",
			Line:        7,
			Column:      5,
		},
		{
			// No location entry and no name: falls back to the id, no position.
			URN:         "property:email",
			ID:          "email",
			Type:        "property",
			DisplayName: "email",
		},
	}, payload.Nodes)

	assert.Equal(t, []projectgraph.Edge{
		{From: "event:signup", To: "property:email"},
	}, payload.Edges)
}

// Consumers cache on content, so the same project must always serialise
// identically regardless of Go's map iteration order.
func TestBuild_OutputIsDeterministic(t *testing.T) {
	build := func() projectgraph.Payload {
		graph := resources.NewGraph()
		for _, id := range []string{"zulu", "alpha", "mike"} {
			graph.AddResource(resources.NewResource(id, "event", resources.ResourceData{}, []string{}))
		}
		graph.AddResource(resources.NewResource("tp", "tracking-plan", resources.ResourceData{}, []string{}))
		graph.AddDependencies("tracking-plan:tp", []string{"event:zulu", "event:alpha", "event:mike"})
		return projectgraph.Build(graph, nil, nil, "1.2.3", ".")
	}

	first := build()
	for range 20 {
		require.Equal(t, first, build())
	}

	assert.Equal(t, []projectgraph.Edge{
		{From: "tracking-plan:tp", To: "event:alpha"},
		{From: "tracking-plan:tp", To: "event:mike"},
		{From: "tracking-plan:tp", To: "event:zulu"},
	}, first.Edges)
}

func TestBuild_ReportsCycleWithoutDroppingTheGraph(t *testing.T) {
	graph := resources.NewGraph()
	graph.AddResource(resources.NewResource("a", "event", resources.ResourceData{}, []string{}))
	graph.AddResource(resources.NewResource("b", "event", resources.ResourceData{}, []string{}))
	graph.AddDependency("event:a", "event:b")
	graph.AddDependency("event:b", "event:a")

	payload := projectgraph.Build(graph, nil, nil, "1.2.3", ".")

	assert.NotEmpty(t, payload.Cycle, "a cycle must be reported")
	assert.Len(t, payload.Nodes, 2, "the graph is still returned so the reader can see the cycle")
}

func TestBuild_DisplayNamePrefersNameOverID(t *testing.T) {
	for _, tc := range []struct {
		name     string
		data     resources.ResourceData
		expected string
	}{
		{"name wins", resources.ResourceData{"name": "Signup"}, "Signup"},
		{"snake case display_name", resources.ResourceData{"display_name": "Sign Up"}, "Sign Up"},
		{"camel case displayName", resources.ResourceData{"displayName": "Sign Up"}, "Sign Up"},
		{"empty name falls back to id", resources.ResourceData{"name": ""}, "signup"},
		{"non-string name falls back to id", resources.ResourceData{"name": 42}, "signup"},
		{"no name at all", resources.ResourceData{}, "signup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := resources.NewGraph()
			graph.AddResource(resources.NewResource("signup", "event", tc.data, []string{}))

			payload := projectgraph.Build(graph, nil, nil, "1.2.3", ".")

			require.Len(t, payload.Nodes, 1)
			assert.Equal(t, tc.expected, payload.Nodes[0].DisplayName)
		})
	}
}
