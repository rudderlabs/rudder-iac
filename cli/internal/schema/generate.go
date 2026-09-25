package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	invopop "github.com/invopop/jsonschema"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema/editor"
)

const (
	RootFileName  = "rudder-spec.schema.json"
	SchemaBaseURL = "https://github.com/rudderlabs/rudder-iac/releases/latest/download/schemas"
)

var ErrUnknownKind = errors.New("unknown spec kind")

type Catalog struct {
	variants map[string][]provider.SchemaVariant
}

func New(schemaProvider provider.SchemaProvider) (*Catalog, error) {
	variants := schemaProvider.SpecSchemas()
	for kind, entries := range variants {
		if len(entries) == 0 {
			return nil, fmt.Errorf("kind %q has no schema variants", kind)
		}
		for _, entry := range entries {
			if entry.Spec == nil {
				return nil, fmt.Errorf("kind %q has a nil schema sample", kind)
			}
		}
	}
	return &Catalog{variants: variants}, nil
}

func (c *Catalog) Kinds() []string {
	kinds := make([]string, 0, len(c.variants))
	for kind := range c.variants {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

func (c *Catalog) ForKind(kind string) (*invopop.Schema, error) {
	variants, ok := c.variants[kind]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
	}

	definitions := invopop.Definitions{}
	branches := make([]*invopop.Schema, 0, len(variants))
	for i, variant := range variants {
		branch := specBlock(variant.Spec)
		for name, replacement := range variant.Replacements {
			replaceProperty(branch, name, schemaFor(replacement))
		}
		for name, value := range variant.Constants {
			setPropertyConst(branch, name, value)
		}
		for _, transform := range variant.Transforms {
			transform(branch)
		}
		hoistDefinitions(branch, fmt.Sprintf("Spec%d", i), definitions)
		for _, version := range variantVersions(variant) {
			branches = append(branches, envelopeBranch(kind, version, branch, definitions))
		}
	}

	return kindSchema(kind, branches, definitions), nil
}

func (c *Catalog) Root() (*invopop.Schema, error) {
	definitions := invopop.Definitions{}
	branches := make([]*invopop.Schema, 0, len(c.variants))
	for _, kind := range c.Kinds() {
		s, err := c.ForKind(kind)
		if err != nil {
			return nil, err
		}
		s.Version = ""
		s.ID = ""
		hoistDefinitions(s, kind, definitions)
		branches = append(branches, s)
	}
	return &invopop.Schema{
		Version:     invopop.Version,
		ID:          invopop.ID(SchemaBaseURL + "/" + RootFileName),
		Title:       "RudderStack CLI spec",
		Definitions: definitions,
		OneOf:       branches,
	}, nil
}

func (c *Catalog) MarshalKind(kind string) ([]byte, error) {
	s, err := c.ForKind(kind)
	if err != nil {
		return nil, err
	}
	return marshal(s)
}

func (c *Catalog) MarshalRoot() ([]byte, error) {
	s, err := c.Root()
	if err != nil {
		return nil, err
	}
	return marshal(s)
}

func FileName(kind string) string {
	return editor.FileName(kind)
}

func URL(kind string) string {
	return SchemaBaseURL + "/" + FileName(kind)
}

func specBlock(sample any) *invopop.Schema {
	reflector := &invopop.Reflector{
		Anonymous:                 true,
		ExpandedStruct:            true,
		AllowAdditionalProperties: false,
		FieldNameTag:              preferredFieldTag(deref(reflect.TypeOf(sample))),
	}
	t := reflect.TypeOf(sample)
	s := reflector.ReflectFromType(t)
	s.Version = ""
	s.ID = ""
	rootName := deref(t).Name()
	if rootName != "" && hasRef(s, "#/$defs/"+rootName) {
		s.Definitions[rootName] = cloneSchemaWithoutDefinitions(s)
	}
	(&enricher{defs: s.Definitions, visited: map[reflect.Type]bool{}}).walk(s, t)
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

func schemaFor(sample any) *invopop.Schema {
	if s, ok := sample.(*invopop.Schema); ok {
		return s
	}
	return specBlock(sample)
}

func replaceProperty(s *invopop.Schema, name string, replacement *invopop.Schema) {
	if s.Properties == nil {
		return
	}
	mergeDefinitions(s.Definitions, replacement.Definitions)
	replacement.Definitions = nil
	s.Properties.Set(name, replacement)
}

func setPropertyConst(s *invopop.Schema, name string, value any) {
	if s.Properties == nil {
		return
	}
	property, ok := s.Properties.Get(name)
	if !ok {
		return
	}
	property.Const = value
}

func mergeDefinitions(target, source invopop.Definitions) {
	for name, definition := range source {
		target[name] = definition
	}
}

func variantVersions(variant provider.SchemaVariant) []string {
	if len(variant.Versions) > 0 {
		return variant.Versions
	}
	return []string{specs.SpecVersionV1}
}

func kindSchema(kind string, branches []*invopop.Schema, definitions invopop.Definitions) *invopop.Schema {
	var branchSchema *invopop.Schema
	if len(branches) == 1 {
		branchSchema = branches[0]
	} else {
		branchSchema = &invopop.Schema{OneOf: branches}
	}
	branchSchema.Version = invopop.Version
	branchSchema.ID = invopop.ID(SchemaBaseURL + "/" + FileName(kind))
	branchSchema.Title = fmt.Sprintf("RudderStack %s spec", kind)
	branchSchema.Definitions = definitions
	return branchSchema
}

func envelopeBranch(kind string, version string, spec *invopop.Schema, definitions invopop.Definitions) *invopop.Schema {
	metadata := specBlock(specs.Metadata{})
	hoistDefinitions(metadata, "Metadata", definitions)

	properties := invopop.NewProperties()
	properties.Set("version", &invopop.Schema{Type: "string", Const: version})
	properties.Set("kind", &invopop.Schema{Type: "string", Const: kind})
	properties.Set("metadata", metadata)
	properties.Set("spec", spec)
	return &invopop.Schema{
		Type:                 "object",
		Properties:           properties,
		Required:             []string{"version", "kind", "metadata", "spec"},
		AdditionalProperties: invopop.FalseSchema,
	}
}

func hoistDefinitions(s *invopop.Schema, namespace string, target invopop.Definitions) {
	if len(s.Definitions) == 0 {
		return
	}
	renames := make(map[string]string, len(s.Definitions))
	for name := range s.Definitions {
		renames[name] = namespace + name
	}
	rewriteRefs(s, renames)
	for name, definition := range s.Definitions {
		target[renames[name]] = definition
	}
	s.Definitions = nil
}

func hasRef(s *invopop.Schema, ref string) bool {
	if s == nil {
		return false
	}
	if s.Ref == ref {
		return true
	}
	for _, child := range childSchemas(s) {
		if hasRef(child, ref) {
			return true
		}
	}
	return false
}

func cloneSchemaWithoutDefinitions(s *invopop.Schema) *invopop.Schema {
	clone := *s
	clone.Definitions = nil
	return &clone
}

func rewriteRefs(s *invopop.Schema, renames map[string]string) {
	if s == nil {
		return
	}
	const prefix = "#/$defs/"
	if name, ok := strings.CutPrefix(s.Ref, prefix); ok {
		if replacement, found := renames[name]; found {
			s.Ref = prefix + replacement
		}
	}
	for _, child := range childSchemas(s) {
		rewriteRefs(child, renames)
	}
}

func childSchemas(s *invopop.Schema) []*invopop.Schema {
	children := append(append(append([]*invopop.Schema{}, s.AllOf...), s.AnyOf...), s.OneOf...)
	children = append(children, s.Not, s.If, s.Then, s.Else, s.Items, s.Contains, s.AdditionalProperties, s.PropertyNames)
	children = append(children, s.PrefixItems...)
	if s.Properties != nil {
		for child := range s.Properties.ValuesFromOldest() {
			children = append(children, child)
		}
	}
	for _, child := range s.PatternProperties {
		children = append(children, child)
	}
	for _, child := range s.DependentSchemas {
		children = append(children, child)
	}
	for _, child := range s.Definitions {
		children = append(children, child)
	}
	return children
}

func marshal(s *invopop.Schema) ([]byte, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling schema: %w", err)
	}
	return append(data, '\n'), nil
}
