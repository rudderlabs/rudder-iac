package ruledoc_test

import (
	"testing"

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
var gatekeeperScopedPatterns = []vrules.MatchPattern{
	{Kind: "categories", Version: "rudder/0.1"},
	{Kind: "categories", Version: "rudder/v0.1"},
	{Kind: "categories", Version: "rudder/v1"},
	{Kind: "custom-types", Version: "rudder/0.1"},
	{Kind: "custom-types", Version: "rudder/v0.1"},
	{Kind: "custom-types", Version: "rudder/v1"},
	{Kind: "event-stream-source", Version: "rudder/0.1"},
	{Kind: "event-stream-source", Version: "rudder/v0.1"},
	{Kind: "event-stream-source", Version: "rudder/v1"},
	{Kind: "events", Version: "rudder/0.1"},
	{Kind: "events", Version: "rudder/v0.1"},
	{Kind: "events", Version: "rudder/v1"},
	{Kind: "properties", Version: "rudder/0.1"},
	{Kind: "properties", Version: "rudder/v0.1"},
	{Kind: "properties", Version: "rudder/v1"},
	{Kind: "retl-source-sql-model", Version: "rudder/0.1"},
	{Kind: "retl-source-sql-model", Version: "rudder/v0.1"},
	{Kind: "retl-source-sql-model", Version: "rudder/v1"},
	{Kind: "tp", Version: "rudder/0.1"},
	{Kind: "tp", Version: "rudder/v0.1"},
	{Kind: "tracking-plan", Version: "rudder/v1"},
	{Kind: "transformation", Version: "rudder/v1"},
	{Kind: "transformation-library", Version: "rudder/v1"},
}

// TestBuild_GatekeeperOnly proves Build assembles a valid catalog from a
// provider in isolation — no client, config, or auth. The provider declares the
// resource match patterns the gatekeeper rules are scoped to (via BuildRegistry),
// so the four project-level gatekeeper rules and their embedded fragments cover
// each other exactly and the catalog validates with zero errors. This is the
// assembly seam that package app exercises end-to-end against the real providers.
func TestBuild_GatekeeperOnly(t *testing.T) {
	cp := &testutils.MockProvider{MatchPatterns: gatekeeperScopedPatterns}

	doc, verrs, err := ruledoc.Build(cp, "test", "2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Empty(t, verrs, "gatekeeper fragments must cover the gatekeeper rules exactly")

	// spec-syntax-valid, resource-kind-version-valid, metadata-syntax-valid,
	// duplicate-urn — the rules BuildRegistry always registers.
	require.Len(t, doc.Rules, 4)
}
