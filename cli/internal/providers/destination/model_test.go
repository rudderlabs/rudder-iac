package destination

import (
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/handler"
	ttypes "github.com/rudderlabs/rudder-iac/cli/internal/providers/transformations/types"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDestinationSpecMapstructureDecode(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"id":                 "ga4-production",
		"display_name":       "Production GA4",
		"type":               "GA4",
		"enabled":            true,
		"definition_version": int64(1),
		"transformation":     "#transformation:my-transform",
		"config": map[string]any{
			"api_secret":     "secret",
			"measurement_id": "G-123",
		},
	}

	var spec DestinationSpec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result: &spec,
	})
	require.NoError(t, err)
	require.NoError(t, decoder.Decode(raw))

	assert.Equal(t, DestinationSpec{
		ID:                "ga4-production",
		DisplayName:       "Production GA4",
		Type:              "GA4",
		Enabled:           true,
		DefinitionVersion: 1,
		Transformation:    "#transformation:my-transform",
		Config: map[string]any{
			"api_secret":     "secret",
			"measurement_id": "G-123",
		},
	}, spec)
}

func TestDestinationSpecMapstructureDecodeDefinitionVersionAsInt(t *testing.T) {
	t.Parallel()

	// YAML decodes integers as int, not int64. mapstructure must coerce to int64.
	raw := map[string]any{
		"id":                 "x",
		"definition_version": 2, // int
		"config":             map[string]any{},
	}

	var spec DestinationSpec
	require.NoError(t, mapstructure.Decode(raw, &spec))
	assert.Equal(t, int64(2), spec.DefinitionVersion)
}

func TestParseTransformationRef(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		ref, err := parseTransformationRef("#transformation:my-transform")
		require.NoError(t, err)
		require.NotNil(t, ref)
		assert.Equal(t, resources.URN("my-transform", ttypes.TransformationResourceType), ref.URN)
		assert.Equal(t, "id", ref.Property)
		assert.NotNil(t, ref.Resolve, "Resolve function must be wired so the apply framework can resolve it")
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		_, err := parseTransformationRef("")
		require.Error(t, err)
	})

	t.Run("wrong kind", func(t *testing.T) {
		t.Parallel()
		_, err := parseTransformationRef("#source:my-source")
		require.Error(t, err)
	})

	t.Run("missing id", func(t *testing.T) {
		t.Parallel()
		_, err := parseTransformationRef("#transformation:")
		require.Error(t, err)
	})

	t.Run("missing prefix", func(t *testing.T) {
		t.Parallel()
		_, err := parseTransformationRef("transformation:my-transform")
		require.Error(t, err)
	})
}

func TestRemoteDestinationMetadata(t *testing.T) {
	t.Parallel()

	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		r := RemoteDestination{Destination: &client.Destination{
			ID:         "dst-1",
			ExternalID: "ga4-production",
			Name:       "Production GA4",
		}}
		assert.Equal(t, handler.RemoteResourceMetadata{
			ID:         "dst-1",
			ExternalID: "ga4-production",
			Name:       "Production GA4",
		}, r.Metadata())
	})

	t.Run("empty external id", func(t *testing.T) {
		t.Parallel()
		r := RemoteDestination{Destination: &client.Destination{
			ID:   "dst-2",
			Name: "Unmanaged",
		}}
		assert.Equal(t, handler.RemoteResourceMetadata{
			ID:         "dst-2",
			ExternalID: "",
			Name:       "Unmanaged",
		}, r.Metadata())
	})
}
