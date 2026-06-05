package ruledoc_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/ruledoc"
	"github.com/rudderlabs/rudder-iac/cli/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuild_GatekeeperOnly proves Build assembles a valid catalog from a
// provider in isolation — no client, config, or auth. An empty provider still
// yields the four project-level gatekeeper rules (always registered by
// BuildRegistry), and their embedded fragments cover them exactly, so the
// catalog validates with zero errors. This is the assembly seam that package
// app exercises end-to-end against the real composed providers.
func TestBuild_GatekeeperOnly(t *testing.T) {
	cp := &testutils.MockProvider{}

	doc, verrs, err := ruledoc.Build(cp, "test", "2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Empty(t, verrs, "gatekeeper fragments must cover the gatekeeper rules exactly")

	// spec-syntax-valid, resource-kind-version-valid, metadata-syntax-valid,
	// duplicate-urn — the rules BuildRegistry always registers.
	require.Len(t, doc.Rules, 4)
	for _, r := range doc.Rules {
		require.Len(t, r.AppliesTo, 1)
		assert.Equal(t, "*", r.AppliesTo[0].Kind)
		assert.Equal(t, "*", r.AppliesTo[0].Version)
	}
}
