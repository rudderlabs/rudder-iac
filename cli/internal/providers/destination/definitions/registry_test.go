package definitions_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)

	require.NoError(t, registry.Register(def))

	registered, err := registry.Get("WEBHOOK", 1)
	require.NoError(t, err)
	assert.Equal(t, def, registered.DestinationDefinition)
}

func TestRegistryDuplicateRegistration(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)

	require.NoError(t, registry.Register(def))
	err := registry.Register(def)
	require.Error(t, err)
}

func TestRegistryRejectsUnmappedSourceType(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.SourceTypes = []string{"unsupported_source"}

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `local source type "unsupported_source" has no API mapping`)
}

func TestRegistryRejectsConnectionModeWithoutSourceType(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.SourceTypes = []string{"web"}
	def.ConnectionModes = map[string][]string{"ios": {"cloud"}}

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `connection modes configured for unsupported source type "ios"`)
}

func TestRegistryRejectsSourceTypeWithoutConnectionModes(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.SourceTypes = append(def.SourceTypes, "ios")

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `source type "ios" has no connection modes`)
}

func TestRegistryRejectsMissingConnectionModes(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.ConnectionModes = nil

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `source type "web" has no connection modes`)
}

func TestRegistryRejectsConsentOverrideWithoutSourceType(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.ConsentValidationOverrides = map[string]common.ConsentValidator{
		"ios": common.ValidateConsentEntries,
	}

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `consent validation override configured for unsupported source type "ios"`)
}

func TestRegistryRejectsNilConsentOverride(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.ConsentValidationOverrides = map[string]common.ConsentValidator{"web": nil}

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `consent validation override for source type "web" is nil`)
}

func TestRegistryRejectsConsentOverrideWithoutSharedConsentModel(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := definitions.WebhookTestDefinition("WEBHOOK", 1)
	def.ConsentValidationOverrides = map[string]common.ConsentValidator{
		"web": common.ValidateConsentEntries,
	}

	err := registry.Register(def)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consent validation overrides require a common.ConsentManagement config field")
}

func TestRegistryRejectsNonSharedConsentModel(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	err := registry.Register(&definitions.DestinationDefinition{
		Type:        "TEST",
		Version:     1,
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
		NewConfig: func() any {
			return &struct {
				ConsentManagement map[string]any `mapstructure:"consent_management"`
			}{}
		},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "consent_management config field must use common.ConsentManagement")
}

func TestRegistrySupportedTypesAndVersions(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.WebhookTestDefinition("WEBHOOK", 1)))
	require.NoError(t, registry.Register(definitions.WebhookTestDefinition("WEBHOOK", 2)))
	require.NoError(t, registry.Register(definitions.GA4TestDefinition()))

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

func TestRegistryGetByAPIType(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	def := &definitions.DestinationDefinition{
		Type:    "s3",
		APIType: "S3",
		Version: 1,
		NewConfig: func() any {
			return &struct {
				BucketName string `mapstructure:"bucket_name" validate:"required"`
			}{}
		},
	}
	require.NoError(t, registry.Register(def))

	registered, err := registry.GetByAPIType("S3", 1)
	require.NoError(t, err)
	assert.Equal(t, "s3", registered.Type)
	assert.Equal(t, "S3", registered.APIType)

	_, err = registry.GetByAPIType("S3", 2)
	require.Error(t, err)
	_, err = registry.GetByAPIType("s3", 1)
	require.Error(t, err)
}

func TestRegistryAPITypeDefaultsToType(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.WebhookTestDefinition("WEBHOOK", 1)))

	registered, err := registry.Get("WEBHOOK", 1)
	require.NoError(t, err)
	assert.Equal(t, "WEBHOOK", registered.APIType)

	byAPI, err := registry.GetByAPIType("WEBHOOK", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestRegistryDuplicateAPITypeVersion(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:    "s3",
		APIType: "S3",
		Version: 1,
		NewConfig: func() any {
			return &struct {
				BucketName string `mapstructure:"bucket_name" validate:"required"`
			}{}
		},
	}))

	err := registry.Register(&definitions.DestinationDefinition{
		Type:    "amazon_s3",
		APIType: "S3",
		Version: 1,
		NewConfig: func() any {
			return &struct {
				BucketName string `mapstructure:"bucket_name" validate:"required"`
			}{}
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `apiType "S3" version 1 is already registered as "s3"`)
}

func TestRegisteredDefinitionMetadataAndConversion(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(definitions.GA4TestDefinition()))

	registered, err := registry.Get("GA4", 1)
	require.NoError(t, err)

	assert.Equal(t, []string{"api_secret"}, registered.SecretKeys())
	assert.ElementsMatch(t, []string{"web", "android"}, registered.SupportedSourceTypes())
	assert.Contains(t, registered.SupportedSourceTypes(), "web")
	assert.NotContains(t, registered.SupportedSourceTypes(), "ios")

	modes, err := registered.ConnectionModes("web")
	require.NoError(t, err)
	assert.Equal(t, []string{"cloud", "device", "hybrid"}, modes)

	assert.Equal(t, []string{"connection_mode", "use_native_sdk"}, registered.SourceTypeConfigKeys())

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
