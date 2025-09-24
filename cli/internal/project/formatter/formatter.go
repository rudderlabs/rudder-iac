package formatter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rudderlabs/rudder-iac/cli/internal/importremote"
	"gopkg.in/yaml.v3"
)

type YAMLFormatter struct {
}

func (f *YAMLFormatter) Format(baseDir string, entity importremote.FormattableEntity) error {
	specYAML, err := encodeYaml(entity.Content)
	if err != nil {
		return err
	}

	dir, _ := filepath.Split(fmt.Sprintf("%s/%s", baseDir, entity.RelativePath))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(fmt.Sprintf("%s/%s", baseDir, entity.RelativePath), specYAML, 0644); err != nil {
		return err
	}
	return nil
}

func encodeYaml(spec any) ([]byte, error) {
	var node yaml.Node
	err := node.Encode(spec)
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

// Function to force quote only string values (not keys)
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
