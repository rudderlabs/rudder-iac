package pathindex

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type ErrPathNotFound struct {
	Path string
}

func (e *ErrPathNotFound) Error() string {
	return fmt.Sprintf("path not found: %s", e.Path)
}

// PathIndexer defines the interface for position lookup
type PathIndexer interface {
	PositionLookup(path string) (*Position, error)
}

// PathIndex stores mapping from JSON Pointer paths to file positions
type PathIndex struct {
	positions map[string]Position
}

// Position represents a location in a YAML file for error reporting
type Position struct {
	Line     int    // 1-indexed line number
	Column   int    // 1-indexed column (caret position)
	LineText string // Full text representation for error display
}

// NewPathIndex creates a new PathIndex by parsing YAML content and building the position map
func NewPathIndexer(content []byte) (PathIndexer, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("parsing YAML content: %w", err)
	}

	pi := &PathIndex{
		positions: make(map[string]Position),
	}

	// Walk the YAML tree and build the index
	pi.walkNode(&node, "", nil)
	return pi, nil
}

// PositionLookup returns the position for a JSON Pointer path
// Returns error if path not found
func (idx *PathIndex) PositionLookup(path string) (*Position, error) {
	pos, ok := idx.positions[path]
	if !ok {
		return nil, &ErrPathNotFound{Path: path}
	}
	return &pos, nil
}

// walkNode recursively walks the YAML node tree and records positions
func (pi *PathIndex) walkNode(node *yaml.Node, path string, key *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Document node is just a wrapper,
		// walk into the actual content
		if len(node.Content) > 0 {
			pi.walkNode(node.Content[0], path, nil)
		}

	case yaml.MappingNode:
		// Mapping nodes contain alternating key-value pairs
		// Record position for the mapping itself if it has a path
		pi.recordPosition(path, node, key)

		// Process key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}

			var (
				keyNode   = node.Content[i]
				valueNode = node.Content[i+1]
			)

			pi.walkNode(
				valueNode,
				fmt.Sprintf("%s/%s", path, keyNode.Value),
				keyNode,
			)
		}

	case yaml.SequenceNode:
		// Record position for the array itself
		pi.recordPosition(path, node, key)

		// Process array elements
		for idx, item := range node.Content {
			arrayPath := fmt.Sprintf("%s/%d", path, idx)
			pi.walkNode(item, arrayPath, nil)
		}

	case yaml.ScalarNode:
		// Terminal node - record its position
		pi.recordPosition(path, node, key)

		// 	case yaml.AliasNode:
		// 		// For alias nodes, use the aliased node's position
		// 		if node.Alias != nil && path != "" {
		// 			pi.walkNode(node.Alias, path, key)
		// 		}
	}
}

// recordPosition records a position for a given path
func (pi *PathIndex) recordPosition(
	path string,
	node *yaml.Node,
	key *yaml.Node,
) {
	if path == "" {
		return
	}

	// Extract line text representation
	lineText := extractLineText(node, key)

	var (
		line   = node.Line
		column = node.Column
	)

	// If the key is not nil, we are in a situation
	// where the node is mapping node or a scalar node.
	// In case it's one of the sequence values, we use the node's line and column.
	if key != nil {
		line = key.Line
		column = key.Column
	}

	pos := Position{
		Line:     line,
		Column:   column,
		LineText: lineText,
	}

	pi.positions[path] = pos
}

// extractLineText constructs a readable text representation from the node
func extractLineText(node *yaml.Node, key *yaml.Node) string {
	switch node.Kind {
	case yaml.ScalarNode:
		if key != nil {
			return fmt.Sprintf("%s: %s", key.Value, node.Value)
		}
		return node.Value

	case yaml.MappingNode:
		// If the mapping node has less than 2 children,
		// it is an empty object and then we only return the key if it's present
		if len(node.Content) < 2 {
			if key != nil {
				return fmt.Sprintf("%s: {}", key.Value)
			}
			return "{}"
		}

		// If the mapping node has more than 2 children,
		// it is a non-empty object and then we return the key with the ellipsis
		if key != nil {
			return fmt.Sprintf("%s: {...}", key.Value)
		}
		return "{...}"

	case yaml.SequenceNode:
		// Show that this is an array
		if key != nil {
			return fmt.Sprintf("%s: [...]", key.Value)
		}
		return "[...]"

	default:
		if key != nil {
			return key.Value
		}
		return ""
	}
}
