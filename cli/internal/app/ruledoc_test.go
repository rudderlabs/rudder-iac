package app

import (
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/viper"
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
	// The gatekeeper fragments document the kinds registered with experimental
	// flags off, which is how CI generates the catalog; an experimental kind
	// such as retl-source-table joins them when it goes GA, as data-graph did.
	// Pin its flag off so a developer's environment cannot fail this test.
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "false")
	// Hermetic config: defaults only, written under a temp dir so the suite
	// never touches the developer's ~/.rudder config.
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	prevExp := viper.Get("experimental")
	viper.Set("experimental", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
	})

	doc, verrs, err := GenerateRuleCatalog("2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Empty(t, verrs, "every registered rule must have an authored fragment and vice versa")
	assert.NotEmpty(t, doc.Rules, "catalog should document at least one rule")
}
