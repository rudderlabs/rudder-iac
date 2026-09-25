package destination

import (
	"fmt"
	"strings"

	invopop "github.com/invopop/jsonschema"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	destdocs "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/docs"
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

// SupportedMatchPatterns declares the (kind, version) pairs this provider fully
// handles. Destinations support only the V1 spec version.
func (p *Provider) SpecSchemas() map[string][]provider.SchemaVariant {
	variants := make([]provider.SchemaVariant, 0, len(p.registry.Definitions()))
	for _, definition := range p.registry.Definitions() {
		variants = append(variants, provider.SchemaVariant{
			Spec: DestinationSpec{},
			Replacements: map[string]any{
				"config": definition.NewConfigSchema(),
			},
			Constants: map[string]any{
				"type":               definition.Type,
				"definition_version": definition.Version,
			},
			Transforms: []provider.SchemaTransform{
				destinationDefinitionTransform(definition),
			},
		})
	}
	return map[string][]provider.SchemaVariant{DestinationSpecKind: variants}
}

func destinationDefinitionTransform(definition *definitions.RegisteredDefinition) provider.SchemaTransform {
	return func(spec *invopop.Schema) {
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
		restrictConnectionMode(config, definition)
		restrictConsentManagement(config, definition)
	}
}

func resolveLocalDefinition(spec *invopop.Schema, ref string) *invopop.Schema {
	const prefix = "#/$defs/"
	name, ok := strings.CutPrefix(ref, prefix)
	if !ok || spec.Definitions == nil {
		return nil
	}
	return spec.Definitions[name]
}

func restrictConnectionMode(config *invopop.Schema, definition *definitions.RegisteredDefinition) {
	property, ok := config.Properties.Get("connection_mode")
	if !ok {
		return
	}

	properties := invopop.NewProperties()
	for _, sourceType := range definition.SupportedSourceTypes() {
		modes, err := definition.ConnectionModes(sourceType)
		if err != nil {
			continue
		}
		properties.Set(sourceType, &invopop.Schema{Type: "string", Enum: stringsToAny(modes)})
	}
	property.Ref = ""
	property.Type = "object"
	property.Properties = properties
	property.AdditionalProperties = invopop.FalseSchema
}

func restrictConsentManagement(config *invopop.Schema, definition *definitions.RegisteredDefinition) {
	property, ok := config.Properties.Get("consent_management")
	if !ok {
		return
	}

	properties := invopop.NewProperties()
	for _, sourceType := range definition.SupportedSourceTypes() {
		properties.Set(sourceType, invopop.TrueSchema)
	}
	property.Ref = ""
	property.Type = "object"
	property.Properties = properties
	property.AdditionalProperties = invopop.FalseSchema
}

func stringsToAny(values []string) []any {
	items := make([]any, len(values))
	for i, value := range values {
		items[i] = value
	}
	return items
}

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
