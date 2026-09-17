package docs

import (
	"bytes"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// The command's whole reason to exist is being runnable with no credentials and
// no network, so the test deliberately sets neither up.
func TestExportValidationRules_EmitsCatalogWithoutAuth(t *testing.T) {
	// The real binary initialises config via cobra.OnInitialize before any RunE;
	// catalog generation reads the API URL from it even though it never calls out.
	config.InitConfig(config.DefaultConfigFile())

	var stdout, stderr bytes.Buffer

	cmd := newCmdExportValidationRules()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	require.NoError(t, cmd.Execute())

	var catalog docs.DocumentedRules
	require.NoError(t, yaml.Unmarshal(stdout.Bytes(), &catalog))

	assert.Equal(t, 1, catalog.SchemaVersion)
	assert.NotEmpty(t, catalog.Rules, "catalog must carry the registered rules")

	// Rule IDs are the join key an agent uses to look up a diagnostic reported by
	// `validate`, so an entry without one makes the catalog useless downstream.
	for _, rule := range catalog.Rules {
		assert.NotEmpty(t, rule.RuleID)
		assert.NotEmpty(t, rule.Description)
	}
}
