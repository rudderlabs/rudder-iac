// Package graph renders a project's resource dependency graph into
// human- and machine-readable formats (dot, mermaid, json).
//
// It is a pure renderer over the already-built resources.Graph: it performs no
// I/O beyond writing to the provided io.Writer and never mutates the graph.
//
// # JSON schema (stable contract)
//
// The json format is the machine contract consumed by the VS Code graph view
// and by agents. It MUST remain backward compatible. A document looks like:
//
//		{
//		  "nodes": [ { "urn": "event:a", "type": "event", "id": "a" } ],
//		  "edges": [ { "from": "event:a", "to": "property:b" } ],
//		  "cycles": [ "event:a → event:b → event:a" ]
//		}
//
//	  - node.urn   fully-qualified resource identifier, formatted "type:id".
//	  - node.type  resource type (the URN prefix).
//	  - node.id    resource id within its type (the URN suffix).
//	  - edge.from  URN of the dependent resource.
//	  - edge.to    URN of the dependency. "from depends on to".
//	  - cycles     each entry is a human-readable path of a circular dependency;
//	               the field is omitted when no cycles exist.
//
// nodes and edges are sorted (nodes by urn, edges by from then to) so the
// output is deterministic and diff-friendly.
package graph

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
)

// ErrUnsupportedFormat is returned when Options.Format is not one of the
// supported formats.
var ErrUnsupportedFormat = errors.New("unsupported graph format")

// Format identifies an output encoding.
type Format string

const (
	FormatDOT     Format = "dot"
	FormatMermaid Format = "mermaid"
	FormatJSON    Format = "json"
)

// Options configures a Render call.
type Options struct {
	// Format selects the output encoding. Required.
	Format Format
	// TypeFilter, when non-empty, restricts the output to nodes of this
	// resource type; edges touching filtered-out nodes are dropped.
	TypeFilter string
}

// Node is a single resource in the exported graph. See the package doc for the
// JSON schema contract.
type Node struct {
	URN  string `json:"urn"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Edge is a directed dependency: From depends on To.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Document is the json representation of the graph (the stable contract).
type Document struct {
	Nodes  []Node   `json:"nodes"`
	Edges  []Edge   `json:"edges"`
	Cycles []string `json:"cycles,omitempty"`
}

// Render writes g to w in the format selected by opts. It reuses the existing
// resources.Graph and never mutates it.
func Render(w io.Writer, g *resources.Graph, opts Options) error {
	doc := buildDocument(g, opts.TypeFilter)

	switch opts.Format {
	case FormatJSON:
		return renderJSON(w, doc)
	case FormatDOT:
		return renderDOT(w, doc)
	case FormatMermaid:
		return renderMermaid(w, doc)
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedFormat, opts.Format)
	}
}

// buildDocument extracts nodes, edges and cycles from the graph, applying the
// optional type filter and sorting everything for deterministic output.
func buildDocument(g *resources.Graph, typeFilter string) Document {
	included := make(map[string]bool)

	// Initialise as empty (non-nil) slices so the json contract always renders
	// "nodes": [] / "edges": [] rather than null, which is friendlier to the
	// VS Code view and other consumers.
	nodes := []Node{}
	for urn, r := range g.Resources() {
		if typeFilter != "" && r.Type() != typeFilter {
			continue
		}
		included[urn] = true
		nodes = append(nodes, Node{URN: urn, Type: r.Type(), ID: r.ID()})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].URN < nodes[j].URN })

	edges := []Edge{}
	for _, n := range nodes {
		for _, dep := range g.GetDependencies(n.URN) {
			// Drop edges pointing at nodes excluded by the type filter so the
			// document never references a node that isn't present.
			if !included[dep] {
				continue
			}
			edges = append(edges, Edge{From: n.URN, To: dep})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})

	// DetectCycles walks the whole graph regardless of the type filter, which is
	// intentional: a cycle can run through a filtered-out type, and surfacing it
	// is more useful than hiding it.
	var cycles []string
	if cycle, err := g.DetectCycles(); err != nil {
		cycles = append(cycles, strings.Join(cycle, " → "))
	}

	return Document{Nodes: nodes, Edges: edges, Cycles: cycles}
}
