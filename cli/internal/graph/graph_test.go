package graph

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildGraph returns a small graph:
//
//	event:a  ──▶  property:b
//	tracking-plan:c ──▶ event:a
//
// where "──▶" means "depends on".
func buildGraph(t *testing.T) *resources.Graph {
	t.Helper()

	g := resources.NewGraph()
	g.AddResource(resources.NewResource("b", "property", resources.ResourceData{}, nil))
	g.AddResource(resources.NewResource("a", "event", resources.ResourceData{}, nil))
	g.AddResource(resources.NewResource("c", "tracking-plan", resources.ResourceData{}, nil))
	g.AddDependency("event:a", "property:b")
	g.AddDependency("tracking-plan:c", "event:a")
	return g
}

func TestRenderJSON_Schema(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, buildGraph(t), Options{Format: FormatJSON})
	require.NoError(t, err)

	var doc Document
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))

	// Nodes and edges are sorted deterministically so the contract is stable.
	assert.Equal(t, Document{
		Nodes: []Node{
			{URN: "event:a", Type: "event", ID: "a"},
			{URN: "property:b", Type: "property", ID: "b"},
			{URN: "tracking-plan:c", Type: "tracking-plan", ID: "c"},
		},
		Edges: []Edge{
			{From: "event:a", To: "property:b"},
			{From: "tracking-plan:c", To: "event:a"},
		},
		Cycles: nil,
	}, doc)
}

func TestRenderJSON_KeysAreStable(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, buildGraph(t), Options{Format: FormatJSON})
	require.NoError(t, err)

	// Pin the exact JSON key names since the VS Code view depends on them.
	out := buf.String()
	for _, key := range []string{`"nodes"`, `"edges"`, `"urn"`, `"type"`, `"id"`, `"from"`, `"to"`} {
		assert.Contains(t, out, key)
	}
}

func TestRenderJSON_EmptyGraphUsesArraysNotNull(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, resources.NewGraph(), Options{Format: FormatJSON})
	require.NoError(t, err)

	// The contract requires empty arrays, never null, so consumers can iterate
	// without a nil check.
	out := buf.String()
	assert.Contains(t, out, `"nodes": []`)
	assert.Contains(t, out, `"edges": []`)
	assert.NotContains(t, out, "null")
}

func TestRenderJSON_ReportsCycles(t *testing.T) {
	g := resources.NewGraph()
	g.AddResource(resources.NewResource("a", "event", resources.ResourceData{}, nil))
	g.AddResource(resources.NewResource("b", "event", resources.ResourceData{}, nil))
	g.AddDependency("event:a", "event:b")
	g.AddDependency("event:b", "event:a")

	var buf bytes.Buffer
	err := Render(&buf, g, Options{Format: FormatJSON})
	require.NoError(t, err)

	var doc Document
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))
	require.NotEmpty(t, doc.Cycles, "cycle should be surfaced, not cause a hang")
	assert.Contains(t, doc.Cycles[0], "event:a")
}

func TestRenderDOT(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, buildGraph(t), Options{Format: FormatDOT})
	require.NoError(t, err)

	out := buf.String()
	assert.True(t, strings.HasPrefix(out, "digraph"))
	assert.Contains(t, out, `"event:a" -> "property:b"`)
	assert.Contains(t, out, `"tracking-plan:c" -> "event:a"`)
	// URN used as node label.
	assert.Contains(t, out, `label="event:a"`)
	assert.True(t, strings.HasSuffix(strings.TrimSpace(out), "}"))
}

func TestRenderMermaid(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, buildGraph(t), Options{Format: FormatMermaid})
	require.NoError(t, err)

	out := buf.String()
	assert.True(t, strings.HasPrefix(strings.TrimSpace(out), "graph TD"))
	assert.Contains(t, out, "-->")
	assert.Contains(t, out, "event:a")
}

func TestRender_TypeFilter(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, buildGraph(t), Options{Format: FormatJSON, TypeFilter: "event"})
	require.NoError(t, err)

	var doc Document
	require.NoError(t, json.Unmarshal(buf.Bytes(), &doc))

	// Only event nodes remain; edges touching filtered-out nodes are dropped.
	assert.Equal(t, []Node{{URN: "event:a", Type: "event", ID: "a"}}, doc.Nodes)
	assert.Empty(t, doc.Edges)
}

func TestRender_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, buildGraph(t), Options{Format: "yaml"})
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}
