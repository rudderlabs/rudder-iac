package location

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Position represents a position in a YAML file
type Position struct {
	Line    int
	Column  int
	Content string // The actual line content from the source file
}

// PathIndex maintains a mapping from YAML paths to positions
type PathIndex struct {
	pathLookup map[string]*Position
	lines      []string // Source file lines for content extraction
}

// YAMLDataIndex builds a PathIndex from YAML data
func YAMLDataIndex(data []byte) (*PathIndex, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("parsing yaml: %w", err)
	}

	// Split source into lines for content extraction
	lines := strings.Split(string(data), "\n")

	index := &PathIndex{
		pathLookup: make(map[string]*Position),
		lines:      lines,
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
		Line:    node.Line,
		Column:  node.Column,
		Content: idx.getLineContent(node.Line),
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
				Line:    keyNode.Line,
				Column:  keyNode.Column,
				Content: idx.getLineContent(keyNode.Line),
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

// getLineContent returns the content of a specific line (1-indexed)
func (idx *PathIndex) getLineContent(lineNum int) string {
	if lineNum <= 0 || lineNum > len(idx.lines) {
		return ""
	}
	return strings.TrimRight(idx.lines[lineNum-1], "\r\n")
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
