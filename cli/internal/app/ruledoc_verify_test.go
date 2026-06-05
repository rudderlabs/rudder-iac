package app

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateRuleCatalog_VerifiesCleanly drives the real composite provider
// (no network) and runs the executable verifier over every authored syntactic
// example. Unlike the structural drift test, this asserts the authored
// examples actually behave as documented when run through the real validation
// engine: every invalid example produces its expected diagnostics and every
// valid example produces none.
//
// The DataGraph experimental flag is off by default, so the datagraph provider
// and its fragments are excluded here — that is expected and consistent with
// the rest of the catalog gate.
func TestGenerateRuleCatalog_VerifiesCleanly(t *testing.T) {
	Initialise("0.0.0-test")
	config.InitConfig(config.DefaultConfigFile())

	doc, verrs, err := GenerateRuleCatalog("2026-01-01T00:00:00Z", false)
	require.NoError(t, err)
	assert.Empty(t, verrs, "every authored example must verify cleanly through the real engine")
	assert.NotEmpty(t, doc.Rules, "catalog should document at least one rule")
}
