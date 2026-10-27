package pathindex

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var ErrPathNotFound = errors.New("path not found")

// PathIndexer defines the interface for position lookup
type PathIndexer interface {
	// PositionLookup returns the position for an exact JSON Pointer path.
	// Returns ErrPathNotFound if the path doesn't exist.
	PositionLookup(path string) (*Position, error)

	// NearestPosition returns the position for the given path or its nearest ancestor.
	// It tries the exact path first, then walks up the hierarchy until a match is found.
	// For example: /a/b/c -> /a/b -> /a -> /
	// Always succeeds since "/" (root) is guaranteed to exist at line 1, column 1.
	NearestPosition(path string) *Position

	// MaskedPosition is NearestPosition with the value the line text shows
	// replaced by "***". The key stays, and a line that shows a
	// container ("key: {...}", "[...]") is unchanged, since it holds no value.
	MaskedPosition(path string) *Position
}

// PathIndex stores mapping from JSON Pointer paths to file positions
type PathIndex struct {
	positions map[string]Position
	// maskedLineTexts holds each path's line text as MaskedPosition prints it.
	// It is built with the index, because only the YAML node tells a scalar
	// list item's bare value apart from a "key: value" line.
	maskedLineTexts map[string]string

	// expanding holds the anchored nodes whose aliases are being walked, so an
	// alias inside its own anchor ends the walk there. aliasBudget caps how
	// many aliases are walked, since each repeats its anchor's whole subtree.
	expanding   map[*yaml.Node]bool
	aliasBudget int
	// throughAlias counts the aliases the walk is below. A node reached
	// through an alias is written for another path, where its value may be a
	// secret that the path being indexed does not mark as one, so its line
	// text shows the value masked.
	throughAlias int
}

// maxAliasWalks caps the aliases one index walks. A path below an alias past
// the cap falls back to its nearest indexed ancestor, as every alias did
// before aliases were walked.
const maxAliasWalks = 1000

var StartingPosition = Position{
	Line:     1,
	Column:   1,
	LineText: "",
}

// Position represents a location in a YAML file for error reporting
type Position struct {
	Line     int    // 1-indexed line number
	Column   int    // 1-indexed column (caret position)
	LineText string // Full text representation for error display
}

// maskedValue stands in for a value that must not be printed.
const maskedValue = "***"

// NewPathIndex creates a new PathIndex by parsing YAML content and building the position map
func NewPathIndexer(content []byte) (PathIndexer, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("parsing YAML content: %w", err)
	}

	pi := &PathIndex{
		positions:       make(map[string]Position),
		maskedLineTexts: make(map[string]string),
		expanding:       make(map[*yaml.Node]bool),
		aliasBudget:     maxAliasWalks,
	}

	// Record root position "/" at line 1, column 1
	// This serves as the ultimate fallback for NearestPosition
	pi.positions["/"] = StartingPosition
	pi.maskedLineTexts["/"] = StartingPosition.LineText

	// Walk the YAML tree and build the index
	pi.walkNode(&node, "", nil)
	return pi, nil
}

// PositionLookup returns the position for a JSON Pointer path
// Returns error if path not found
func (idx *PathIndex) PositionLookup(path string) (*Position, error) {
	pos, ok := idx.positions[path]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPathNotFound, path)
	}
	return &pos, nil
}

// NearestPosition returns the position for the given path or its nearest ancestor.
// It tries the exact path first, then walks up the hierarchy until a match is found.
// For example: /a/b/c -> /a/b -> /a -> /
// Always succeeds since "/" (root) is guaranteed to exist at line 1, column 1.
func (idx *PathIndex) NearestPosition(path string) *Position {
	pos := idx.positions[idx.nearestPath(path)]
	return &pos
}

// MaskedPosition is NearestPosition with the value the line text shows
// replaced by maskedValue ("***").
func (idx *PathIndex) MaskedPosition(path string) *Position {
	nearest := idx.nearestPath(path)
	pos := idx.positions[nearest]
	pos.LineText = idx.maskedLineTexts[nearest]
	return &pos
}

// nearestPath returns path when it is indexed, or else its nearest indexed
// ancestor, falling back to the root "/".
func (idx *PathIndex) nearestPath(path string) string {
	// Try exact match first
	if _, ok := idx.positions[path]; ok {
		return path
	}

	// Walk up the path hierarchy
	currentPath := path
	for currentPath != "" && currentPath != "/" {
		// Find the last slash and truncate
		lastSlash := strings.LastIndex(currentPath, "/")
		if lastSlash <= 0 {
			// Reached root or no more slashes
			break
		}

		currentPath = currentPath[:lastSlash]
		if _, ok := idx.positions[currentPath]; ok {
			return currentPath
		}
	}

	// Fall back to root position (always exists)
	return "/"
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

	case yaml.MappingNode, yaml.SequenceNode:
		// Record position for the mapping or array itself, then its children
		pi.recordPosition(path, node, key)
		pi.walkChildren(node, path)

	case yaml.ScalarNode:
		// Terminal node - record its position
		pi.recordPosition(path, node, key)

	case yaml.AliasNode:
		// An alias is indexed at its own place, and its line shows the alias
		// as written, such as "to: *bad", never the anchor's value, which can
		// be a secret under a key the masking does not treat as one. The paths
		// below an alias are indexed at the anchor's lines. Without this a
		// diagnostic on an aliased value printed an ancestor's line, such as a
		// list item's first property, which can be a secret the masking, deciding
		// on the aliased path, did not mask.
		pi.recordPosition(path, node, key)
		target := node.Alias
		if target == nil || pi.expanding[target] || pi.aliasBudget <= 0 {
			return
		}
		pi.aliasBudget--
		pi.expanding[target] = true
		pi.throughAlias++
		defer func() {
			delete(pi.expanding, target)
			pi.throughAlias--
		}()
		pi.walkChildren(target, path)
	}
}

// walkChildren walks the values of a mapping or the items of a sequence,
// below path.
func (pi *PathIndex) walkChildren(node *yaml.Node, path string) {
	switch node.Kind {
	case yaml.MappingNode:
		pi.walkMapping(node, path, map[string]bool{})

	case yaml.SequenceNode:
		for idx, item := range node.Content {
			pi.walkNode(item, fmt.Sprintf("%s/%d", path, idx), nil)
		}
	}
}

// walkMapping walks the values of a mapping below path, skipping the keys in
// indexed. A merge key (<<) adds the keys of the mappings it names at their
// own lines, below path, as the decoded spec has them: a key the mapping
// writes wins over a merged one, and among merged mappings the first wins, as
// the YAML decoder merges them. Without this a diagnostic on a merged key
// printed the mapping's line, which for a list item shows its first
// property, such as a secret.
func (pi *PathIndex) walkMapping(node *yaml.Node, path string, indexed map[string]bool) {
	var merges []*yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode, valueNode := node.Content[i], node.Content[i+1]
		if keyNode.ShortTag() == "!!merge" {
			merges = append(merges, valueNode)
			continue
		}
		if indexed[keyNode.Value] {
			continue
		}
		indexed[keyNode.Value] = true
		pi.walkNode(valueNode, fmt.Sprintf("%s/%s", path, keyNode.Value), keyNode)
	}

	for _, merge := range merges {
		sources := []*yaml.Node{merge}
		if merge.Kind == yaml.SequenceNode {
			sources = merge.Content
		}
		for _, source := range sources {
			pi.walkMerged(source, path, indexed)
		}
	}
}

// walkMerged walks one mapping a merge key names, directly or through an
// alias, below path. Past the alias cap, or inside the anchor it names, the
// mapping's keys are indexed at the alias, whose line shows only the alias,
// so a diagnostic on one never falls back to a line that shows a value.
func (pi *PathIndex) walkMerged(source *yaml.Node, path string, indexed map[string]bool) {
	if source.Kind != yaml.AliasNode {
		if source.Kind == yaml.MappingNode {
			pi.walkMapping(source, path, indexed)
		}
		return
	}
	target := source.Alias
	if target == nil || target.Kind != yaml.MappingNode {
		return
	}
	if pi.expanding[target] || pi.aliasBudget <= 0 {
		for i := 0; i+1 < len(target.Content); i += 2 {
			if name := target.Content[i].Value; !indexed[name] {
				indexed[name] = true
				pi.recordPosition(fmt.Sprintf("%s/%s", path, name), source, nil)
			}
		}
		return
	}
	pi.aliasBudget--
	pi.expanding[target] = true
	pi.throughAlias++
	defer func() {
		delete(pi.expanding, target)
		pi.throughAlias--
	}()
	pi.walkMapping(target, path, indexed)
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
	// In case it's one of the sequence values, we use the node's line and column. For example:
	// abc:
	// - 1
	// - 2
	// For abc sequence node, when we are processing the first item aka `1`,
	// we will have only the value node, so we will skip the below condition
	// as will have key as nil.
	if key != nil {
		line = key.Line
		column = key.Column
	}

	masked := maskedLineText(node, key, lineText)
	if pi.throughAlias > 0 {
		lineText = masked
	}

	pos := Position{
		Line:     line,
		Column:   column,
		LineText: lineText,
	}

	pi.positions[path] = pos
	pi.maskedLineTexts[path] = masked
}

// maskedLineText returns lineText, built by extractLineText, with the value it
// shows replaced by maskedValue. A scalar shows its value after its key, or
// alone as a list item. A list item that is a mapping shows its first
// property's value. Other lines show no value, and an empty value or a null is
// left as it is.
func maskedLineText(node *yaml.Node, key *yaml.Node, lineText string) string {
	switch {
	case showsValue(node):
		if key != nil {
			return fmt.Sprintf("%s: %s", key.Value, maskedValue)
		}
		return maskedValue

	case node.Kind == yaml.MappingNode && key == nil && len(node.Content) >= 2 && showsValue(node.Content[1]):
		return fmt.Sprintf("- %s: %s", node.Content[0].Value, maskedValue)
	}
	return lineText
}

// nullSpellings are the plain scalars YAML reads as null.
var nullSpellings = []string{"~", "null", "Null", "NULL"}

// showsValue reports whether node is a scalar whose line text shows a value
// that could be a secret. An empty value does not, and neither does a null
// written as ~ or null: masking it would hide the null that a check such as
// "must not be null" reports. A value tagged !!null by hand is still masked.
func showsValue(node *yaml.Node) bool {
	if node.Kind != yaml.ScalarNode || node.Value == "" {
		return false
	}
	return node.ShortTag() != "!!null" || !slices.Contains(nullSpellings, node.Value)
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

		// Array element (no key) - show first property to help identify the element
		firstKey := node.Content[0]
		firstVal := node.Content[1]
		return fmt.Sprintf("- %s: %s", firstKey.Value, firstVal.Value)

	case yaml.SequenceNode:
		// Show that this is an array
		if key != nil {
			return fmt.Sprintf("%s: [...]", key.Value)
		}
		return "[...]"

	case yaml.AliasNode:
		// The alias as written, not the value of its anchor.
		if key != nil {
			return fmt.Sprintf("%s: *%s", key.Value, node.Value)
		}
		return "*" + node.Value

	default:
		if key != nil {
			return key.Value
		}
		return ""
	}
}
