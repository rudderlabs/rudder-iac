// Package rules implements validation rules for the import-manifest spec kind.
// Rules are registered through the ImportManifestProvider and evaluated by the
// shared rules engine alongside resource-provider rules.
package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Syntactic returns the syntactic rules contributed by the import-manifest
// provider. They run before resource graph construction.
func Syntactic() []rules.Rule {
	return []rules.Rule{
		newSpecShapeRule(),
		newURNUniqueRule(),
		newInlineClashRule(),
	}
}

// --- spec-shape ---

// specShapeRule validates the structural shape of `spec.workspaces` on each
// import-manifest file: non-empty workspaces, workspace_id present, urn and
// remote_id present per resource, no duplicate URNs within a single file.
type specShapeRule struct{}

func newSpecShapeRule() rules.Rule { return &specShapeRule{} }

func (r *specShapeRule) ID() string               { return "import-manifest/spec-shape" }
func (r *specShapeRule) Severity() rules.Severity { return rules.Error }
func (r *specShapeRule) Description() string {
	return "import-manifest spec must declare workspaces with workspace_id and urn/remote_id resource entries"
}
func (r *specShapeRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1),
	}
}
func (r *specShapeRule) Examples() rules.Examples { return rules.Examples{} }

func (r *specShapeRule) Validate(ctx *rules.ValidationContext) []rules.ValidationResult {
	workspaces, ok := ctx.Spec["workspaces"].([]any)
	if !ok || len(workspaces) == 0 {
		return []rules.ValidationResult{{
			Reference: "/spec/workspaces",
			Message:   "'spec.workspaces' must be a non-empty list",
		}}
	}

	var results []rules.ValidationResult
	for i, raw := range workspaces {
		ws, ok := raw.(map[string]any)
		if !ok {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/spec/workspaces/%d", i),
				Message:   "workspace entry must be an object",
			})
			continue
		}

		if wsID, _ := ws["workspace_id"].(string); wsID == "" {
			results = append(results, rules.ValidationResult{
				Reference: fmt.Sprintf("/spec/workspaces/%d/workspace_id", i),
				Message:   "'workspace_id' is required",
			})
		}

		resourcesRaw, _ := ws["resources"].([]any)
		seenURNs := make(map[string]int, len(resourcesRaw))
		for j, resRaw := range resourcesRaw {
			res, ok := resRaw.(map[string]any)
			if !ok {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d", i, j),
					Message:   "resource entry must be an object",
				})
				continue
			}
			urn, _ := res["urn"].(string)
			remoteID, _ := res["remote_id"].(string)
			if urn == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", i, j),
					Message:   "'urn' is required",
				})
			}
			if remoteID == "" {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d/remote_id", i, j),
					Message:   "'remote_id' is required",
				})
			}
			if urn == "" {
				continue
			}
			if prev, dup := seenURNs[urn]; dup {
				results = append(results, rules.ValidationResult{
					Reference: fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", i, j),
					Message: fmt.Sprintf(
						"duplicate URN %q (first seen at workspace %d resource %d)",
						urn, i, prev,
					),
				})
			} else {
				seenURNs[urn] = j
			}
		}
	}

	return results
}

// --- urn-unique (cross-file) ---

// urnUniqueRule ensures URNs are unique across every import-manifest file in
// the project. It is a ProjectRule, so it runs once with all specs in scope.
type urnUniqueRule struct{}

func newURNUniqueRule() rules.Rule { return &urnUniqueRule{} }

func (r *urnUniqueRule) ID() string               { return "import-manifest/urn-unique" }
func (r *urnUniqueRule) Severity() rules.Severity { return rules.Error }
func (r *urnUniqueRule) Description() string {
	return "a URN must appear in at most one import-manifest entry across the project"
}
func (r *urnUniqueRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{
		rules.MatchKindVersion(specs.KindImportManifest, specs.SpecVersionV1),
	}
}
func (r *urnUniqueRule) Examples() rules.Examples                            { return rules.Examples{} }
func (r *urnUniqueRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult { return nil }

type urnLocation struct {
	filePath string
	ref      string
}

func (r *urnUniqueRule) ValidateProject(
	allSpecs map[string]*rules.ValidationContext,
) map[string][]rules.ValidationResult {
	occurrences := make(map[string][]urnLocation)

	for filePath, ctx := range allSpecs {
		if ctx.Kind != specs.KindImportManifest {
			continue
		}
		for wsIdx, urn := range collectManifestURNs(ctx.Spec) {
			occurrences[urn.value] = append(occurrences[urn.value], urnLocation{
				filePath: filePath,
				ref: fmt.Sprintf(
					"/spec/workspaces/%d/resources/%d/urn",
					urn.workspaceIdx, urn.resourceIdx,
				),
			})
			_ = wsIdx
		}
	}

	results := make(map[string][]rules.ValidationResult)
	for urn, locs := range occurrences {
		if len(locs) <= 1 {
			continue
		}
		for _, loc := range locs {
			results[loc.filePath] = append(results[loc.filePath], rules.ValidationResult{
				Reference: loc.ref,
				Message:   fmt.Sprintf("URN %q appears in multiple import-manifest entries", urn),
			})
		}
	}
	return results
}

// --- inline-clash (manifest vs inline metadata.import) ---

// inlineClashRule flags URNs that appear both in a manifest file and in a
// resource spec's inline metadata.import block. Surface area for migration;
// severity is an error so operators clean up one side.
type inlineClashRule struct{}

func newInlineClashRule() rules.Rule { return &inlineClashRule{} }

func (r *inlineClashRule) ID() string               { return "import-manifest/inline-clash" }
func (r *inlineClashRule) Severity() rules.Severity { return rules.Error }
func (r *inlineClashRule) Description() string {
	return "a URN must not appear in both an import-manifest file and a resource spec's inline metadata.import"
}
func (r *inlineClashRule) AppliesTo() []rules.MatchPattern {
	return []rules.MatchPattern{rules.MatchAll()}
}
func (r *inlineClashRule) Examples() rules.Examples                            { return rules.Examples{} }
func (r *inlineClashRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult { return nil }

func (r *inlineClashRule) ValidateProject(
	allSpecs map[string]*rules.ValidationContext,
) map[string][]rules.ValidationResult {
	// Collect manifest URNs → the one file that declares them (uniqueness is
	// enforced by urn-unique, so picking the first occurrence is fine).
	manifestURNs := make(map[string]string)
	for filePath, ctx := range allSpecs {
		if ctx.Kind != specs.KindImportManifest {
			continue
		}
		for _, urn := range collectManifestURNs(ctx.Spec) {
			if _, ok := manifestURNs[urn.value]; !ok {
				manifestURNs[urn.value] = filePath
			}
		}
	}
	if len(manifestURNs) == 0 {
		return nil
	}

	results := make(map[string][]rules.ValidationResult)
	for filePath, ctx := range allSpecs {
		if ctx.Kind == specs.KindImportManifest {
			continue
		}
		for _, entry := range collectInlineURNs(ctx.Metadata) {
			if _, clash := manifestURNs[entry.value]; !clash {
				continue
			}
			results[filePath] = append(results[filePath], rules.ValidationResult{
				Reference: fmt.Sprintf(
					"/metadata/import/workspaces/%d/resources/%d/urn",
					entry.workspaceIdx, entry.resourceIdx,
				),
				Message: fmt.Sprintf(
					"URN %q is declared in an import-manifest and must not also appear in inline metadata.import",
					entry.value,
				),
			})
		}
	}
	return results
}

// --- shared helpers ---

type indexedURN struct {
	value        string
	workspaceIdx int
	resourceIdx  int
}

// collectManifestURNs walks a manifest spec's spec.workspaces and returns
// every declared URN with its (workspace, resource) index pair so callers can
// cite a precise JSON Pointer reference.
func collectManifestURNs(spec map[string]any) []indexedURN {
	workspaces, _ := spec["workspaces"].([]any)
	var out []indexedURN
	for i, raw := range workspaces {
		ws, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		resources, _ := ws["resources"].([]any)
		for j, resRaw := range resources {
			res, ok := resRaw.(map[string]any)
			if !ok {
				continue
			}
			urn, _ := res["urn"].(string)
			if urn == "" {
				continue
			}
			out = append(out, indexedURN{value: urn, workspaceIdx: i, resourceIdx: j})
		}
	}
	return out
}

// collectInlineURNs walks a resource spec's metadata.import.workspaces and
// returns every declared URN with precise index pairs for error references.
func collectInlineURNs(metadata map[string]any) []indexedURN {
	importBlock, _ := metadata["import"].(map[string]any)
	if importBlock == nil {
		return nil
	}
	workspaces, _ := importBlock["workspaces"].([]any)
	var out []indexedURN
	for i, raw := range workspaces {
		ws, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		resources, _ := ws["resources"].([]any)
		for j, resRaw := range resources {
			res, ok := resRaw.(map[string]any)
			if !ok {
				continue
			}
			urn, _ := res["urn"].(string)
			if urn == "" {
				continue
			}
			out = append(out, indexedURN{value: urn, workspaceIdx: i, resourceIdx: j})
		}
	}
	return out
}
