package googleadwordsremarketinglists_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	googleadwordsremarketinglists "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/google_adwords_remarketing_lists"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func register(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleadwordsremarketinglists.NewDefinition()))

	registered, err := registry.Get("google_adwords_remarketing_lists", 1)
	require.NoError(t, err)
	return registered
}

// validConfig is the minimal config every subtest starts from.
func validConfig() map[string]any {
	return map[string]any{
		"rudder_account_id": "gads-account",
		"customer_id":       "1234567890",
		"sub_account":       false,
	}
}

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(googleadwordsremarketinglists.NewDefinition()))

	registered, err := registry.Get("google_adwords_remarketing_lists", 1)
	require.NoError(t, err)

	assert.Equal(t, "google_adwords_remarketing_lists", registered.Type)
	assert.Equal(t, "GOOGLE_ADWORDS_REMARKETING_LISTS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Empty(t, registered.SecretKeys())

	expectedSourceTypes := []string{"cloud", "warehouse"}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	// Upstream lists shopify; the CLI never declares it.
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")

	assert.Equal(t, []string{"audience_id"}, registered.ConnectionRequiredKeys("cloud", "cloud"))
	assert.Nil(t, registered.ConnectionRequiredKeys("warehouse", "cloud"))

	assert.Equal(t, []string{"mirror"}, registered.SyncBehaviours())
	assert.False(t, registered.SupportsVisualMapper())
	assert.True(t, registered.SupportsVisualMapperV2())

	byAPI, err := registry.GetByAPIType("GOOGLE_ADWORDS_REMARKETING_LISTS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestGoogleAdwordsRemarketingListsConfigValidation(t *testing.T) {
	t.Parallel()

	registered := register(t)

	for _, key := range []string{"rudder_account_id", "customer_id", "sub_account"} {
		t.Run("missing "+key, func(t *testing.T) {
			t.Parallel()
			config := validConfig()
			delete(config, key)

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("customer_id accepts a template and rejects a multiline literal", func(t *testing.T) {
		t.Parallel()

		config := validConfig()
		config["customer_id"] = "{{ .GADS_CUSTOMER_ID || 1234567890 }}"
		assert.Empty(t, registered.ValidateConfig(config))

		config["customer_id"] = "123\n456"
		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/customer_id", errors[0].Path)
	})

	// The schema.json allOf branch keyed on subAccount, as a required_if tag.
	t.Run("login_customer_id required when sub_account is true", func(t *testing.T) {
		t.Parallel()

		config := validConfig()
		config["sub_account"] = true

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/login_customer_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")

		config["login_customer_id"] = "9876543210"
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("login_customer_id optional when sub_account is false", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validConfig()))
	})

	t.Run("invalid enum values rejected", func(t *testing.T) {
		t.Parallel()

		for key, value := range map[string]string{
			"type_of_list":            "Nonsense",
			"user_data_consent":       "MAYBE",
			"personalization_consent": "MAYBE",
		} {
			config := validConfig()
			config[key] = value

			errors := registered.ValidateConfig(config)
			require.Len(t, errors, 1, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "must be one of")
		}
	})

	t.Run("invalid user_schema member rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["user_schema"] = []any{"email", "EMAIL"}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/user_schema/1", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// The spec shipped in the README and used for the rETL validate proof.
	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id": "gads-remarketing-account",
			"customer_id":       "1234567890",
			"sub_account":       false,
			"user_schema":       []any{"email", "phone"},
			"is_hash_required":  true,
			"type_of_list":      "General",
			"connection_mode":   map[string]any{"warehouse": "cloud"},
		}))
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(map[string]any{
			"rudder_account_id":       "gads-account",
			"customer_id":             "1234567890",
			"sub_account":             true,
			"login_customer_id":       "9876543210",
			"list_id":                 "list-1",
			"audience_id":             "aud-1",
			"user_schema":             []any{"email", "phone", "addressInfo"},
			"is_hash_required":        true,
			"type_of_list":            "userID",
			"user_data_consent":       "GRANTED",
			"personalization_consent": "DENIED",
			"connection_mode":         map[string]any{"cloud": "cloud", "warehouse": "cloud"},
			"consent_management": map[string]any{
				"warehouse": []any{
					map[string]any{
						"provider":            "custom",
						"resolution_strategy": "or",
						"consents":            []any{"one", "two"},
					},
				},
			},
		}))
	})

	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["connection_mode"] = map[string]any{"warehouse": "cloud"}
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["connection_mode"] = map[string]any{"warehouse": "device"}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["consent_management"] = map[string]any{"web": []any{}}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'web' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()
		config := validConfig()
		config["consent_management"] = map[string]any{
			"cloud": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestGoogleAdwordsRemarketingListsDefaults(t *testing.T) {
	t.Parallel()

	registered := register(t)

	// schema.json defaults these; a spec that omits them must still carry them
	// or every apply reports a phantom diff. subAccount is defaulted upstream
	// too, but it is also required, so the spec always supplies it.
	assert.Equal(t, map[string]any{
		"rudder_account_id":       "gads-account",
		"customer_id":             "1234567890",
		"sub_account":             false,
		"is_hash_required":        true,
		"type_of_list":            "General",
		"user_data_consent":       "UNSPECIFIED",
		"personalization_consent": "UNSPECIFIED",
	}, registered.ApplyDefaults(validConfig()))
}

func TestGoogleAdwordsRemarketingListsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := googleadwordsremarketinglists.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"rudder_account_id": "gads-account",
				"customer_id": "1234567890",
				"sub_account": false
			}`,
			APIJSON: `{
				"rudderAccountId": "gads-account",
				"customerId": "1234567890",
				"subAccount": false
			}`,
		},
		{
			Name: "full with consent",
			LocalJSON: `{
				"rudder_account_id": "gads-account",
				"customer_id": "1234567890",
				"sub_account": true,
				"login_customer_id": "9876543210",
				"list_id": "list-1",
				"audience_id": "aud-1",
				"user_schema": ["email", "phone"],
				"is_hash_required": true,
				"type_of_list": "userID",
				"user_data_consent": "GRANTED",
				"personalization_consent": "DENIED",
				"connection_mode": {"cloud": "cloud", "warehouse": "cloud"},
				"consent_management": {
					"cloud": [
						{
							"provider": "custom",
							"resolution_strategy": "or",
							"consents": ["one_cloud", "two_cloud"]
						}
					]
				}
			}`,
			APIJSON: `{
				"rudderAccountId": "gads-account",
				"customerId": "1234567890",
				"subAccount": true,
				"loginCustomerId": "9876543210",
				"listId": "list-1",
				"audienceId": "aud-1",
				"userSchema": ["email", "phone"],
				"isHashRequired": true,
				"typeOfList": "userID",
				"userDataConsent": "GRANTED",
				"personalizationConsent": "DENIED",
				"connectionMode": {"cloud": "cloud", "warehouse": "cloud"},
				"consentManagement": {
					"cloud": [
						{
							"provider": "custom",
							"resolutionStrategy": "or",
							"consents": [
								{"consent": "one_cloud"},
								{"consent": "two_cloud"}
							]
						}
					]
				}
			}`,
		},
	})
}
