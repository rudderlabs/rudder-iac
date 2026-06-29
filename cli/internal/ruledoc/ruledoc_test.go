package ruledoc_test

import (
	"testing"

	providerrules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/datacatalog/localcatalog"
	essource "github.com/rudderlabs/rudder-iac/cli/internal/providers/event-stream/source"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/sqlmodel"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/ruledoc"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatekeeperScopedPatterns mirrors the resource (kind, version) pairs declared in
// the project-level gatekeeper fragments (metadata-syntax-valid, duplicate-urn).
// The MockProvider returns these so BuildRegistry scopes those two rules to them,
// and the generated catalog matches the authored fragments exactly.
func gatekeeperScopedPatterns() []vrules.MatchPattern {
	var p []vrules.MatchPattern
	// Kinds carrying both legacy and v1 specs.
	for _, kind := range []string{
		localcatalog.KindCategories,
		localcatalog.KindCustomTypes,
		essource.ResourceKind,
		localcatalog.KindEvents,
		localcatalog.KindProperties,
		sqlmodel.ResourceKind,
	} {
		p = append(p, providerrules.LegacyVersionPatterns(kind)...)
		p = append(p, providerrules.V1VersionPatterns(kind)...)
	}
	// Tracking plans: legacy kind "tp", v1 kind "tracking-plan".
	p = append(p, providerrules.LegacyVersionPatterns(localcatalog.KindTrackingPlans)...)
	p = append(p, providerrules.V1VersionPatterns(localcatalog.KindTrackingPlansV1)...)
	// v1-only kinds.
	p = append(p, providerrules.V1VersionPatterns(ttypes.TransformationSpecKind)...)
	p = append(p, providerrules.V1VersionPatterns(ttypes.LibrarySpecKind)...)
	return p
}

// TestBuild_GatekeeperOnly proves Build assembles a valid catalog from a
// provider in isolation — no client, config, or auth. The provider declares the
// resource match patterns the gatekeeper rules are scoped to (via BuildRegistry),
// and Build always wires the import-manifest provider, so the four project-level
// gatekeeper rules plus the two import-manifest rules and their embedded fragments
// cover each other exactly and the catalog validates with zero errors. This is the
// assembly seam that package app exercises end-to-end against the real providers.
func TestBuild_GatekeeperOnly(t *testing.T) {
	cp := &testutils.MockProvider{MatchPatterns: gatekeeperScopedPatterns()}

	doc, verrs, err := ruledoc.Build(cp, "test", "2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Empty(t, verrs, "fragments must cover the registered rules exactly")

	ids := make([]string, 0, len(doc.Rules))
	for _, r := range doc.Rules {
		ids = append(ids, r.RuleID)
	}
	// The gatekeeper rules BuildRegistry always registers (incl. the cross-source
	// manifest-inline conflict rule), plus the import-manifest rules the manifest
	// provider contributes (two syntactic + the orphaned-urn semantic rule).
	assert.ElementsMatch(t, []string{
		"project/spec-syntax-valid",
		"project/resource-kind-version-valid",
		"project/metadata-syntax-valid",
		"project/duplicate-urn",
		"project/manifest-inline-conflict",
		"import-manifest/spec-syntax-valid",
		"import-manifest/duplicate-urn",
		"import-manifest/orphaned-urn",
	}, ids)
}
