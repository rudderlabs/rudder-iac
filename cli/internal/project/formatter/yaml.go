package formatter

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	defaultIndent = 2
)

// YAMLFormatter formats data into YAML with custom string quoting behavior.
// String values are always double-quoted while keys remain unquoted.
type YAMLFormatter struct{}

// Format converts data to YAML format with 2-space indentation and quoted string values.
func (f YAMLFormatter) Format(data any) ([]byte, error) {
	var node yaml.Node

	if err := node.Encode(data); err != nil {
		return nil, fmt.Errorf("encoding data to YAML node: %w", err)
	}
	forceStringQuotes(&node)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(defaultIndent)

	if err := encoder.Encode(&node); err != nil {
		return nil, fmt.Errorf("encoding YAML node to bytes: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// Extension returns "yaml" as the file extension.
func (f YAMLFormatter) Extension() []string {
	return []string{"yaml", "yml"}
}

// forceStringQuotes walks the YAML node tree and forces double quotes on all string values.
// Keys in mappings are left unquoted to maintain readability.
func forceStringQuotes(node *yaml.Node) {
	switch node.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		// Process all children
		for _, child := range node.Content {
			forceStringQuotes(child)
		}
	case yaml.MappingNode:
		// For mapping nodes, skip keys (even indices) and only process values (odd indices)
		for i, child := range node.Content {
			if i%2 == 1 { // Only process values (odd indices)
				forceStringQuotes(child)
			} else {
				// Still need to recurse into keys in case they contain nested structures
				// but don't quote the key itself if it's a string
				if child.Kind != yaml.ScalarNode {
					forceStringQuotes(child)
				}
			}
		}
	case yaml.ScalarNode:
		// Only quote if it's a string
		if node.Tag == "!!str" {
			node.Style = yaml.DoubleQuotedStyle
		}
	}
}
