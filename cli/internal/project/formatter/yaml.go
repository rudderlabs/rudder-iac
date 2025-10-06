package formatter

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// YAMLFormatter formats data into YAML with custom string quoting behavior.
// String values are always double-quoted while keys remain unquoted.
type YAMLFormatter struct{}

func NewYAMLFormatter() *YAMLFormatter {
	return &YAMLFormatter{}
}

// Format converts data to YAML format with 2-space indentation and quoted string values.
func (f YAMLFormatter) Format(data any) ([]byte, error) {
	var node yaml.Node
	err := node.Encode(data)
	if err != nil {
		return nil, err
	}
	forceStringQuotes(&node)

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err = encoder.Encode(&node)
	if err != nil {
		return nil, err
	}
	err = encoder.Close()
	if err != nil {
		return nil, err
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
