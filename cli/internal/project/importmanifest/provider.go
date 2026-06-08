// Package importmanifest provides a project-level provider for import-manifest
// specs. It lives outside the CompositeProvider tree and contributes no resource
// graph nodes; it parses the manifest and exposes the aggregated entries for
// workspace-scoped broadcast to resource handlers.
package importmanifest

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type Provider struct {
	entries []specs.WorkspaceImportMetadata
}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) SupportedKinds() []string {
	return []string{specs.KindImportManifest}
}

func (p *Provider) SupportedTypes() []string {
	return nil
}

func (p *Provider) SupportedMatchPatterns() []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1),
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

// ParseSpec returns an empty ParsedSpec — manifest specs carry no resource URNs.
func (p *Provider) ParseSpec(path string, s *specs.Spec) (*specs.ParsedSpec, error) {
	return &specs.ParsedSpec{}, nil
}

func (p *Provider) ResourceGraph() (*resources.Graph, error) {
	return resources.NewGraph(), nil
}

// Rule sets are nil until the manifest validation rules land in a later change.

func (p *Provider) SyntacticRules() []rules.Rule {
	return nil
}

func (p *Provider) SemanticRules() []rules.Rule {
	return nil
}

// ImportManifest returns a copy of the aggregated manifest, or nil when no
// manifest was loaded. The copy keeps callers from corrupting append-only state.
func (p *Provider) ImportManifest() *specs.WorkspacesImportMetadata {
	if len(p.entries) == 0 {
		return nil
	}
	workspaces := make([]specs.WorkspaceImportMetadata, len(p.entries))
	copy(workspaces, p.entries)
	return &specs.WorkspacesImportMetadata{Workspaces: workspaces}
}
