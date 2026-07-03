package destination

import (
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type Provider struct {
	*provider.BaseProvider
}

func NewProvider(c *client.Client, registry *definitions.Registry) *Provider {
	return &Provider{
		BaseProvider: provider.NewBaseProvider([]provider.Handler{
			NewHandler(c, registry),
		}),
	}
}

func (p *Provider) LoadLegacySpec(_ string, s *specs.Spec) error {
	return &provider.ErrUnsupportedSpecKind{Kind: s.Kind}
}

func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	return prules.V1VersionPatterns(DestinationSpecKind)
}

func (p *Provider) SyntacticRules() []rules.Rule {
	return nil
}

func (p *Provider) SemanticRules() []rules.Rule {
	return nil
}
