package specdocs

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

// connectionModeKey is the config block mapping source tokens to modes.
const connectionModeKey = "connection_mode"

const (
	// templateNote heads every spec template. Validate checks the conditions
	// of requiredIf and requiredUnless against the values the caller fills
	// in, which the template cannot know.
	templateNote = "Replace each null with a value. A value you fill in can meet another key's\n" +
		"requiredIf or requiredUnless condition, which makes that key required too."
	// customValidatorNote follows templateNote when the type has custom
	// validators, whose conditions only their Go code states.
	customValidatorNote = "A custom validator (checkedBy) can require more keys under conditions the\n" +
		"docs cannot state; validate names them."
)

// specTemplate writes a destination spec holding what every spec of this type
// needs for doc's source: each top-level required key, the keys the chosen
// connection mode requires, each top-level key a requiredIf or requiredUnless
// rule requires once the defaults apply, and the connection_mode entry. A
// secret holds its slot and any other key null, so validate names each key
// still to fill. The mode is cloud when the source token allows it, or the
// only mode there is; otherwise the entry is left out for the caller to
// choose.
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
	for _, f := range doc.Fields {
		if requiredByDefaults(f, fields) && !slices.Contains(keys, f.Key) {
			keys = append(keys, f.Key)
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
	root.HeadComment = templateNote
	if len(doc.CustomValidators) > 0 {
		root.HeadComment += "\n" + customValidatorNote
	}
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

// requiredByDefaults reports whether a top-level key's requiredIf or
// requiredUnless rule holds on the defaults alone. Validate applies the
// defaults before it checks, so the rule holds in any spec that leaves the
// condition keys out, as the template does. A condition key without a default
// takes the caller's value, so it never decides here.
func requiredByDefaults(f Field, fields map[string]Field) bool {
	if strings.Contains(f.Key, ".") {
		return false
	}
	requiredIf := len(f.RequiredIf) > 0 && !slices.ContainsFunc(f.RequiredIf, func(c Condition) bool {
		met, decided := metByDefault(c, fields)
		return !decided || !met
	})
	requiredUnless := len(f.RequiredUnless) > 0 && !slices.ContainsFunc(f.RequiredUnless, func(c Condition) bool {
		met, decided := metByDefault(c, fields)
		return !decided || met
	})
	return requiredIf || requiredUnless
}

// metByDefault reports whether the default of the condition's key meets the
// condition, and whether the key has a default that decides it at all.
func metByDefault(c Condition, fields map[string]Field) (met, decided bool) {
	key, ok := fields[c.Key]
	if !ok || key.Default == nil || key.DefaultRequiresParent {
		return false, false
	}
	return reflect.DeepEqual(key.Default, c.Value), true
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
