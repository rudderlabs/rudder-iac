package rules

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type importManifestSyntaxRule struct{}

func NewImportManifestSyntaxRule() rules.Rule {
	return &importManifestSyntaxRule{}
}

func (r *importManifestSyntaxRule) ID() string               { return "project/import-manifest-syntax" }
func (r *importManifestSyntaxRule) Severity() rules.Severity { return rules.Error }
func (r *importManifestSyntaxRule) Description() string {
	return "import manifest syntax must be valid"
}
func (r *importManifestSyntaxRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{rules.MatchKind(specs.KindImportManifest)}
}
func (r *importManifestSyntaxRule) Examples() rules.Examples { return rules.Examples{} }

func (r *importManifestSyntaxRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	var results []rules.ValidationResult

	workspacesRaw, ok := ctx.Spec["workspaces"]
	if !ok {
		results = append(results, rules.ValidationResult{
			RuleID:    r.ID(),
			Severity:  r.Severity(),
			Message:   "manifest must contain 'workspaces' field",
			FilePath:  ctx.FilePath,
			FileName:  ctx.FileName,
			Reference: "/spec/workspaces",
		})
		return results
	}

	workspaces, ok := workspacesRaw.([]any)
	if !ok || len(workspaces) == 0 {
		results = append(results, rules.ValidationResult{
			RuleID:    r.ID(),
			Severity:  r.Severity(),
			Message:   "manifest must contain at least one workspace",
			FilePath:  ctx.FilePath,
			FileName:  ctx.FileName,
			Reference: "/spec/workspaces",
		})
		return results
	}

	if err := validateManifestPayloadStrict(ctx.Spec); err != nil {
		results = append(results, rules.ValidationResult{
			RuleID:    r.ID(),
			Severity:  r.Severity(),
			Message:   fmt.Sprintf("manifest contains unknown or invalid fields: %s", err.Error()),
			FilePath:  ctx.FilePath,
			FileName:  ctx.FileName,
			Reference: "/spec",
		})
		return results
	}

	seenURNs := make(map[string]string) // urn -> reference path for first occurrence

	for i, wsRaw := range workspaces {
		ws, ok := wsRaw.(map[string]any)
		if !ok {
			results = append(results, rules.ValidationResult{
				RuleID:    r.ID(),
				Severity:  r.Severity(),
				Message:   fmt.Sprintf("workspace at index %d must be a map", i),
				FilePath:  ctx.FilePath,
				FileName:  ctx.FileName,
				Reference: fmt.Sprintf("/spec/workspaces/%d", i),
			})
			continue
		}

		wsID, _ := ws["workspace_id"].(string)
		if wsID == "" {
			results = append(results, rules.ValidationResult{
				RuleID:    r.ID(),
				Severity:  r.Severity(),
				Message:   fmt.Sprintf("missing required field 'workspace_id' in workspace at index %d", i),
				FilePath:  ctx.FilePath,
				FileName:  ctx.FileName,
				Reference: fmt.Sprintf("/spec/workspaces/%d/workspace_id", i),
			})
		}

		resourcesRaw, ok := ws["resources"]
		if !ok {
			continue
		}
		resources, ok := resourcesRaw.([]any)
		if !ok {
			continue
		}

		for j, resRaw := range resources {
			res, ok := resRaw.(map[string]any)
			if !ok {
				continue
			}

			urn, _ := res["urn"].(string)
			remoteID, _ := res["remote_id"].(string)

			entry := specs.ImportIds{
				URN:      urn,
				RemoteID: remoteID,
			}
			if err := entry.Validate(); err != nil {
				results = append(results, rules.ValidationResult{
					RuleID:    r.ID(),
					Severity:  r.Severity(),
					Message:   fmt.Sprintf("invalid import resource in workspace '%s': %s", wsID, err.Error()),
					FilePath:  ctx.FilePath,
					FileName:  ctx.FileName,
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d", i, j),
				})
			}

			if urn != "" {
				ref := fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", i, j)
				if _, exists := seenURNs[urn]; exists {
					results = append(results, rules.ValidationResult{
						RuleID:    r.ID(),
						Severity:  r.Severity(),
						Message:   fmt.Sprintf("duplicate URN '%s' within manifest", urn),
						FilePath:  ctx.FilePath,
						FileName:  ctx.FileName,
						Reference: ref,
					})
				} else {
					seenURNs[urn] = ref
				}
			}
		}
	}

	return results
}

func validateManifestPayloadStrict(specMap map[string]any) error {
	raw, err := yaml.Marshal(specMap)
	if err != nil {
		return fmt.Errorf("marshalling spec: %w", err)
	}
	var payload struct {
		Workspaces []specs.WorkspaceImportMetadata `yaml:"workspaces"`
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	return decoder.Decode(&payload)
}
