package destination

import (
	"context"
	"fmt"
	"strings"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/namer"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/writer"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	destdocs "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/docs"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resolver"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources/state"
	"github.com/rudderlabs/rudder-iac/cli/internal/syncer/differ"
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
	handler  *HandlerImpl
}

var (
	_ provider.PlanChecker     = &Provider{}
	_ provider.ImportPreviewer = &Provider{}
)

// NewProvider constructs the destination provider with the given API client and
// definition registry. The registry is expected to be populated by the caller;
// this constructor registers no definitions itself.
func NewProvider(c *client.Client, registry *definitions.Registry) *Provider {
	impl := &HandlerImpl{client: c, registry: registry}
	return &Provider{
		BaseProvider: provider.NewBaseProvider([]provider.Handler{handler.NewHandler(impl)}),
		registry:     registry,
		handler:      impl,
	}
}

// RETLConnectionResourceType is retl/connection.ResourceType, which imports
// this package and so cannot be imported here; a test pins the two together.
const RETLConnectionResourceType = "retl-connection"

// warehouseConnection is a remote Reverse ETL connection in an import.
type warehouseConnection interface {
	WarehouseDestinationID() string
}

// FormatForExport exports importable destinations. Which secrets a destination
// needs at connect time depends on the sources it connects to, so the
// destinations the same import connects a warehouse source to are noted first
// (see requiredSecretKeys).
func (p *Provider) FormatForExport(
	collection *resources.RemoteResources,
	idNamer namer.Namer,
	inputResolver resolver.ReferenceResolver,
) ([]writer.FormattableEntity, []importmanifest.ImportEntry, error) {
	connected := map[string]bool{}
	for _, res := range collection.GetAll(RETLConnectionResourceType) {
		if conn, ok := res.Data.(warehouseConnection); ok {
			connected[conn.WarehouseDestinationID()] = true
		}
	}
	p.handler.warehouseConnected = connected
	return p.BaseProvider.FormatForExport(collection, idNamer, inputResolver)
}

// PreviewImports reports, per destination the plan imports, the config the
// import writes over the remote one. Import pushes the spec to the destination
// it adopts, so a value the spec does not share with the remote changes it. Per
// destination the plan updates, it reports only the remote keys the update
// removes that the plan's own comparison can't show. See
// provider.ImportPreviewer.
func (p *Provider) PreviewImports(ctx context.Context, plan *planner.Plan, st *state.State) (map[string]differ.ResourceDiff, error) {
	// A resource the same plan imports keeps its remote ID, so a reference to it
	// resolves to that ID once its import runs.
	adopted := map[string]string{}
	for _, op := range plan.Operations {
		if op.Type == planner.Import && op.Resource.ImportMetadata() != nil {
			adopted[op.Resource.URN()] = op.Resource.ImportMetadata().RemoteId
		}
	}

	result := map[string]differ.ResourceDiff{}
	for _, op := range plan.Operations {
		if op.Resource.Type() != DestinationResourceType {
			continue
		}
		data, ok := op.Resource.RawData().(*DestinationResource)
		if !ok {
			continue
		}
		if op.Type == planner.Update {
			urn := op.Resource.URN()
			diffs, err := p.handler.previewUpdate(data, st.GetResource(urn))
			if err != nil {
				return nil, fmt.Errorf("previewing update of %s: %w", urn, err)
			}
			if len(diffs) > 0 {
				result[urn] = differ.ResourceDiff{URN: urn, Diffs: diffs}
			}
			continue
		}
		if op.Type != planner.Import || op.Resource.ImportMetadata() == nil {
			continue
		}
		diffs, secretOnly, err := p.handler.previewImport(ctx, data, op.Resource.ImportMetadata().RemoteId, st, adopted)
		if err != nil {
			return nil, fmt.Errorf("previewing import of %s: %w", op.Resource.URN(), err)
		}
		if len(diffs) > 0 {
			result[op.Resource.URN()] = differ.ResourceDiff{URN: op.Resource.URN(), Diffs: diffs, SecretOnly: secretOnly}
		}
	}
	return result, nil
}

// CheckPlan refuses a destination linked to a transformation the same plan
// creates. A new transformation is published only once the apply finishes, and
// the API links a destination only to a published one, so the link would fail
// partway through every apply. Drop this check once transformations are
// published before their dependents run. See provider.PlanChecker.
func (p *Provider) CheckPlan(_ context.Context, plan *planner.Plan, _ *state.State) error {
	newTransformations := make(map[string]bool)
	for _, op := range plan.Operations {
		if op.Type == planner.Create && op.Resource.Type() == ttypes.TransformationResourceType {
			newTransformations[op.Resource.URN()] = true
		}
	}

	var refused []string
	for _, op := range plan.Operations {
		dest, ok := op.Resource.RawData().(*DestinationResource)
		if !ok || dest.Transformation == nil || !newTransformations[dest.Transformation.URN] {
			continue
		}
		refused = append(refused, fmt.Sprintf("%s links #%s", op.Resource.URN(), dest.Transformation.URN))
	}
	if len(refused) == 0 {
		return nil
	}
	return fmt.Errorf("cannot link a destination to a transformation created in the same apply: "+
		"a new transformation is published only once the apply finishes, and only a published "+
		"transformation can be linked. Apply the new transformation first without the link (leave a new "+
		"destination's transformation unset, and keep an existing destination's current one), then link "+
		"it in a second apply:\n  %s", strings.Join(refused, "\n  "))
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
