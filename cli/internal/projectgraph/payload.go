// Package projectgraph renders a loaded project's resource graph and validation
// diagnostics as a stable JSON document for machine consumers — chiefly the
// VS Code extension, which owns none of the spec semantics itself and relies on
// this payload for every resource, relationship and error it displays.
package projectgraph

import (
	"sort"

	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation"
)

// SchemaVersion is the contract version of Payload. Consumers pin it and
// degrade when they see a version they do not know. Bump it on any change that
// is not purely additive.
const SchemaVersion = 1

// Payload is the complete machine-readable view of a loaded project.
type Payload struct {
	SchemaVersion int          `json:"schemaVersion"`
	CLIVersion    string       `json:"cliVersion"`
	Location      string       `json:"location"`
	Nodes         []Node       `json:"nodes"`
	Edges         []Edge       `json:"edges"`
	Cycle         []string     `json:"cycle,omitempty"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
}

// Node is one resource in the graph, addressed by its URN.
type Node struct {
	URN         string `json:"urn"`
	ID          string `json:"id"`
	Type        string `json:"type"`
	DisplayName string `json:"displayName"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
}

// Edge is a dependency relation: From depends on To. Renderers that want data
// to flow left-to-right reverse it; the payload states the relation the graph
// actually holds rather than a presentation of it.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Diagnostic is a validation finding, flattened to what an editor needs to
// place a squiggle.
type Diagnostic struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

// Location is where a URN is declared. It mirrors project.SourceLocation, kept
// separate so this package does not import project and invert the dependency.
type Location struct {
	File   string
	Line   int
	Column int
}

// Build assembles the payload. graph may be nil — a project whose syntax
// validation failed never reaches graph construction, and the diagnostics
// explaining why are still worth returning.
//
// Output is fully sorted so that repeated runs over an unchanged project
// produce byte-identical JSON, which lets consumers cache on content.
func Build(
	graph *resources.Graph,
	diagnostics validation.Diagnostics,
	locations map[string]Location,
	cliVersion string,
	location string,
) Payload {
	payload := Payload{
		SchemaVersion: SchemaVersion,
		CLIVersion:    cliVersion,
		Location:      location,
		Nodes:         make([]Node, 0),
		Edges:         make([]Edge, 0),
		Diagnostics:   make([]Diagnostic, 0, len(diagnostics)),
	}

	for _, d := range diagnostics {
		payload.Diagnostics = append(payload.Diagnostics, Diagnostic{
			RuleID:   d.RuleID,
			Severity: d.Severity.String(),
			Message:  d.Message,
			File:     d.File,
			Line:     d.Position.Line,
			Column:   d.Position.Column,
		})
	}

	if graph == nil {
		return payload
	}

	for urn, resource := range graph.Resources() {
		node := Node{
			URN:         urn,
			ID:          resource.ID(),
			Type:        resource.Type(),
			DisplayName: displayName(resource),
		}

		if loc, ok := locations[urn]; ok {
			node.File = loc.File
			node.Line = loc.Line
			node.Column = loc.Column
		}

		payload.Nodes = append(payload.Nodes, node)

		for _, dependency := range graph.GetDependencies(urn) {
			payload.Edges = append(payload.Edges, Edge{From: urn, To: dependency})
		}
	}

	sort.Slice(payload.Nodes, func(i, j int) bool {
		return payload.Nodes[i].URN < payload.Nodes[j].URN
	})
	sort.Slice(payload.Edges, func(i, j int) bool {
		if payload.Edges[i].From != payload.Edges[j].From {
			return payload.Edges[i].From < payload.Edges[j].From
		}
		return payload.Edges[i].To < payload.Edges[j].To
	})

	// DetectCycles signals a cycle by returning BOTH the path and a non-nil
	// error describing it, so the path is what we want on the error branch.
	// Reported rather than propagated: a cycle makes apply impossible, but the
	// graph containing it is exactly what a reader needs in order to fix it.
	cycle, _ := graph.DetectCycles()
	payload.Cycle = cycle

	return payload
}

// displayName prefers a human-facing name from the resource's data, falling
// back to the id, which every resource has.
func displayName(r *resources.Resource) string {
	for _, key := range []string{"name", "display_name", "displayName"} {
		if v, ok := r.Data()[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return r.ID()
}
