package docs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Serialize writes the resolved rules catalog as rules.yaml to outDir. YAML is
// the canonical artifact consumed by the Hugo translation layer downstream. The
// file is overwritten on each run.
//
// JSON output (for LLM tooling) was intentionally dropped: there is no consumer
// today, and it can be re-derived from the canonical YAML as a quick incremental
// lift when one appears.
func Serialize(doc DocumentedRules, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir %s: %w", outDir, err)
	}

	var node yaml.Node
	if err := node.Encode(doc); err != nil {
		return fmt.Errorf("encoding yaml node: %w", err)
	}
	preferDoubleQuotes(&node)

	// Emit 2-space indentation to match the rudder-hugo catalog (the file this
	// converges to), rather than yaml.v3's 4-space default.
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return fmt.Errorf("marshaling yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("closing yaml encoder: %w", err)
	}

	if err := writeFile(filepath.Join(outDir, "rules.yaml"), buf.Bytes()); err != nil {
		return err
	}

	return nil
}

// preferDoubleQuotes rewrites single-quoted scalars to double-quoted. yaml.v3
// defaults a string containing a single quote to single-quoted style with the
// quote doubled (e.g. ”'name” is required'); the hand-authored reference
// catalog uses the more readable double-quoted form ("'name' is required").
// Matching it keeps the generated artifact byte-stable against that reference
// and easier to read. Only the quoting style changes — never the value.
func preferDoubleQuotes(n *yaml.Node) {
	if n.Kind == yaml.ScalarNode && n.Style == yaml.SingleQuotedStyle {
		n.Style = yaml.DoubleQuotedStyle
	}
	for _, child := range n.Content {
		preferDoubleQuotes(child)
	}
}

func writeFile(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
