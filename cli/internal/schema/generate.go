package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ForKind reflects the full Draft 2020-12 schema for one spec kind.
func ForKind(kind string, sample any) *jsonschema.Schema {
	return envelope(kind, nil, sample)
}

// ForKindVersions reflects one body shared by an explicit set of spec versions.
func ForKindVersions(kind string, sample any, versions ...string) *jsonschema.Schema {
	versions = append([]string(nil), versions...)
	slices.Sort(versions)
	return envelope(kind, versions, sample)
}

// ForVersionedKind reflects one spec kind whose body differs by spec version.
func ForVersionedKind(kind string, samples map[string]any) *jsonschema.Schema {
	branches := make([]*jsonschema.Schema, 0, len(samples))
	for _, version := range sortedKeys(samples) {
		branches = append(branches, envelope(kind, []string{version}, samples[version]))
	}
	return combinedSchema(fmt.Sprintf("RudderStack %s spec", kind), branches)
}

// VersionsForKind extracts the concrete versions a provider advertises for kind.
func VersionsForKind(kind string, patterns []vrules.MatchPattern) []string {
	versions := make([]string, 0, len(patterns))
	seen := make(map[string]struct{}, len(patterns))
	for _, pattern := range patterns {
		if pattern.Kind != kind || pattern.Version == "*" {
			continue
		}
		if _, ok := seen[pattern.Version]; ok {
			continue
		}
		seen[pattern.Version] = struct{}{}
		versions = append(versions, pattern.Version)
	}
	slices.Sort(versions)
	return versions
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// Root combines provider-owned schemas into one kind-discriminated schema.
// Definitions are namespaced per kind because reflected types can share names
// while representing different provider contracts.
func Root(schemas Set) (*jsonschema.Schema, error) {
	branches := make([]*jsonschema.Schema, 0, len(schemas))
	definitions := make(jsonschema.Definitions)
	for _, kind := range Kinds(schemas) {
		branch, err := cloneSchema(schemas[kind])
		if err != nil {
			return nil, fmt.Errorf("cloning schema for %q: %w", kind, err)
		}
		branch.Version = ""
		for name, definition := range branch.Definitions {
			namespaced := kind + "-" + name
			rewriteRefs(definition, "#/$defs/"+name, "#/$defs/"+namespaced)
			definitions[namespaced] = definition
			rewriteRefs(branch, "#/$defs/"+name, "#/$defs/"+namespaced)
		}
		branch.Definitions = nil
		branches = append(branches, branch)
	}

	return &jsonschema.Schema{
		Version:     jsonschema.Version,
		Title:       "RudderStack CLI spec",
		OneOf:       branches,
		Definitions: definitions,
	}, nil
}

func cloneSchema(source *jsonschema.Schema) (*jsonschema.Schema, error) {
	raw, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}
	var clone jsonschema.Schema
	if err := json.Unmarshal(raw, &clone); err != nil {
		return nil, err
	}
	return &clone, nil
}

func rewriteRefs(node *jsonschema.Schema, oldRef, newRef string) {
	if node == nil {
		return
	}
	if node.Ref == oldRef {
		node.Ref = newRef
	}
	for _, child := range append(append(append([]*jsonschema.Schema{}, node.AllOf...), node.AnyOf...), node.OneOf...) {
		rewriteRefs(child, oldRef, newRef)
	}
	for _, child := range []*jsonschema.Schema{node.Not, node.If, node.Then, node.Else, node.Items, node.Contains, node.AdditionalProperties, node.PropertyNames, node.ContentSchema} {
		rewriteRefs(child, oldRef, newRef)
	}
	if node.Properties != nil {
		for pair := node.Properties.Oldest(); pair != nil; pair = pair.Next() {
			rewriteRefs(pair.Value, oldRef, newRef)
		}
	}
	for _, children := range []map[string]*jsonschema.Schema{node.PatternProperties, node.DependentSchemas, node.Definitions} {
		for _, child := range children {
			rewriteRefs(child, oldRef, newRef)
		}
	}
	for _, child := range node.PrefixItems {
		rewriteRefs(child, oldRef, newRef)
	}
	if strings.HasPrefix(node.DynamicRef, oldRef) {
		node.DynamicRef = strings.Replace(node.DynamicRef, oldRef, newRef, 1)
	}
}

// specBlock reflects a kind's `spec:` struct and enriches it with constraints
// carried on `validate` struct tags, which the reflector does not understand on
// its own. Nested and recursive types are represented via $defs/$ref (the
// reflector's default) so self-referential spec structs terminate cleanly.
func reflectedFieldName(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func specBlock(sample any) *jsonschema.Schema {
	r := &jsonschema.Reflector{
		ExpandedStruct: true,
		KeyNamer:       reflectedFieldName,
	}

	t := reflect.TypeOf(sample)
	s := r.ReflectFromType(t)
	// The spec block lives inside the envelope; it declares its own dialect only
	// at the envelope root.
	s.Version = ""
	s.ID = ""

	enrich := &enricher{defs: s.Definitions, visitedNodes: map[enrichmentVisit]bool{}}
	enrich.walk(s, t)
	return s
}

// envelope wraps a reflected spec block in the standard rudder spec envelope.
func envelope(kind string, versions []string, sample any) *jsonschema.Schema {
	props := jsonschema.NewProperties()
	versionSchema := &jsonschema.Schema{
		Type:        "string",
		Description: "Spec format version, e.g. rudder/v1.",
	}
	switch len(versions) {
	case 1:
		versionSchema.Const = versions[0]
	case 2:
		versionSchema.Enum = []any{versions[0], versions[1]}
	default:
		if len(versions) > 0 {
			versionSchema.Enum = make([]any, len(versions))
			for i, version := range versions {
				versionSchema.Enum[i] = version
			}
		}
	}
	props.Set("version", versionSchema)
	props.Set("kind", &jsonschema.Schema{
		Type:  "string",
		Const: kind,
	})
	metadata := specBlock(specs.Metadata{})
	metadata.Description = "Spec metadata (name and optional import block)."
	spec := specBlock(sample)

	// $ref pointers ("#/$defs/...") resolve against the document root, so nested
	// definitions must live at the envelope root, not under a property where the
	// references would dangle.
	defs := metadata.Definitions
	for name, definition := range spec.Definitions {
		defs[name] = definition
	}
	metadata.Definitions = nil
	spec.Definitions = nil
	props.Set("metadata", metadata)
	props.Set("spec", spec)

	return &jsonschema.Schema{
		Version:              jsonschema.Version,
		Title:                fmt.Sprintf("RudderStack %s spec", kind),
		Type:                 "object",
		Properties:           props,
		Required:             []string{"version", "kind", "metadata", "spec"},
		AdditionalProperties: jsonschema.FalseSchema,
		Definitions:          defs,
	}
}

func combinedSchema(title string, branches []*jsonschema.Schema) *jsonschema.Schema {
	definitions := make(jsonschema.Definitions)
	for index, branch := range branches {
		branch.Version = ""
		for name, definition := range branch.Definitions {
			namespaced := fmt.Sprintf("branch-%d-%s", index, name)
			rewriteRefs(definition, "#/$defs/"+name, "#/$defs/"+namespaced)
			definitions[namespaced] = definition
			rewriteRefs(branch, "#/$defs/"+name, "#/$defs/"+namespaced)
		}
		branch.Definitions = nil
	}
	return &jsonschema.Schema{
		Version:     jsonschema.Version,
		Title:       title,
		OneOf:       branches,
		Definitions: definitions,
	}
}
