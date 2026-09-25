package app

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSchemasMatchesSupportedKinds(t *testing.T) {
	Initialise("test")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	provider, err := newCompositeProvider()
	require.NoError(t, err)
	catalog, err := GenerateSchemas()
	require.NoError(t, err)

	kinds := provider.SupportedKinds()
	sort.Strings(kinds)
	assert.Equal(t, kinds, catalog.Kinds())
	assert.Contains(t, catalog.Kinds(), "data-graph")
}

func TestGenerateSchemasMatchesFlagGatedSupportedKinds(t *testing.T) {
	Initialise("test")
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	viper.Set("experimental", true)
	viper.Set("flags.retlConnectionSupport", true)
	viper.Set("flags.retlTableSupport", true)
	t.Cleanup(viper.Reset)

	provider, err := newCompositeProvider()
	require.NoError(t, err)
	catalog, err := GenerateSchemas()
	require.NoError(t, err)

	kinds := provider.SupportedKinds()
	sort.Strings(kinds)
	assert.Equal(t, kinds, catalog.Kinds())
	assert.Contains(t, catalog.Kinds(), "retl-source-table")
	assert.Contains(t, catalog.Kinds(), "retl-connections")
}
