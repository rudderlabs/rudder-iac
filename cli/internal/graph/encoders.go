package graph

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func renderJSON(w io.Writer, doc Document) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("encoding json graph: %w", err)
	}
	return nil
}

// renderDOT emits a graphviz digraph. Nodes are shaped/coloured by resource
// type so the rendered image conveys the type at a glance.
func renderDOT(w io.Writer, doc Document) error {
	var b strings.Builder
	b.WriteString("digraph resources {\n")
	b.WriteString("  rankdir=LR;\n")

	for _, n := range doc.Nodes {
		shape, color := dotStyle(n.Type)
		fmt.Fprintf(&b, "  %q [label=%q, shape=%s, style=filled, fillcolor=%q];\n",
			n.URN, n.URN, shape, color)
	}

	for _, e := range doc.Edges {
		fmt.Fprintf(&b, "  %q -> %q;\n", e.From, e.To)
	}

	for _, c := range doc.Cycles {
		fmt.Fprintf(&b, "  // cycle: %s\n", c)
	}

	b.WriteString("}\n")

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing dot graph: %w", err)
	}
	return nil
}

// renderMermaid emits a mermaid `graph TD` diagram. Node ids are sanitised to
// mermaid-safe identifiers while the URN is kept as the visible label.
func renderMermaid(w io.Writer, doc Document) error {
	var b strings.Builder
	b.WriteString("graph TD\n")

	for _, n := range doc.Nodes {
		fmt.Fprintf(&b, "  %s[%q]\n", mermaidID(n.URN), n.URN)
	}

	for _, e := range doc.Edges {
		fmt.Fprintf(&b, "  %s --> %s\n", mermaidID(e.From), mermaidID(e.To))
	}

	for _, c := range doc.Cycles {
		fmt.Fprintf(&b, "  %%%% cycle: %s\n", c)
	}

	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("writing mermaid graph: %w", err)
	}
	return nil
}

// dotStyle maps a resource type to a graphviz shape and fill colour. Unknown
// types fall back to a neutral default rather than being rejected, so new
// resource types render without a code change.
func dotStyle(resourceType string) (shape, color string) {
	switch resourceType {
	case "event":
		return "box", "#cfe2ff"
	case "property":
		return "ellipse", "#d1e7dd"
	case "tracking-plan":
		return "component", "#fff3cd"
	case "custom-type":
		return "ellipse", "#e2e3e5"
	default:
		return "box", "#f8f9fa"
	}
}

// mermaidID converts a URN into a mermaid-safe node identifier. Mermaid node
// ids may not contain ":" or "-", so they are replaced with "_".
func mermaidID(urn string) string {
	r := strings.NewReplacer(":", "_", "-", "_", ".", "_", "/", "_")
	return "n_" + r.Replace(urn)
}
