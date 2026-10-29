package destination

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	destdocs "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/docs"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/planner"
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

// The composite finds plan checkers by type assertion, so a drifted CheckPlan
// signature would silently stop the check from running.
var _ provider.PlanChecker = &Provider{}

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

// CheckPlan refuses a destination linked to a transformation the same plan
// creates. A new transformation is published only once the apply finishes, and
// the API links a destination only to a published one, so the link would fail
// partway through every apply. See provider.PlanChecker.
func (p *Provider) CheckPlan(_ context.Context, plan *planner.Plan, _ *state.State) error {
	created := make(map[string]bool)
	for _, op := range plan.Operations {
		if op.Type == planner.Create && op.Resource.Type() == ttypes.TransformationResourceType {
			created[op.Resource.URN()] = true
		}
	}

	var refused []string
	for _, op := range plan.Operations {
		dest, ok := op.Resource.RawData().(*DestinationResource)
		if !ok || dest.Transformation == nil || !created[dest.Transformation.URN] {
			continue
		}
		refused = append(refused, fmt.Sprintf("%s links #%s", op.Resource.URN(), dest.Transformation.URN))
	}
	if len(refused) == 0 {
		return nil
	}
	return fmt.Errorf("cannot link a destination to a transformation created in the same apply: "+
		"a new transformation is published only once the apply finishes, and only a published "+
		"transformation can be linked. Apply the new transformation first, leaving the destination's "+
		"transformation as it is, then link it in a second apply:\n  %s", strings.Join(refused, "\n  "))
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

// SyntacticRules returns the destination syntactic rules. Each is wrapped by
// maskSecretLines, so a result whose line can show a secret is masked.
func (p *Provider) SyntacticRules() []vrules.Rule {
	return maskSecretLines(p.registry,
		NewSpecSyntaxValidRule(p.registry),
		NewSecretReferenceRule(p.registry),
		NewSecretReferenceUniqueRule(p.registry),
	)
}

// SemanticRules returns the destination semantic rules, wrapped like the
// syntactic ones.
func (p *Provider) SemanticRules() []vrules.Rule {
	return maskSecretLines(p.registry,
		NewSemanticValidRule(),
	)
}

// RuleDocEntries returns the authored documentation fragments embedded with
// the destination provider, joined to registered rules by the docs generator.
func (p *Provider) RuleDocEntries() []vdocs.RuleDocEntry {
	entries, _ := vdocs.LoadRuleDocEntries(destdocs.FragmentsFS, ".")
	return entries
}
