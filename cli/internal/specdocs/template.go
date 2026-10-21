package specdocs

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"

	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

// connectionModeKey is the config block mapping source tokens to modes.
const connectionModeKey = "connection_mode"

// specTemplate writes a destination spec holding what every spec of this type
// needs for doc's source: each top-level required key, the keys the chosen
// connection mode requires, and the connection_mode entry. A secret holds its
// slot and any other key null, so validate names each key still to fill. The
// mode is cloud when the source token allows it, or the only mode there is;
// otherwise the entry is left out for the caller to choose.
func specTemplate(doc DestinationDoc, id string) (string, error) {
	var (
		fields = make(map[string]Field, len(doc.Fields))
		config = mappingNode()
		keys   []string
	)
	for _, f := range doc.Fields {
		fields[f.Key] = f
		if f.Required && f.Key != connectionModeKey {
			keys = append(keys, f.Key)
		}
	}

	modes := doc.ConnectionModes[doc.Source.Token]
	mode := chooseMode(modes.Modes)
	for _, key := range modes.RequiredKeys[mode] {
		if key != connectionModeKey && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}

	for _, key := range keys {
		appendPair(config, key, templateValue(fields[key]))
	}
	if mode != "" {
		entry := mappingNode()
		appendPair(entry, doc.Source.Token, stringNode(mode))
		appendPair(config, connectionModeKey, entry)
	}

	metadata := mappingNode()
	appendPair(metadata, "name", stringNode(id))

	spec := mappingNode()
	appendPair(spec, "id", stringNode(id))
	appendPair(spec, "display_name", stringNode(id))
	appendPair(spec, "type", stringNode(doc.Type))
	appendPair(spec, "definition_version", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(doc.DefinitionVersion, 10)})
	appendPair(spec, "enabled", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	appendPair(spec, "config", config)

	root := mappingNode()
	appendPair(root, "version", stringNode(specs.SpecVersionV1))
	appendPair(root, "kind", stringNode(destination.DestinationSpecKind))
	appendPair(root, "metadata", metadata)
	appendPair(root, "spec", spec)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return "", fmt.Errorf("encoding spec template: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("encoding spec template: %w", err)
	}
	return buf.String(), nil
}

func chooseMode(modes []string) string {
	if slices.Contains(modes, "cloud") {
		return "cloud"
	}
	if len(modes) == 1 {
		return modes[0]
	}
	return ""
}

func templateValue(f Field) *yaml.Node {
	if f.Var != "" {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Style: yaml.SingleQuotedStyle, Value: secret.Reference(f.Var)}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}
}

func mappingNode() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func appendPair(mapping *yaml.Node, key string, value *yaml.Node) {
	mapping.Content = append(mapping.Content, stringNode(key), value)
}
