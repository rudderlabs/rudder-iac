package app

import (
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateRuleCatalog_CompleteAndDriftFree exercises the real provider
// composition and rule registry, then asserts the joined catalog passes
// validation with zero errors.
//
// This is the same drift/completeness gate the CI gen-rule-docs step enforces,
// expressed as a unit test: if a registered rule loses its authored
// *.docs.yaml fragment — or a fragment references a rule that no longer exists
// — verrs is non-empty and this fails locally, with no CI round-trip needed.
func TestGenerateRuleCatalog_CompleteAndDriftFree(t *testing.T) {
	Initialise("test")
	// Hermetic config: defaults only, written under a temp dir so the suite
	// never touches the developer's ~/.rudder config.
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	doc, verrs, err := GenerateRuleCatalog("2026-01-01T00:00:00Z", false)
	require.NoError(t, err)
	assert.Empty(t, verrs, "every registered rule must have an authored fragment and vice versa")
	assert.NotEmpty(t, doc.Rules, "catalog should document at least one rule")
}
