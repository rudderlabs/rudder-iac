package fbcustomaudience_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	fbcustomaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/fb_custom_audience"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(fbcustomaudience.NewDefinition()))

	registered, err := registry.Get("fb_custom_audience", 1)
	require.NoError(t, err)
	return registered
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(fbcustomaudience.NewDefinition()))

	registered, err := registry.Get("fb_custom_audience", 1)
	require.NoError(t, err)

	assert.Equal(t, "fb_custom_audience", registered.Type)
	assert.Equal(t, "FB_CUSTOM_AUDIENCE", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"access_token", "app_secret"}, registered.SecretKeys())

	expectedSourceTypes := []string{"cloud", "warehouse"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	// Upstream lists shopify; the CLI never declares it.
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "web")

	assert.Equal(t, []string{"audience_id"}, registered.ConnectionRequiredKeys("cloud", "cloud"))
	assert.Equal(t, []string{"ad_account_id"}, registered.ConnectionRequiredKeys("warehouse", "cloud"))
	assert.Nil(t, registered.ConnectionRequiredKeys("warehouse", "device"))

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("FB_CUSTOM_AUDIENCE", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestFbCustomAudienceConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	t.Run("missing rudder_account_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/rudder_account_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	// schema.json constrains rudderAccountId to a bare `^.{1,100}$` with no
	// {{ … || … }} branch, so the tag is plain `pattern`. That still admits a
	// short template as a literal, exactly as the upstream regex does — the
	// constraint is length and single-line-ness, nothing more.
	t.Run("rudder_account_id rejects overlong and multiline values", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			strings.Repeat("a", 101),
			"bad\nvalue",
		} {
			errors := registered.ValidateConfig(map[string]any{"rudder_account_id": value})
			require.NotEmpty(t, errors, value)
			assert.Equal(t, "/rudder_account_id", errors[0].Path)
		}
	})

	t.Run("ad_account_id accepts a template and rejects a bad literal", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"ad_account_id":     "{{ .FB_AD_ACCOUNT_ID || act_123 }}",
		})
		assert.Empty(t, errors)

		errors = registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"ad_account_id":     "act\n123",
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/ad_account_id", errors[0].Path)
	})

	t.Run("invalid user_schema member rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"user_schema":       []any{"EMAIL", "NOT_A_FIELD"},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/user_schema/1", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid type rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"type":              "SOMETHING_ELSE",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/type", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("invalid sub_type rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"sub_type":          "SOMETHING_ELSE",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/sub_type", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		})
		assert.Empty(t, errors)
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"connection_mode":   map[string]any{"warehouse": "device"},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{"rudder_account_id": "acct-1"})
		assert.Empty(t, errors)
	})

	// The spec shipped in the README and used for the rETL validate proof.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "fb-audience-account",
			"ad_account_id":     "act_1234567890",
			"user_schema":       []any{"EMAIL", "PHONE", "FN", "LN"},
			"is_hash_required":  true,
			"type":              "FILE_IMPORTED",
			"sub_type":          "MULTI_HASHES",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"access_token":      "access-token-1",
			"app_secret":        "app-secret-1",
			"user_schema":       []any{"EMAIL", "PHONE"},
			"is_hash_required":  true,
			"is_raw":            false,
			"disable_format":    false,
			"type":              "FILE_IMPORTED",
			"sub_type":          "MULTI_HASHES",
			"ad_account_id":     "act_1234567890",
			"audience_id":       "aud-1",
			"connection_mode":   map[string]any{"cloud": "cloud", "warehouse": "cloud"},
			"consent_management": map[string]any{
				"warehouse": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "and",
						"consents":            []any{"one", "two"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"not_a_field":       true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id":  "acct-1",
			"consent_management": map[string]any{"web": []any{}},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'web' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"rudder_account_id": "acct-1",
			"consent_management": map[string]any{
				"warehouse": []any{map[string]any{"provider": "unknown"}},
			},
		})
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestFbCustomAudienceDefaults(t *testing.T) {
	t.Parallel()

	registered := register(t)

	// schema.json defaults these; a spec that omits them must still carry them
	// or every apply reports a phantom diff.
	assert.Equal(t, map[string]any{
		"rudder_account_id": "acct-1",
		"is_hash_required":  true,
		"is_raw":            false,
		"disable_format":    false,
		"type":              "NA",
		"sub_type":          "NA",
	}, registered.ApplyDefaults(map[string]any{"rudder_account_id": "acct-1"}))

	// An explicit value wins, including one equal to the zero value.
	applied := registered.ApplyDefaults(map[string]any{
		"rudder_account_id": "acct-1",
		"is_hash_required":  false,
	})
	assert.Equal(t, false, applied["is_hash_required"])
}

func TestFbCustomAudienceConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := fbcustomaudience.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "acct-1"
			}`,
			APIJSON: `{
				"rudderAccountId": "acct-1"
			}`,
		},
		{
			Name: "full with consent",
			LocalJSON: `{
				"rudder_account_id": "acct-1",
				"access_token": "access-token-1",
				"app_secret": "app-secret-1",
				"user_schema": ["EMAIL", "PHONE"],
				"is_hash_required": true,
				"is_raw": false,
				"disable_format": false,
				"type": "FILE_IMPORTED",
				"sub_type": "MULTI_HASHES",
				"ad_account_id": "act_1234567890",
				"audience_id": "aud-1",
				"connection_mode": {"cloud": "cloud", "warehouse": "cloud"},
				"consent_management": {
					"warehouse": [
						{
							"provider": "custom",
							"resolution_strategy": "and",
							"consents": ["one_warehouse", "two_warehouse"]
						}
					]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "acct-1",
				"accessToken": "access-token-1",
				"appSecret": "app-secret-1",
				"userSchema": ["EMAIL", "PHONE"],
				"isHashRequired": true,
				"isRaw": false,
				"disableFormat": false,
				"type": "FILE_IMPORTED",
				"subType": "MULTI_HASHES",
				"adAccountId": "act_1234567890",
				"audienceId": "aud-1",
				"connectionMode": {"cloud": "cloud", "warehouse": "cloud"},
				"consentManagement": {
					"warehouse": [
						{
							"provider": "custom",
							"resolutionStrategy": "and",
							"consents": [
								{"consent": "one_warehouse"},
								{"consent": "two_warehouse"}
							]
						}
					]
				}
			}`,
		},
	})
}
