package app

import (
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
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
