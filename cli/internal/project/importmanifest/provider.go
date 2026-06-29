// Package importmanifest provides a project-level provider for import-manifest
// specs. It lives outside the CompositeProvider tree and contributes no resource
// graph nodes; it parses the manifest and exposes the aggregated entries for
// workspace-scoped broadcast to resource handlers.
package importmanifest

import (
	"fmt"

	manifestdocs "github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	mrules "github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// KindImportManifest is the spec kind this provider owns. It is a project-level
// kind handled outside the resource provider tree, so the provider that gives it
// meaning is its source of truth. The canonical value lives in the manifestspec
// leaf so the validation rules can reference it without importing this package.
const KindImportManifest = manifestspec.KindImportManifest

type Provider struct {
	entries []specs.WorkspaceImportMetadata
}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) SupportedKinds() []string {
	return []string{KindImportManifest}
}

func (p *Provider) SupportedTypes() []string {
	return nil
}

func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(KindImportManifest, specs.SpecVersionV1),
	}
}

// LoadSpec appends the spec's workspace entries to provider state. Appending is
// safe because cross-file URN collisions are caught by validation before this
// runs — no merge step needed.
func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	workspaces, err := parseWorkspaces(s)
	if err != nil {
		return fmt.Errorf("parsing import-manifest %s: %w", path, err)
	}
	p.entries = append(p.entries, workspaces...)
	return nil
}

func (p *Provider) LoadLegacySpec(path string, s *specs.Spec) error {
	return fmt.Errorf("import-manifest spec %s does not support legacy version %s", path, s.Version)
}

// ParseSpec extracts one URNEntry per manifest resource, each paired with the
// JSON-pointer path into spec.workspaces. The cross-source manifest-inline
// conflict rule consumes these to intersect manifest URNs with inline ones.
func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	workspaces, err := parseWorkspaces(s)
	if err != nil {
		return nil, err
	}
	urns := make([]specs.URNEntry, 0)
	for wi, ws := range workspaces {
		for ri, r := range ws.Resources {
			if r.URN == "" {
				continue
			}
			urns = append(urns, specs.URNEntry{
				URN:             r.URN,
				JSONPointerPath: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", wi, ri),
			})
		}
	}
	return &specs.ParsedSpec{URNs: urns}, nil
}

func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	return resources.NewGraph(), nil
}

func (p *Provider) SyntacticRules() []rules.Rule {
	return []rules.Rule{
		mrules.NewManifestSpecSyntaxValidRule(),
		mrules.NewManifestDuplicateURNRule(),
	}
}

func (p *Provider) SemanticRules() []rules.Rule {
	return []rules.Rule{
		mrules.NewOrphanedURNRule(),
	}
}

// RuleDocEntries returns the authored documentation fragments for the manifest
// rules, embedded alongside the provider.
func (p *Provider) RuleDocEntries() []docs.RuleDocEntry {
	entries, _ := docs.LoadRuleDocEntries(manifestdocs.FragmentsFS, ".")
	return entries
}

// ImportManifest returns the aggregated workspace entries, scoped to the
// active workspace, wrapped in the shared WorkspacesImportMetadata type.
//
// Returns nil when no manifest specs were loaded. An empty activeWorkspaceID
// means "all workspaces" (unscoped — used by validate); otherwise only the
// entries for the active workspace are returned (D13: at-most-one remote per
// URN reaches a handler). Returns nil when no entry matches the active
// workspace. The orchestrator broadcasts this into the resource provider tree
// via ImportManifestConsumer.
func (p *Provider) ImportManifest(activeWorkspaceID string) *specs.WorkspacesImportMetadata {
	if len(p.entries) == 0 {
		return nil
	}
	if activeWorkspaceID == "" {
		return &specs.WorkspacesImportMetadata{Workspaces: p.entries}
	}
	scoped := make([]specs.WorkspaceImportMetadata, 0, len(p.entries))
	for _, ws := range p.entries {
		if ws.WorkspaceID == activeWorkspaceID {
			scoped = append(scoped, ws)
		}
	}
	if len(scoped) == 0 {
		return nil
	}
	return &specs.WorkspacesImportMetadata{Workspaces: scoped}
}
