package rules

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest/manifestspec"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// manifestInlineConflictRule flags a URN declared in BOTH an import-manifest and
// a legacy inline metadata.import block. Inline metadata is the single-source
// legacy format being migrated away from, so any overlap with a manifest is a
// migration ambiguity — an operator must remove one. Reported at both locations.
//
// It is workspace-AGNOSTIC: it keys on URN alone (unlike the manifest
// duplicate-urn rule), because an overlap is a conflict regardless of which
// workspace each side names.
type manifestInlineConflictRule struct {
	manifestParseSpec ParseSpecFunc        // manifest URN extraction
	resourceParseSpec ParseSpecFunc        // resolves inline local_id via LegacyResourceType
	patterns          []rules.MatchPattern // union of manifest + resource patterns
}

// NewManifestInlineConflictRule needs both providers' info: the manifest
// ParseSpec (manifest URNs), the resource ParseSpec (to resolve inline local_id
// entries), and the active patterns (manifest + resource) it applies to.
func NewManifestInlineConflictRule(
	manifestParseSpec ParseSpecFunc,
	resourceParseSpec ParseSpecFunc,
	patterns []rules.MatchPattern,
) rules.Rule {
	return &manifestInlineConflictRule{
		manifestParseSpec: manifestParseSpec,
		resourceParseSpec: resourceParseSpec,
		patterns:          patterns,
	}
}

func (r *manifestInlineConflictRule) ID() string               { return "project/manifest-inline-conflict" }
func (r *manifestInlineConflictRule) Severity() rules.Severity { return rules.Error }
func (r *manifestInlineConflictRule) Description() string {
	return "a URN must not be defined in both an import-manifest and inline metadata.import"
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

type urnLoc struct {
	path    string
	pointer string
}

// ValidateSpecs partitions the contexts into manifest URN locations and inline
// metadata.import URN locations, then flags every URN present in both — at each
// of its locations on both sides.
func (r *manifestInlineConflictRule) ValidateSpecs(
	contexts map[string]*rules.ValidationContext,
) map[string][]rules.ValidationResult {
	manifestURNs := map[string][]urnLoc{}
	inlineURNs := map[string][]urnLoc{}

	for path, ctx := range contexts {
		if ctx.Kind == manifestspec.KindImportManifest {
			r.collectManifestURNs(path, ctx, manifestURNs)
			continue
		}
		r.collectInlineURNs(path, ctx, inlineURNs)
	}

	results := map[string][]rules.ValidationResult{}
	for urn, mLocs := range manifestURNs {
		iLocs, conflict := inlineURNs[urn]
		if !conflict {
			continue
		}
		msg := fmt.Sprintf("URN '%s' is defined in both an import-manifest and inline metadata; remove the inline metadata entry", urn)
		for _, l := range append(mLocs, iLocs...) {
			results[l.path] = append(results[l.path], rules.ValidationResult{Reference: l.pointer, Message: msg})
		}
	}
	return results
}

func (r *manifestInlineConflictRule) collectManifestURNs(path string, ctx *rules.ValidationContext, out map[string][]urnLoc) {
	parsed, err := r.manifestParseSpec(path, specFromContext(ctx))
	if err != nil {
		return // malformed manifest is the manifest spec-syntax-valid rule's concern
	}
	for _, e := range parsed.URNs {
		out[e.URN] = append(out[e.URN], urnLoc{path: path, pointer: e.JSONPointerPath})
	}
}

// collectInlineURNs reads a resource spec's inline metadata.import block,
// resolving each entry to a URN via its urn or local_id (the latter through the
// resource's LegacyResourceType), mirroring metadata-syntax-valid.
func (r *manifestInlineConflictRule) collectInlineURNs(path string, ctx *rules.ValidationContext, out map[string][]urnLoc) {
	metadata, err := specFromContext(ctx).CommonMetadata()
	if err != nil || metadata.Import == nil {
		return
	}

	var legacyResourceType string
	for wi, ws := range metadata.Import.Workspaces {
		for ri, res := range ws.Resources {
			urn := res.URN
			if urn == "" && res.LocalID != "" {
				if legacyResourceType == "" {
					parsed, perr := r.resourceParseSpec(path, specFromContext(ctx))
					if perr != nil || parsed.LegacyResourceType == "" {
						continue // local_id unsupported for this kind — metadata-syntax-valid reports it
					}
					legacyResourceType = parsed.LegacyResourceType
				}
				urn = resources.URN(res.LocalID, legacyResourceType)
			}
			if urn == "" {
				continue
			}
			out[urn] = append(out[urn], urnLoc{
				path:    path,
				pointer: fmt.Sprintf("/metadata/import/workspaces/%d/resources/%d/urn", wi, ri),
			})
		}
	}
}

func specFromContext(ctx *rules.ValidationContext) *specs.Spec {
	return &specs.Spec{Version: ctx.Version, Kind: ctx.Kind, Metadata: ctx.Metadata, Spec: ctx.Spec}
}
