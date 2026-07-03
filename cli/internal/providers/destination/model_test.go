package destination_test

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestinationSpecMapstructureDecode(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"id":                 "ga4-production",
		"display_name":       "Production GA4",
		"type":               "GA4",
		"enabled":            true,
		"definition_version": int64(1),
		"transformation":     "#transformation:my-transform",
		"config": map[string]any{
			"api_secret": "secret",
		},
	}

	var spec destination.DestinationSpec
	require.NoError(t, mapstructure.Decode(input, &spec))

	assert.Equal(t, &destination.DestinationSpec{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Transformation:    "#transformation:my-transform",
		Config: map[string]any{
			"api_secret": "secret",
		},
	}, &spec)
}

func TestRemoteDestinationMetadata(t *testing.T) {
	t.Parallel()

	remote := destination.RemoteDestination{
		Destination: &client.Destination{
			ID:         "dst-remote-1",
			ExternalID: "ga4-production",
			Name:       "Production GA4",
		},
	}

	assert.Equal(t, handler.RemoteResourceMetadata{
		ID:         "dst-remote-1",
		ExternalID: "ga4-production",
		Name:       "Production GA4",
	}, remote.Metadata())
}
