// Package format provides a deterministic, idempotent canonical formatter for
// spec YAML, akin to gofmt/terraform fmt. It normalizes whitespace, indentation
// and quoting only — key order and comments are preserved (author intent), so a
// formatted spec parses identically to the original.
package format

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// canonicalIndent is the number of spaces per nesting level in formatted output.
const canonicalIndent = 2

// Source formats a single YAML document into canonical form. It re-encodes the
// parsed node tree with fixed indentation while preserving comments and key
// order. Empty or whitespace-only input formats to empty output.
//
// The transform is idempotent: Source(Source(x)) == Source(x).
func Source(data []byte) ([]byte, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// An empty or comment-free blank document yields a zero-value node with no
	// content; nothing to re-encode.
	if node.Kind == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(canonicalIndent)

	if err := encoder.Encode(&node); err != nil {
		return nil, fmt.Errorf("encoding YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}

	return buf.Bytes(), nil
}
