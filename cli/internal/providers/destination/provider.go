package destination

import (
	"fmt"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	destdocs "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	vdocs "github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// importDir is the top-level base directory import output is written under.
// Destinations own their full path via ImportPath ("destinations/"), so the
// base stays empty rather than nesting under another provider's directory.
const importDir = ""

// Provider wraps BaseProvider with the single destination handler.
type Provider struct {
	*provider.BaseProvider
	registry *definitions.Registry
}

// NewProvider constructs the destination provider with the given API client and
// definition registry. The registry is expected to be populated by the caller;
// this constructor registers no definitions itself.
func NewProvider(c *client.Client, registry *definitions.Registry) *Provider {
	return &Provider{
		BaseProvider: provider.NewBaseProvider([]provider.Handler{
			NewHandler(c, registry),
		}),
		registry: registry,
	}
}

// LoadLegacySpec rejects legacy spec versions — destinations are v1-only.
func (p *Provider) LoadLegacySpec(_ string, s *specs.Spec) error {
	return fmt.Errorf("destination specs require version '%s', got '%s'. Legacy versions are not supported", specs.SpecVersionV1, s.Version)
}

// SpecSchemas returns one destination branch per registered type/version so
// editors see the concrete config fields and connection-mode constraints.
func (p *Provider) SpecSchemas() schema.Set {
	definitions := p.registry.Definitions()
	variants := make([]schema.Variant, 0, len(definitions))
	for _, definition := range definitions {
		variants = append(variants, schema.Variant{
			Sample: DestinationSpec{},
			Replacements: map[string]any{
				"config": definition.NewConfigSchema(),
			},
			Constants: map[string]any{
				"type":               definition.Type,
				"definition_version": definition.Version,
			},
			Transforms: []schema.Transform{destinationDefinitionTransform(definition)},
		})
	}
	versions := schema.VersionsForKind(DestinationSpecKind, p.SupportedMatchPatterns())
	return schema.Set{
		DestinationSpecKind: schema.MustForKindVariants(DestinationSpecKind, versions, variants...),
	}
}

func destinationDefinitionTransform(definition *definitions.RegisteredDefinition) schema.Transform {
	return func(spec *jsonschema.Schema) {
		config, ok := spec.Properties.Get("config")
		if !ok {
			return
		}
		if config.Ref != "" {
			config = resolveLocalDefinition(spec, config.Ref)
		}
		if config == nil || config.Properties == nil {
			return
		}
		// Destination defaults and custom validators can satisfy or replace many
		// field-level `required` tags. Keep generated config schemas structural so
		// they do not reject specs accepted after ApplyDefaults.
		clearRequiredConstraints(config, spec.Definitions, map[*jsonschema.Schema]bool{})
		restrictConnectionMode(config, definition)
	}
}

func clearRequiredConstraints(node *jsonschema.Schema, defs jsonschema.Definitions, seen map[*jsonschema.Schema]bool) {
	if node == nil || seen[node] {
		return
	}
	seen[node] = true
	if node.Ref != "" {
		if resolved := resolveLocalDefinition(&jsonschema.Schema{Definitions: defs}, node.Ref); resolved != nil {
			clearRequiredConstraints(resolved, defs, seen)
		}
		return
	}
	node.Required = nil
	for _, child := range schemaNodeChildren(node) {
		clearRequiredConstraints(child, defs, seen)
	}
}

func schemaNodeChildren(node *jsonschema.Schema) []*jsonschema.Schema {
	children := append(append(append([]*jsonschema.Schema{}, node.AllOf...), node.AnyOf...), node.OneOf...)
	children = append(children, node.Not, node.If, node.Then, node.Else, node.Items, node.Contains, node.AdditionalProperties, node.PropertyNames, node.ContentSchema)
	children = append(children, node.PrefixItems...)
	if node.Properties != nil {
		for pair := node.Properties.Oldest(); pair != nil; pair = pair.Next() {
			children = append(children, pair.Value)
		}
	}
	return children
}

func resolveLocalDefinition(spec *jsonschema.Schema, ref string) *jsonschema.Schema {
	const prefix = "#/$defs/"
	name, ok := strings.CutPrefix(ref, prefix)
	if !ok || spec.Definitions == nil {
		return nil
	}
	return spec.Definitions[name]
}

func restrictConnectionMode(config *jsonschema.Schema, definition *definitions.RegisteredDefinition) {
	property, ok := config.Properties.Get("connection_mode")
	if !ok {
		return
	}

	properties := jsonschema.NewProperties()
	for _, sourceType := range definition.SupportedSourceTypes() {
		modes, err := definition.ConnectionModes(sourceType)
		if err != nil {
			continue
		}
		values := make([]any, len(modes))
		for i, mode := range modes {
			values[i] = mode
		}
		properties.Set(sourceType, &jsonschema.Schema{Type: "string", Enum: values})
	}
	property.Ref = ""
	property.Type = "object"
	property.Properties = properties
	property.AdditionalProperties = jsonschema.FalseSchema
}

// SupportedMatchPatterns declares the (kind, version) pairs this provider fully
// handles. Destinations support only the V1 spec version.
func (p *Provider) SupportedMatchPatterns() []vrules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

func (p *Provider) SyntacticRules() []vrules.Rule {
	return []vrules.Rule{
		NewSpecSyntaxValidRule(p.registry),
	}
}

func (p *Provider) SemanticRules() []vrules.Rule {
	return []vrules.Rule{
		NewSemanticValidRule(),
	}
}

// RuleDocEntries returns the authored documentation fragments embedded with
// the destination provider, joined to registered rules by the docs generator.
func (p *Provider) RuleDocEntries() []vdocs.RuleDocEntry {
	entries, _ := vdocs.LoadRuleDocEntries(destdocs.FragmentsFS, ".")
	return entries
}
