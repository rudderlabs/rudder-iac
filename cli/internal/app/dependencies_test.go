package app

import (
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComposeProvidersIncludesDataGraph(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))

	c, err := client.New("test-token")
	require.NoError(t, err)

	composite, providers, err := composeProviders(c)
	require.NoError(t, err)
	require.NotNil(t, providers.DataGraph)

	cp, ok := composite.(*provider.CompositeProvider)
	require.True(t, ok)
	assert.Same(t, providers.DataGraph, cp.Providers["datagraph"])
}

func TestNewDestinationRegistryRegistersS3WhenFlagEnabled(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", true)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	registry, err := newDestinationRegistry(config.GetConfig())
	require.NoError(t, err)
	assert.Equal(t, []string{"s3"}, registry.SupportedTypes())
}

func TestNewDestinationRegistryEmptyWhenFlagDisabled(t *testing.T) {
	config.InitConfig(filepath.Join(t.TempDir(), "config.json"))
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.destinationSupport")
	viper.Set("experimental", true)
	viper.Set("flags.destinationSupport", false)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.destinationSupport", prevFlag)
	})

	registry, err := newDestinationRegistry(config.GetConfig())
	require.NoError(t, err)
	assert.Empty(t, registry.SupportedTypes())
}
