package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// manifestDuplicateURNRule is a MultiSpecRule that flags a (workspace_id, urn)
// pair appearing more than once across all import-manifest files — whether the
// duplicates are in the same file or different files. The same urn under
// different workspaces is legal (it maps to a different remote per workspace).
type manifestDuplicateURNRule struct{}

// NewManifestDuplicateURNRule returns the manifest duplicate-urn rule.
func NewManifestDuplicateURNRule() vrules.Rule {
	return &manifestDuplicateURNRule{}
}

func (r *manifestDuplicateURNRule) ID() string { return "import-manifest/duplicate-urn" }

func (r *manifestDuplicateURNRule) Severity() vrules.Severity { return vrules.Error }

func (r *manifestDuplicateURNRule) Description() string {
	return "a (workspace_id, urn) must not be defined more than once across import-manifest files"
}

func (r *manifestDuplicateURNRule) AppliesTo() []vrules.MatchPattern {
	return []vrules.MatchPattern{
		vrules.MatchKindVersion(manifestspec.KindImportManifest, specs.SpecVersionV1),
	}
}

func (r *manifestDuplicateURNRule) Examples() vrules.Examples { return vrules.Examples{} }

// Validate is a no-op — this rule operates as a MultiSpecRule via ValidateSpecs.
func (r *manifestDuplicateURNRule) Validate(_ *vrules.ValidationContext) []vrules.ValidationResult {
	return nil
}

// wsURN keys uniqueness by workspace and urn together: the same urn under two
// different workspaces is legal, so only a repeated (workspace_id, urn) collides.
type wsURN struct {
	workspaceID string
	urn         string
}

// ValidateSpecs receives only import-manifest contexts (engine pre-filtered by
// AppliesTo) and flags every resource whose (workspace_id, urn) is defined more
// than once across them. Counting then flagging surfaces all occurrences, and
// makes same-file and cross-file duplicates the same case.
func (r *manifestDuplicateURNRule) ValidateSpecs(
	manifests map[string]*vrules.ValidationContext,
) map[string][]vrules.ValidationResult {
	counts := countURNs(manifests)

	results := map[string][]vrules.ValidationResult{}
	for path, ctx := range manifests {
		forEachResourceURN(ctx, func(i, j int, key wsURN) {
			if counts[key] > 1 {
				results[path] = append(results[path], vrules.ValidationResult{
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", i, j),
					Message:   fmt.Sprintf("duplicate URN '%s' in workspace '%s'", key.urn, key.workspaceID),
				})
			}
		})
	}
	return results
}

func countURNs(manifests map[string]*vrules.ValidationContext) map[wsURN]int {
	counts := map[wsURN]int{}
	for _, ctx := range manifests {
		forEachResourceURN(ctx, func(_, _ int, key wsURN) {
			counts[key]++
		})
	}
	return counts
}

// forEachResourceURN walks one manifest's resources, invoking fn for each with a
// non-empty urn under an identified workspace. Malformed specs, unidentified
// workspaces, and urn-less resources are skipped — the spec-syntax-valid rule
// reports those.
func forEachResourceURN(ctx *vrules.ValidationContext, fn func(i, j int, key wsURN)) {
	workspaces, err := manifestspec.DecodeWorkspaces(ctx.Spec)
	if err != nil {
		return
	}
	for i, workspace := range workspaces {
		if workspace.WorkspaceID == "" {
			continue
		}
		for j, resource := range workspace.Resources {
			if resource.URN == "" {
				continue
			}
			fn(i, j, wsURN{workspaceID: workspace.WorkspaceID, urn: resource.URN})
		}
	}
}
