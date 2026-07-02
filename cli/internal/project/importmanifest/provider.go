// Package importmanifest provides a project-level provider for import-manifest
// specs. It lives outside the CompositeProvider tree and contributes no resource
// graph nodes; it parses the manifest and exposes the aggregated entries for
// workspace-scoped broadcast to resource handlers.
package importmanifest

import (
	"fmt"

	"github.com/samber/lo"

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
	// entries holds one merged entry per workspace, unifying a workspace split
	// across multiple manifest files.
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

// LoadSpec merges the spec's workspace entries into provider state, keeping one
// entry per workspace so a workspace split across multiple manifest files is
// unified. Validation guarantees no duplicate (workspace_id, urn), so appending
// resources needs no deduplication.
func (p *Provider) LoadSpec(path string, s *specs.Spec) error {
	workspaces, err := parseWorkspaces(s)
	if err != nil {
		return fmt.Errorf("parsing import-manifest %s: %w", path, err)
	}

	for _, ws := range workspaces {
		_, i, found := lo.FindIndexOf(p.entries, func(e specs.WorkspaceImportMetadata) bool {
			return e.WorkspaceID == ws.WorkspaceID
		})

		if found {
			p.entries[i].Resources = append(p.entries[i].Resources, ws.Resources...)
			continue
		}
		p.entries = append(p.entries, ws)
	}
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

// ImportManifest returns the merged workspace entries, one per workspace. The
// slice is empty when no manifest specs were loaded.
func (p *Provider) ImportManifest() []specs.WorkspaceImportMetadata {
	return p.entries
}
