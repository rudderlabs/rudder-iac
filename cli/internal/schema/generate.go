package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/invopop/jsonschema"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/editor"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

var ErrNilSpec = errors.New("schema sample must not be nil")

// Transform adjusts a reflected spec-body schema for a provider-owned variant.
type Transform func(*jsonschema.Schema)

// Variant describes one concrete body selected by fields within a spec kind.
type Variant struct {
	Sample       any
	Replacements map[string]any
	Constants    map[string]any
	Transforms   []Transform
}

// ForKind reflects the full Draft 2020-12 schema for one spec kind.
func ForKind(kind string, sample any) (*jsonschema.Schema, error) {
	return ForKindVersions(kind, sample)
}

// MustForKind is for provider declarations whose concrete sample is known at
// compile time. It panics only when the declaration itself is invalid.
func MustForKind(kind string, sample any) *jsonschema.Schema {
	s, err := ForKind(kind, sample)
	if err != nil {
		panic(err)
	}
	return s
}

// ForKindVersions reflects one body shared by an explicit set of spec versions.
func ForKindVersions(kind string, sample any, versions ...string) (*jsonschema.Schema, error) {
	if isNilSample(sample) {
		return nil, ErrNilSpec
	}
	versions = append([]string(nil), versions...)
	slices.Sort(versions)
	return identifyKindSchema(kind, envelope(kind, versions, sample)), nil
}

// MustForKindVersions is the provider-declaration form of ForKindVersions.
func MustForKindVersions(kind string, sample any, versions ...string) *jsonschema.Schema {
	s, err := ForKindVersions(kind, sample, versions...)
	if err != nil {
		panic(err)
	}
	return s
}

// ForVersionedKind reflects one spec kind whose body differs by spec version.
func ForVersionedKind(kind string, samples map[string]any) (*jsonschema.Schema, error) {
	branches := make([]*jsonschema.Schema, 0, len(samples))
	for _, version := range sortedKeys(samples) {
		if isNilSample(samples[version]) {
			return nil, fmt.Errorf("version %q: %w", version, ErrNilSpec)
		}
		branches = append(branches, envelope(kind, []string{version}, samples[version]))
	}
	return identifyKindSchema(kind, combinedSchema(fmt.Sprintf("RudderStack %s spec", kind), branches)), nil
}

// MustForVersionedKind is the provider-declaration form of ForVersionedKind.
func MustForVersionedKind(kind string, samples map[string]any) *jsonschema.Schema {
	s, err := ForVersionedKind(kind, samples)
	if err != nil {
		panic(err)
	}
	return s
}

// ForKindVariants reflects discriminator-selected body variants for one kind.
func ForKindVariants(kind string, versions []string, variants ...Variant) (*jsonschema.Schema, error) {
	branches := make([]*jsonschema.Schema, 0, len(variants))
	for index, variant := range variants {
		if isNilSample(variant.Sample) {
			return nil, fmt.Errorf("variant %d: %w", index, ErrNilSpec)
		}
		body := specBlock(variant.Sample)
		for name, replacement := range variant.Replacements {
			if isNilSample(replacement) {
				return nil, fmt.Errorf("variant %d replacement %q: %w", index, name, ErrNilSpec)
			}
			replaceProperty(body, name, specBlock(replacement))
		}
		for name, value := range variant.Constants {
			setPropertyConst(body, name, value)
		}
		for _, transform := range variant.Transforms {
			transform(body)
		}
		branches = append(branches, envelopeWithSpec(kind, versions, body))
	}
	return identifyKindSchema(kind, combinedSchema(fmt.Sprintf("RudderStack %s spec", kind), branches)), nil
}

// MustForKindVariants is the provider-declaration form of ForKindVariants.
func MustForKindVariants(kind string, versions []string, variants ...Variant) *jsonschema.Schema {
	s, err := ForKindVariants(kind, versions, variants...)
	if err != nil {
		panic(err)
	}
	return s
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
		branch.ID = ""
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
		ID:          jsonschema.ID(strings.TrimRight(editor.DefaultSchemaURLBase, "/") + "/" + RootFileName),
		Title:       "RudderStack CLI spec",
		OneOf:       branches,
		Definitions: definitions,
	}, nil
}

func isNilSample(sample any) bool {
	if sample == nil {
		return true
	}
	value := reflect.ValueOf(sample)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func identifyKindSchema(kind string, s *jsonschema.Schema) *jsonschema.Schema {
	s.ID = jsonschema.ID(editor.URL(kind))
	return s
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

func reflectedFieldName(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func specBlock(sample any) *jsonschema.Schema {
	t := reflect.TypeOf(sample)
	r := &jsonschema.Reflector{
		ExpandedStruct: true,
		FieldNameTag:   preferredFieldTag(deref(t)),
		KeyNamer:       reflectedFieldName,
	}

	s := r.ReflectFromType(t)
	// The spec block lives inside the envelope; it declares its own dialect only
	// at the envelope root.
	s.Version = ""
	s.ID = ""

	enrich := &enricher{defs: s.Definitions, visitedNodes: map[enrichmentVisit]bool{}}
	enrich.walk(s, t)
	return s
}

func preferredFieldTag(t reflect.Type) string {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		if tag := field.Tag.Get("json"); tag != "" && strings.Split(tag, ",")[0] != "" {
			return "json"
		}
	}
	return "mapstructure"
}

func replaceProperty(s *jsonschema.Schema, name string, replacement *jsonschema.Schema) {
	if s.Properties == nil {
		return
	}
	if s.Definitions == nil {
		s.Definitions = make(jsonschema.Definitions)
	}
	for definitionName, definition := range replacement.Definitions {
		s.Definitions[definitionName] = definition
	}
	replacement.Definitions = nil
	s.Properties.Set(name, replacement)
}

func setPropertyConst(s *jsonschema.Schema, name string, value any) {
	if s.Properties == nil {
		return
	}
	property, ok := s.Properties.Get(name)
	if !ok {
		return
	}
	property.Const = value
}

// envelope wraps a reflected spec block in the standard rudder spec envelope.
func envelope(kind string, versions []string, sample any) *jsonschema.Schema {
	return envelopeWithSpec(kind, versions, specBlock(sample))
}

func envelopeWithSpec(kind string, versions []string, spec *jsonschema.Schema) *jsonschema.Schema {
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
