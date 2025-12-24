package location

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Position represents a position in a YAML file
type Position struct {
	Line   int
	Column int
}

// PathIndex maintains a mapping from YAML paths to positions
type PathIndex struct {
	pathLookup map[string]*Position
}

// YAMLDataIndex builds a PathIndex from YAML data
func YAMLDataIndex(data []byte) (*PathIndex, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	index := &PathIndex{
		pathLookup: make(map[string]*Position),
	}

	if len(node.Content) > 0 {
		index.walk(node.Content[0], "")
	}

	return index, nil
}

// walk recursively visits YAML nodes to build the path index
func (idx *PathIndex) walk(node *yaml.Node, currentPath string) {
	// Store position for the current path
	idx.pathLookup[currentPath] = &Position{
		Line:   node.Line,
		Column: node.Column,
	}

	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			idx.walk(child, currentPath)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			path := currentPath + "/" + keyNode.Value
			// Also index the key itself
			idx.pathLookup[path] = &Position{
				Line:   keyNode.Line,
				Column: keyNode.Column,
			}
			idx.walk(valueNode, path)
		}
	case yaml.SequenceNode:
		for i, child := range node.Content {
			path := fmt.Sprintf("%s/%d", currentPath, i)
			idx.walk(child, path)
		}
	}
}

// Lookup returns the position information at the specified path
func (idx *PathIndex) Lookup(path string) *Position {
	if idx == nil {
		return nil
	}
	// Normalise path (ensure it starts with /)
	if !strings.HasPrefix(path, "/") && path != "" {
		path = "/" + path
	}
	return idx.pathLookup[path]
}
