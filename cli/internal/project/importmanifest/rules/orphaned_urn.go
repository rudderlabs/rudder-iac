package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// orphanedURNRule is a semantic rule that flags a manifest URN with no matching
// resource in the built graph — a stale entry or a typo. It runs post-graph
// (ctx.Graph populated) and is scoped to the active workspace: only the URNs the
// targeted workspace imports must resolve to a local resource. URNs under other
// workspaces (e.g. a different environment's block) are not in play for the run
// and are left to that environment's own apply.
type orphanedURNRule struct{}

// NewOrphanedURNRule returns the post-graph orphaned-URN rule.
func NewOrphanedURNRule() vrules.Rule {
	return &orphanedURNRule{}
}

func (r *orphanedURNRule) ID() string { return "import-manifest/orphaned-urn" }

func (r *orphanedURNRule) Severity() vrules.Severity { return vrules.Error }

func (r *orphanedURNRule) Description() string {
	return "Every import-manifest URN must match a resource defined in the project"
}

func (r *orphanedURNRule) AppliesTo() []vrules.MatchPattern {
	return []vrules.MatchPattern{vrules.MatchKindVersion(manifestspec.KindImportManifest, specs.SpecVersionV1)}
}

func (r *orphanedURNRule) Examples() vrules.Examples { return vrules.Examples{} }

func (r *orphanedURNRule) Validate(ctx *vrules.ValidationContext) []vrules.ValidationResult {
	if !ctx.HasGraph() {
		return nil
	}
	workspaces, err := manifestspec.DecodeWorkspaces(ctx.Spec)
	if err != nil {
		return nil // malformed shape is the spec-syntax-valid rule's concern
	}

	var results []vrules.ValidationResult
	for wi, ws := range workspaces {
		// Only the active workspace's entries are in play for this run. An empty
		// WorkspaceID (validate with no target) checks every workspace.
		if ctx.WorkspaceID != "" && ws.WorkspaceID != ctx.WorkspaceID {
			continue
		}
		for ri, res := range ws.Resources {
			if res.URN == "" {
				continue
			}
			if _, ok := ctx.Graph.GetResource(res.URN); !ok {
				results = append(results, vrules.ValidationResult{
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", wi, ri),
					Message:   fmt.Sprintf("manifest URN '%s' does not match any resource in the project", res.URN),
				})
			}
		}
	}
	return results
}
