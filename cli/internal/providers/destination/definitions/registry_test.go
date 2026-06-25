package definitions_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := sampleDefinition("WEBHOOK", 1)

	require.NoError(t, registry.Register(def))

	registered, err := registry.Get("WEBHOOK", 1)
	require.NoError(t, err)
	assert.Equal(t, def, registered.DestinationDefinition)
}

func TestRegistryDuplicateRegistration(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := sampleDefinition("WEBHOOK", 1)

	require.NoError(t, registry.Register(def))
	err := registry.Register(def)
	require.Error(t, err)
}

func TestRegistrySupportedTypesAndVersions(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(sampleDefinition("WEBHOOK", 1)))
	require.NoError(t, registry.Register(sampleDefinition("WEBHOOK", 2)))
	require.NoError(t, registry.Register(sampleDefinition("GA4", 1)))

	assert.ElementsMatch(t, []string{"GA4", "WEBHOOK"}, registry.SupportedTypes())
	assert.True(t, registry.IsSupported("WEBHOOK"))
	assert.False(t, registry.IsSupported("S3"))

	versions, err := registry.Versions("WEBHOOK")
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, versions)

	_, err = registry.Versions("S3")
	require.Error(t, err)
}

func TestRegistryGetUnknown(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	_, err := registry.Get("WEBHOOK", 1)
	require.Error(t, err)
}

func TestRegisteredDefinitionMetadataAndConversion(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := &definitions.DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("apiSecret", "api_secret"),
			converter.Simple("measurementId", "measurement_id"),
		},
		SecretKeys: []string{"api_secret"},
		Schema:     json.RawMessage(testSchema),
	}
	require.NoError(t, registry.Register(def))

	registered, err := registry.Get("GA4", 1)
	require.NoError(t, err)

	assert.Equal(t, []string{"api_secret"}, registered.SecretKeys())
	assert.ElementsMatch(t, []string{"web", "android"}, registered.SupportedSourceTypes())
	assert.True(t, registered.IsSourceTypeSupported("web"))
	assert.False(t, registered.IsSourceTypeSupported("ios"))

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud", "device", "hybrid"}, modes)

	assert.Equal(t, []string{"connection_mode", "use_native_sdk", "consent_management"}, registered.SourceTypeConfigKeys())

	local := map[string]any{
		"api_secret":     "secret",
		"measurement_id": "G-123",
	}
	api, err := registered.LocalToAPI(local)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"apiSecret":     "secret",
		"measurementId": "G-123",
	}, api)

	back, err := registered.APIToLocal(api)
	require.NoError(t, err)
	assert.Equal(t, local, back)
}

func sampleDefinition(destType string, version int64) *definitions.DestinationDefinition {
	return &definitions.DestinationDefinition{
		Type:    destType,
		Version: version,
		Properties: []converter.ConfigProperty{
			converter.Simple("webhookUrl", "webhook_url"),
		},
		Schema: json.RawMessage(`{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type": "object",
			"required": ["webhook_url"],
			"properties": {
				"webhook_url": { "type": "string" },
				"connection_mode": {
					"type": "object",
					"properties": {
						"web": { "type": "string", "enum": ["cloud"] }
					}
				}
			},
			"additionalProperties": false
		}`),
	}
}
