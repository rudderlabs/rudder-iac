package destination

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
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
}

// NewProvider constructs the destination provider with the given API client and
// definition registry. The registry is expected to be populated by the caller;
// this constructor registers no definitions itself.
func NewProvider(c *client.Client, registry *definitions.Registry) *Provider {
	return &Provider{
		BaseProvider: provider.NewBaseProvider([]provider.Handler{
			NewHandler(c, registry),
		}),
	}
}

// LoadLegacySpec rejects legacy spec versions — destinations are v1-only.
func (p *Provider) LoadLegacySpec(_ string, s *specs.Spec) error {
	return fmt.Errorf("destination specs require version '%s', got '%s'. Legacy versions are not supported", specs.SpecVersionV1, s.Version)
}

// SupportedMatchPatterns declares the (kind, version) pairs this provider fully
// handles. Destinations support only the V1 spec version.
func (p *Provider) SupportedMatchPatterns() []vrules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

// SyntacticRules returns no rules this ticket — full validation is RUD-2853.
func (p *Provider) SyntacticRules() []vrules.Rule {
	return nil
}

// SemanticRules returns no rules this ticket — full validation is RUD-2853.
func (p *Provider) SemanticRules() []vrules.Rule {
	return nil
}

// RuleDocEntries returns no authored fragments yet — destination validation
// rules and their docs land with RUD-2853.
func (p *Provider) RuleDocEntries() []vdocs.RuleDocEntry {
	return nil
}
