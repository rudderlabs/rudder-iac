// Package rules holds the validation rules owned by the import-manifest provider.
package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// manifestSpecValidRule is a per-file syntactic rule that validates one
// import-manifest's structure: workspaces present, each with a workspace_id, each
// resource importable via urn (local_id is rejected for manifests), and no
// duplicate urn within a single workspace block.
//
// It hand-rolls the Rule interface like its project-level peers (project/rules)
// rather than the resource-provider typed-rule framework, so it can strict-decode
// (reject unknown fields) and emit manifest-specific messages.
type manifestSpecValidRule struct{}

// NewManifestSpecSyntaxValidRule returns the per-file import-manifest structural rule.
func NewManifestSpecSyntaxValidRule() vrules.Rule {
	return &manifestSpecValidRule{}
}

func (r *manifestSpecValidRule) ID() string { return "import-manifest/spec-syntax-valid" }

func (r *manifestSpecValidRule) Severity() vrules.Severity { return vrules.Error }

func (r *manifestSpecValidRule) Description() string {
	return "import-manifest spec syntax must be valid"
}

func (r *manifestSpecValidRule) AppliesTo() []vrules.MatchPattern {
	return []vrules.MatchPattern{vrules.MatchKindVersion(manifestspec.KindImportManifest, specs.SpecVersionV1)}
}

func (r *manifestSpecValidRule) Examples() vrules.Examples { return vrules.Examples{} }

func (r *manifestSpecValidRule) Validate(ctx *vrules.ValidationContext) []vrules.ValidationResult {
	workspaces, err := manifestspec.DecodeWorkspaces(ctx.Spec)
	if err != nil {
		return []vrules.ValidationResult{{Reference: "/spec", Message: "spec.workspaces is malformed"}}
	}
	if len(workspaces) == 0 {
		return []vrules.ValidationResult{{Reference: "/spec/workspaces", Message: "manifest has no workspaces"}}
	}

	var results []vrules.ValidationResult
	for i, workspace := range workspaces {
		results = append(results, validateWorkspace(i, workspace)...)
	}
	return results
}

// validateWorkspace checks one workspace block: workspace_id present and each
// resource well-formed. URN uniqueness is the duplicate-urn rule's concern.
func validateWorkspace(i int, workspace specs.WorkspaceImportMetadata) []vrules.ValidationResult {
	var results []vrules.ValidationResult

	if workspace.WorkspaceID == "" {
		results = append(results, vrules.ValidationResult{
			Reference: fmt.Sprintf("/spec/workspaces/%d/workspace_id", i),
			Message:   "workspace_id is required",
		})
	}

	for j, resource := range workspace.Resources {
		results = append(results, validateResource(i, j, resource)...)
	}

	return results
}

// validateResource checks one resource is importable, reporting every problem:
// urn is required (local_id is not supported for manifests) and remote_id is
// required.
func validateResource(i, j int, resource specs.ImportIds) []vrules.ValidationResult {
	base := fmt.Sprintf("/spec/workspaces/%d/resources/%d", i, j)
	var results []vrules.ValidationResult

	if resource.URN == "" {
		results = append(results, vrules.ValidationResult{
			Reference: base + "/urn",
			Message:   "urn is required in manifests (local_id not supported)",
		})
	}
	if resource.RemoteID == "" {
		results = append(results, vrules.ValidationResult{
			Reference: base + "/remote_id",
			Message:   "remote_id is required",
		})
	}

	return results
}
