package rules

import (
	"fmt"

	"github.com/samber/lo"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// manifestInlineConflictRule flags a (workspace_id, urn) declared in BOTH an
// import-manifest and a legacy inline metadata.import block with DIFFERING
// remote_ids. Inline metadata is the single-source legacy format being migrated
// away from, so an overlap that disagrees on the remote target is a migration
// ambiguity — an operator must remove one. Reported at both locations.
//
// It is workspace-SCOPED (like the manifest duplicate-urn rule): the same urn
// under different workspaces maps to a different remote per workspace, so it is
// legal. An overlap that agrees on remote_id names the same import target and is
// harmless, so it is not flagged either.
type manifestInlineConflictRule struct {
	resourceParseSpec ParseSpecFunc        // resolves inline local_id via LegacyResourceType
	patterns          []rules.MatchPattern // union of manifest + resource patterns
}

// NewManifestInlineConflictRule needs the resource ParseSpec (to resolve inline
// local_id entries) and the active patterns (manifest + resource) it applies to.
func NewManifestInlineConflictRule(
	resourceParseSpec ParseSpecFunc,
	patterns []rules.MatchPattern,
) rules.Rule {
	return &manifestInlineConflictRule{
		resourceParseSpec: resourceParseSpec,
		patterns:          patterns,
	}
}

func (r *manifestInlineConflictRule) ID() string               { return "project/manifest-inline-conflict" }
func (r *manifestInlineConflictRule) Severity() rules.Severity { return rules.Error }
func (r *manifestInlineConflictRule) Description() string {
	return "A (workspace_id, urn) must not be defined in both an import-manifest and inline metadata.import with differing remote_ids"
}

// AppliesTo is the union of manifest + resource patterns, so the engine delivers
// both manifest and resource contexts to ValidateSpecs.
func (r *manifestInlineConflictRule) AppliesTo() []rules.MatchPattern {
	return r.patterns
}

func (r *manifestInlineConflictRule) Examples() rules.Examples { return rules.Examples{} }

// Validate is a no-op — this rule operates as a MultiSpecRule via ValidateSpecs.
func (r *manifestInlineConflictRule) Validate(_ *rules.ValidationContext) []rules.ValidationResult {
	return nil
}

// wsURN keys an overlap by workspace and urn together: the same urn under two
// different workspaces maps to a different remote per workspace, so only a
// shared (workspace_id, urn) can collide.
type wsURN struct {
	workspaceID string
	urn         string
}

type urnLoc struct {
	path     string
	pointer  string
	remoteID string
}

// ValidateSpecs partitions the contexts into manifest and inline metadata.import
// (workspace_id, urn) locations, then flags every key present in both whose
// remote_ids disagree — at each of its locations on both sides. A key present in
// both that agrees on remote_id names the same import target and is left clean.
func (r *manifestInlineConflictRule) ValidateSpecs(
	contexts map[string]*rules.ValidationContext,
) map[string][]rules.ValidationResult {
	manifestURNs := map[wsURN][]urnLoc{}
	inlineURNs := map[wsURN][]urnLoc{}

	for path, ctx := range contexts {
		if ctx.Kind == manifestspec.KindImportManifest {
			r.collectManifestURNs(ctx, manifestURNs)
			continue
		}
		r.collectInlineURNs(path, ctx, inlineURNs)
	}

	results := map[string][]rules.ValidationResult{}
	for key, manifestLocs := range manifestURNs {
		inlineLocs, overlap := inlineURNs[key]
		if !overlap {
			continue
		}

		allLocs := lo.Flatten([][]urnLoc{manifestLocs, inlineLocs})
		if sameRemoteID(allLocs) {
			continue
		}
		msg := fmt.Sprintf("URN '%s' in workspace '%s' is defined in both an import-manifest and inline metadata with differing remote_ids; remove the inline metadata entry", key.urn, key.workspaceID)
		for _, loc := range allLocs {
			results[loc.path] = append(results[loc.path], rules.ValidationResult{Reference: loc.pointer, Message: msg})
		}
	}
	return results
}

// sameRemoteID reports whether every location shares one remote_id — the "same
// import target" case that is not a conflict. Empty input is not a match.
func sameRemoteID(locs []urnLoc) bool {
	if len(locs) == 0 {
		return false
	}
	sharedRemoteID := locs[0].remoteID
	return lo.EveryBy(locs, func(loc urnLoc) bool { return loc.remoteID == sharedRemoteID })
}

func (r *manifestInlineConflictRule) collectManifestURNs(ctx *rules.ValidationContext, out map[wsURN][]urnLoc) {
	workspaces, err := manifestspec.DecodeWorkspaces(ctx.Spec)
	if err != nil {
		return // malformed manifest is the manifest spec-syntax-valid rule's concern
	}
	for i, workspace := range workspaces {
		if workspace.WorkspaceID == "" {
			continue // unidentified workspace — the spec-syntax-valid rule reports it
		}
		for j, resource := range workspace.Resources {
			if resource.URN == "" {
				continue
			}
			key := wsURN{workspaceID: workspace.WorkspaceID, urn: resource.URN}
			out[key] = append(out[key], urnLoc{
				path:     ctx.FilePath,
				pointer:  fmt.Sprintf("/spec/workspaces/%d/resources/%d/urn", i, j),
				remoteID: resource.RemoteID,
			})
		}
	}
}

// collectInlineURNs reads a resource spec's inline metadata.import block,
// resolving each entry to a URN via its urn or local_id (the latter through the
// resource's LegacyResourceType), mirroring metadata-syntax-valid.
func (r *manifestInlineConflictRule) collectInlineURNs(path string, ctx *rules.ValidationContext, out map[wsURN][]urnLoc) {
	metadata, err := specFromContext(ctx).CommonMetadata()
	if err != nil || metadata.Import == nil {
		return
	}

	var legacyResourceType string
	for i, workspace := range metadata.Import.Workspaces {
		if workspace.WorkspaceID == "" {
			continue // unidentified workspace — metadata-syntax-valid reports it
		}
		for j, resource := range workspace.Resources {
			urn := resource.URN
			if urn == "" && resource.LocalID != "" {
				if legacyResourceType == "" {
					parsed, perr := r.resourceParseSpec(path, specFromContext(ctx))
					if perr != nil || parsed.LegacyResourceType == "" {
						continue // local_id unsupported for this kind — metadata-syntax-valid reports it
					}
					legacyResourceType = parsed.LegacyResourceType
				}
				urn = resources.URN(resource.LocalID, legacyResourceType)
			}
			if urn == "" {
				continue
			}
			key := wsURN{workspaceID: workspace.WorkspaceID, urn: urn}
			out[key] = append(out[key], urnLoc{
				path:     path,
				pointer:  fmt.Sprintf("/metadata/import/workspaces/%d/resources/%d/urn", i, j),
				remoteID: resource.RemoteID,
			})
		}
	}
}

func specFromContext(ctx *rules.ValidationContext) *specs.Spec {
	return &specs.Spec{Version: ctx.Version, Kind: ctx.Kind, Metadata: ctx.Metadata, Spec: ctx.Spec}
}
