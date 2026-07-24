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

func TestNewDestinationRegistryFlagMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                   string
		destinationSupport     bool
		unverifiedDestinations bool
		wantTypes              []string
	}{
		{
			name:                   "both flags disabled",
			destinationSupport:     false,
			unverifiedDestinations: false,
			wantTypes:              []string{},
		},
		{
			name:                   "destinationSupport off ignores unverifiedDestinations",
			destinationSupport:     false,
			unverifiedDestinations: true,
			wantTypes:              []string{},
		},
		{
			name:                   "destinationSupport on without unverifiedDestinations",
			destinationSupport:     true,
			unverifiedDestinations: false,
			wantTypes:              []string{},
		},
		{
			name:                   "both flags enabled registers s3",
			destinationSupport:     true,
			unverifiedDestinations: true,
			wantTypes:              []string{"s3"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Config{
				ExperimentalFlags: config.ExperimentalConfig{
					DestinationSupport:     tc.destinationSupport,
					UnverifiedDestinations: tc.unverifiedDestinations,
				},
			}

			registry, err := newDestinationRegistry(cfg)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTypes, registry.SupportedTypes())
		})
	}
}
