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

	return encodeNode(&node)
}

// Extension returns "yaml" as the file extension.
func (f YAMLFormatter) Extension() []string {
	return []string{"yaml", "yml"}
}

// YAMLOrderedFormatter behaves like YAMLFormatter but reorders the migrated
// output so that every key present in the Original node keeps its original
// position. Keys added by the caller (not in Original) are appended at the
// end of their parent mapping. For sequences of mappings, items are matched
// by an identity key (id, then name) so list order survives.
//
// Intended for the migrate write path, where users want git diffs that
// reflect only semantic migrations — not alphabetization.
type YAMLOrderedFormatter struct {
	Original *yaml.Node
}

func (f YAMLOrderedFormatter) Format(data any) ([]byte, error) {
	var node yaml.Node

	if err := node.Encode(data); err != nil {
		return nil, fmt.Errorf("encoding data to YAML node: %w", err)
	}
	if f.Original != nil {
		reorderToMatch(&node, f.Original)
	}
	forceStringQuotes(&node)

	return encodeNode(&node)
}

func (f YAMLOrderedFormatter) Extension() []string {
	return []string{"yaml", "yml"}
}

func encodeNode(node *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(defaultIndent)

	if err := encoder.Encode(node); err != nil {
		return nil, fmt.Errorf("encoding YAML node to bytes: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}

	return buf.Bytes(), nil
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

// reorderToMatch rewrites newNode.Content in place so its order tracks origNode.
//
// Algorithm (same shape for both mappings and sequences):
//
//  1. Index the children of newNode by a matching key.
//  2. Walk origNode's children in order; for each one whose key exists in
//     newNode, claim the corresponding new child into the output and recurse
//     into its subtree so nested orderings are also restored.
//  3. Append any new-only children (not claimed in step 2) at the end of the
//     output, preserving their original relative order.
//
// For mappings the matching key is the key scalar's value; for sequences of
// mappings the matching key is an identity field (id, then name) probed on
// each item. Sequences without a usable identity key fall back to positional
// matching (recurse on child pairs in index order).
//
// Nodes that exist only on the new side are never dropped — they always end
// up after the origin-ordered prefix.
func reorderToMatch(newNode, origNode *yaml.Node) {
	if newNode == nil || origNode == nil {
		return
	}

	newRoot := unwrapDocument(newNode)
	origRoot := unwrapDocument(origNode)
	if newRoot == nil || origRoot == nil || newRoot.Kind != origRoot.Kind {
		return
	}

	switch newRoot.Kind {
	case yaml.MappingNode:
		reorderMapping(newRoot, origRoot)
	case yaml.SequenceNode:
		reorderSequence(newRoot, origRoot)
	}
}

// unwrapDocument returns the top-level node inside a DocumentNode wrapper,
// or the node itself. yaml.Unmarshal produces a DocumentNode but node.Encode
// does not, so the two inputs to reorderToMatch may differ at the root.
func unwrapDocument(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

// reorderMapping aligns newNode's key order with origNode's.
//
// yaml.v3 stores mapping content as a flat slice [k0, v0, k1, v1, ...]; each
// logical pair occupies two consecutive slice positions.
func reorderMapping(newNode, origNode *yaml.Node) {
	newPairs := newNode.Content
	origPairs := origNode.Content

	// 1. Index: mapping key (string) -> position of its key node in newPairs.
	//    Only scalar keys are indexable; non-scalar keys are skipped here and
	//    end up appended at step 3 via the !claimed path.
	newIdx := make(map[string]int, len(newPairs)/2)
	for i := 0; i < len(newPairs); i += 2 {
		if newPairs[i].Kind == yaml.ScalarNode {
			newIdx[newPairs[i].Value] = i
		}
	}

	reordered := make([]*yaml.Node, 0, len(newPairs))
	claimed := make([]bool, len(newPairs))

	// 2. Walk origPairs in order; claim the matching new pair and recurse
	//    into its value so nested orderings are restored too.
	for i := 0; i < len(origPairs); i += 2 {
		origKey := origPairs[i]
		if origKey.Kind != yaml.ScalarNode {
			continue
		}
		ni, ok := newIdx[origKey.Value]
		if !ok {
			// Key exists in original but was removed by the migration; drop it.
			continue
		}
		reorderToMatch(newPairs[ni+1], origPairs[i+1])
		reordered = append(reordered, newPairs[ni], newPairs[ni+1])
		claimed[ni] = true
	}

	// 3. Append any new-only pairs (keys not in origPairs) at the end,
	//    preserving their relative order.
	for i := 0; i < len(newPairs); i += 2 {
		if claimed[i] {
			continue
		}
		reordered = append(reordered, newPairs[i], newPairs[i+1])
	}

	newNode.Content = reordered
}

// identityKeys lists the candidate fields used to match items across
// sequences of mappings, in priority order. Each has schema-enforced
// uniqueness within its list in rudder-iac specs:
//   - `id`: properties, events, categories, custom types, tracking plan
//     rules, datagraph models/relationships.
//   - `$ref`: v0 tracking plan rule / variant case property references
//     (validated by "duplicate property reference" semantic rules).
//   - `property`: v1 equivalent of `$ref`.
//
// Lists that don't carry one of these fall back to positional matching.
var identityKeys = []string{"id", "$ref", "property"}

// reorderSequence aligns list-item order for sequences of mappings using an
// identity key. Sequences of scalars or mixed kinds fall back to positional
// recursion.
func reorderSequence(newNode, origNode *yaml.Node) {
	newItems := newNode.Content
	origItems := origNode.Content

	identity := pickIdentityKey(origItems, newItems)
	if identity == "" {
		// Positional fallback: recurse on index-aligned pairs, leave lengths
		// and ordering as produced by the migrator. No items are dropped —
		// newNode.Content is left intact, so if the migration reshuffled the
		// list the order may be wrong but every element is still present.
		n := len(origItems)
		if len(newItems) < n {
			n = len(newItems)
		}
		for i := 0; i < n; i++ {
			reorderToMatch(newItems[i], origItems[i])
		}
		return
	}

	// 1. Index: identity value -> position of item in newItems.
	newByID := make(map[string]int, len(newItems))
	for i, item := range newItems {
		if id, ok := mappingScalar(item, identity); ok {
			newByID[id] = i
		}
	}

	reordered := make([]*yaml.Node, 0, len(newItems))
	claimed := make([]bool, len(newItems))

	// 2. Walk origItems in order; claim the matching new item and recurse.
	for _, origItem := range origItems {
		id, ok := mappingScalar(origItem, identity)
		if !ok {
			continue
		}
		ni, ok := newByID[id]
		if !ok {
			// Key exists in original but was removed by the migration; drop it.
			continue
		}
		reorderToMatch(newItems[ni], origItem)
		reordered = append(reordered, newItems[ni])
		claimed[ni] = true
	}

	// 3. Append any new-only items at the end, preserving their relative order.
	for i, item := range newItems {
		if claimed[i] {
			continue
		}
		reordered = append(reordered, item)
	}

	newNode.Content = reordered
}

// pickIdentityKey returns the first identityKeys entry that appears on at
// least one item in both sequences. Returns "" when both sides aren't
// all-mappings or no candidate matches, signalling positional fallback.
func pickIdentityKey(origItems, newItems []*yaml.Node) string {
	if !allMappings(origItems) || !allMappings(newItems) {
		return ""
	}
	for _, key := range identityKeys {
		if sequenceHasKey(origItems, key) && sequenceHasKey(newItems, key) {
			return key
		}
	}
	return ""
}

func allMappings(items []*yaml.Node) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if item.Kind != yaml.MappingNode {
			return false
		}
	}
	return true
}

func sequenceHasKey(items []*yaml.Node, key string) bool {
	for _, item := range items {
		if _, ok := mappingScalar(item, key); ok {
			return true
		}
	}
	return false
}

// mappingScalar returns the scalar value stored at `key` inside a mapping
// node, or "",false if the node is not a mapping, the key isn't present, or
// its value isn't a scalar.
func mappingScalar(n *yaml.Node, key string) (string, bool) {
	if n == nil || n.Kind != yaml.MappingNode {
		return "", false
	}
	for i := 0; i < len(n.Content); i += 2 {
		k := n.Content[i]
		if k.Kind != yaml.ScalarNode || k.Value != key {
			continue
		}
		if i+1 >= len(n.Content) {
			return "", false
		}
		v := n.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			return "", false
		}
		return v.Value, true
	}
	return "", false
}
