package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

type importManifestProjectRule struct{}

func NewImportManifestProjectRule() rules.Rule {
	return &importManifestProjectRule{}
}

func (r *importManifestProjectRule) ID() string               { return "project/import-manifest-cross-file" }
func (r *importManifestProjectRule) Severity() rules.Severity { return rules.Error }
func (r *importManifestProjectRule) Description() string {
	return "import manifest URNs must be unique across files and not conflict with inline metadata"
}
func (r *importManifestProjectRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{rules.MatchAll()}
}
func (r *importManifestProjectRule) Examples() rules.Examples { return rules.Examples{} }

// Validate returns nil -- cross-file logic is in ValidateProject
func (r *importManifestProjectRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

// ValidateProject implements ProjectRule for cross-file manifest checks
func (r *importManifestProjectRule) ValidateProject(
	allSpecs map[string]*rules.ValidationContext,
) map[string][]rules.ValidationResult {
	type urnLocation struct {
		FilePath  string
		Reference string
	}

	manifestURNs := make(map[string]urnLocation)
	results := make(map[string][]rules.ValidationResult)

	for filePath, ctx := range allSpecs {
		if ctx.Kind != specs.KindImportManifest {
			continue
		}

		for _, entry := range extractManifestURNs(ctx.Spec) {
			if prev, exists := manifestURNs[entry.URN]; exists {
				results[filePath] = append(results[filePath], rules.ValidationResult{
					RuleID:    r.ID(),
					Severity:  r.Severity(),
					Message:   fmt.Sprintf("URN '%s' defined in both %s and %s", entry.URN, prev.FilePath, filePath),
					FilePath:  filePath,
					FileName:  ctx.FileName,
					Reference: entry.Reference,
				})
			} else {
				manifestURNs[entry.URN] = urnLocation{FilePath: filePath, Reference: entry.Reference}
			}
		}
	}

	if len(manifestURNs) == 0 {
		return results
	}

	// Detect conflicts with inline metadata.import in resource specs
	for filePath, ctx := range allSpecs {
		if ctx.Kind == specs.KindImportManifest {
			continue
		}

		inlineURNs := extractInlineImportURNs(ctx.Metadata)
		for _, urn := range inlineURNs {
			if manifestLoc, exists := manifestURNs[urn]; exists {
				results[manifestLoc.FilePath] = append(results[manifestLoc.FilePath], rules.ValidationResult{
					RuleID:    r.ID(),
					Severity:  r.Severity(),
					Message:   fmt.Sprintf("URN '%s' found in both manifest %s and inline metadata in %s", urn, manifestLoc.FilePath, filePath),
					FilePath:  manifestLoc.FilePath,
					FileName:  manifestLoc.FilePath,
					Reference: manifestLoc.Reference,
				})
			}
		}
	}

	return results
}

type manifestURNEntry struct {
	URN       string
	Reference string
}

func extractManifestURNs(specMap map[string]any) []manifestURNEntry {
	var entries []manifestURNEntry

	workspacesRaw, ok := specMap["workspaces"]
	if !ok {
		return nil
	}

	workspaces, ok := workspacesRaw.([]any)
	if !ok {
		return nil
	}

	for i, wsRaw := range workspaces {
		ws, ok := wsRaw.(map[string]any)
		if !ok {
			continue
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
			if urn != "" {
				entries = append(entries, manifestURNEntry{
					URN:       urn,
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", i, j),
				})
			}
		}
	}

	return entries
}

func extractInlineImportURNs(metadata map[string]any) []string {
	var urns []string

	importRaw, ok := metadata["import"]
	if !ok {
		return nil
	}
	importMap, ok := importRaw.(map[string]any)
	if !ok {
		return nil
	}
	workspacesRaw, ok := importMap["workspaces"]
	if !ok {
		return nil
	}
	workspaces, ok := workspacesRaw.([]any)
	if !ok {
		return nil
	}

	for _, wsRaw := range workspaces {
		ws, ok := wsRaw.(map[string]any)
		if !ok {
			continue
		}
		resourcesRaw, ok := ws["resources"]
		if !ok {
			continue
		}
		resources, ok := resourcesRaw.([]any)
		if !ok {
			continue
		}
		for _, resRaw := range resources {
			res, ok := resRaw.(map[string]any)
			if !ok {
				continue
			}
			if urn, ok := res["urn"].(string); ok && urn != "" {
				urns = append(urns, urn)
			}
		}
	}

	return urns
}
