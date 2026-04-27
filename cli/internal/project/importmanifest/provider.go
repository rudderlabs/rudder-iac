package importmanifest

import (
	"fmt"

	manifestrules "github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Provider owns the lifecycle of import-manifest specs: parses the payload,
// accumulates workspace entries across multiple manifest files, exposes the
// aggregated data for broadcast to resource handlers, and supplies validation
// rules that run in the same engine as resource-provider rules.
//
// It is parallel to — not nested under — CompositeProvider. The resource
// CompositeProvider never sees import-manifest specs.
type Provider struct {
	entries []specs.WorkspaceImportMetadata
}

// New returns a Provider with empty state. Callers should default-wire one
// inside project.New; there is no configuration to swap per caller.
func New() *Provider {
	return &Provider{}
}

// --- TypeProvider ---

func (p *Provider) SupportedKinds() []string {
	return []string{specs.KindImportManifest}
}

// SupportedTypes returns nil — manifests contribute no resource types.
func (p *Provider) SupportedTypes() []string {
	return nil
}

func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1),
	}
}

// --- SpecLoader ---

// LoadSpec parses the manifest's spec.workspaces payload and appends it to
// the aggregate. Cross-file URN collisions are caught upstream by the
// syntactic rule before this runs, so appending is always safe.
func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	workspaces, err := parseWorkspaces(s)
	if err != nil {
		return fmt.Errorf("parsing import-manifest %s: %w", path, err)
	}
	p.entries = append(p.entries, workspaces...)
	return nil
}

// LoadLegacySpec rejects — the manifest is a V1-only spec.
func (p *Provider) LoadLegacySpec(path string, s *specs.Spec) error {
	return fmt.Errorf("import-manifest does not support legacy spec versions")
}

// ParseSpec returns an empty ParsedSpec — the manifest contributes no URNs.
func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	return &specs.ParsedSpec{}, nil
}

// ResourceGraph returns an empty graph — the manifest contributes no nodes.
func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	return resources.NewGraph(), nil
}

// --- RuleProvider ---

func (p *Provider) SyntacticRules() []rules.Rule {
	return manifestrules.Syntactic()
}

func (p *Provider) SemanticRules() []rules.Rule {
	return manifestrules.Semantic()
}

// --- Accessor for broadcast ---

// ImportManifest returns the aggregated workspace entries wrapped in the
// shared WorkspacesImportMetadata type, or nil when no manifest specs were
// loaded. The orchestrator uses this to broadcast into the resource
// provider tree via ImportManifestConsumer.
func (p *Provider) ImportManifest() *specs.WorkspacesImportMetadata {
	if len(p.entries) == 0 {
		return nil
	}
	return &specs.WorkspacesImportMetadata{Workspaces: p.entries}
}
