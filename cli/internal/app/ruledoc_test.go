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
	// The gen-rule-docs make target defaults the import-manifest and rETL
	// flags on, so the kinds they register are documented while still
	// experimental. Match that here.
	t.Setenv("RUDDERSTACK_X_IMPORT_MERGE", "true")
	t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")
	// Hermetic config: defaults only, written under a temp dir so the suite
	// never touches the developer's ~/.rudder config.
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	// Import-manifest and rETL connection support are on so the providers
	// register the rules their embedded fragments document; the gen-rule-docs
	// workflows set the same flags for generation.
	prevExp := viper.Get("experimental")
	prevImportMerge := viper.Get("flags.importMerge")
	prevRetlConnections := viper.Get("flags.retlConnectionSupport")
	viper.Set("experimental", true)
	viper.Set("flags.importMerge", true)
	viper.Set("flags.retlConnectionSupport", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.importMerge", prevImportMerge)
		viper.Set("flags.retlConnectionSupport", prevRetlConnections)
	})

	doc, verrs, err := GenerateRuleCatalog("2026-01-01T00:00:00Z")
	require.NoError(t, err)
	assert.Empty(t, verrs, "every registered rule must have an authored fragment and vice versa")
	assert.NotEmpty(t, doc.Rules, "catalog should document at least one rule")

	ruleIDs := make(map[string]struct{}, len(doc.Rules))
	for _, rule := range doc.Rules {
		ruleIDs[rule.RuleID] = struct{}{}
	}
	for _, ruleID := range []string{
		"destination/semantic-valid",
		"destination/spec-syntax-valid",
		"event-stream/connection/enabled-endpoints-valid",
		"event-stream/connection/semantic-valid",
		"event-stream/connection/spec-syntax-valid",
		"import-manifest/duplicate-urn",
		"import-manifest/orphaned-urn",
		"import-manifest/spec-syntax-valid",
		"project/manifest-inline-conflict",
	} {
		assert.Contains(t, ruleIDs, ruleID)
	}
}
