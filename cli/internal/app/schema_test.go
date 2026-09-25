package app

import (
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/importmanifest"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/connection"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/retl/table"
	"github.com/rudderlabs/rudder-iac/cli/internal/schema"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSchemas(t *testing.T) {
	t.Run("default schemas match composite kinds and include data graph without credentials", func(t *testing.T) {
		t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
		t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "false")
		config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

		schemas, err := GenerateSchemas()
		require.NoError(t, err)

		composite, err := newCompositeProvider()
		require.NoError(t, err)
		require.Implements(t, (*provider.SchemaProvider)(nil), composite)
		expectedKinds := append(composite.SupportedKinds(), importmanifest.New().SupportedKinds()...)
		assert.ElementsMatch(t, expectedKinds, schema.Kinds(schemas))
		assert.Contains(t, schemas, "data-graph")
		assert.Contains(t, schemas, importmanifest.KindImportManifest)
	})

	t.Run("experimental flags include RETL table and connection schemas", func(t *testing.T) {
		t.Setenv("RUDDERSTACK_ACCESS_TOKEN", "")
		t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")
		t.Setenv("RUDDERSTACK_X_RETL_TABLE_SUPPORT", "true")
		config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
		// The app package's rule-catalog test restores this explicit viper key;
		// set it directly so this test remains order-independent.
		previous := viper.Get("flags.retlConnectionSupport")
		viper.Set("flags.retlConnectionSupport", true)
		t.Cleanup(func() { viper.Set("flags.retlConnectionSupport", previous) })

		schemas, err := GenerateSchemas()
		require.NoError(t, err)
		assert.Contains(t, schemas, table.ResourceKind)
		assert.Contains(t, schemas, connection.ResourceKind)
	})
}
