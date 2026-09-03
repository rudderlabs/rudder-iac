package facebookconversions_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	facebookconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookconversions.NewDefinition()))

	registered, err := registry.Get("facebook_conversions", 1)
	require.NoError(t, err)

	assert.Equal(t, "facebook_conversions", registered.Type)
	assert.Equal(t, "FACEBOOK_CONVERSIONS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"access_token"}, registered.SecretKeys())
	assert.Empty(t, registered.GatedKeyPaths())
	assert.Equal(t, map[string]any{
		"action_source":      "website",
		"limited_data_usage": false,
		"test_destination":   false,
		"remove_external_id": false,
	}, registered.ConfigDefaults())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity",
		"cloud", "react_native", "flutter", "cordova",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	for _, sourceType := range expectedSourceTypes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, []string{"cloud"}, modes)
	}

	byAPI, err := registry.GetByAPIType("FACEBOOK_CONVERSIONS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestFacebookConversionsApplyDefaults(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookconversions.NewDefinition()))
	registered, err := registry.Get("facebook_conversions", 1)
	require.NoError(t, err)

	t.Run("fills defaults omitted by the spec", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, map[string]any{
			"dataset_id":         "dataset-1",
			"access_token":       "access-token-1",
			"action_source":      "website",
			"limited_data_usage": false,
			"test_destination":   false,
			"remove_external_id": false,
		}, registered.ApplyDefaults(validMinimalConfig()))
	})

	t.Run("keeps values the spec sets", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["action_source"] = "app"
		config["limited_data_usage"] = true
		config["test_destination"] = true
		config["remove_external_id"] = true

		assert.Equal(t, map[string]any{
			"dataset_id":         "dataset-1",
			"access_token":       "access-token-1",
			"action_source":      "app",
			"limited_data_usage": true,
			"test_destination":   true,
			"remove_external_id": true,
		}, registered.ApplyDefaults(config))
	})
}

func TestFacebookConversionsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(facebookconversions.NewDefinition()))
	registered, err := registry.Get("facebook_conversions", 1)
	require.NoError(t, err)

	t.Run("missing dataset_id", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"access_token": "access-token-1",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/dataset_id", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("missing access_token", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id": "dataset-1",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/access_token", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("invalid action_source", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id":    "dataset-1",
			"access_token":  "access-token-1",
			"action_source": "invalid",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/action_source", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	// schema.json declares a plain enum with no {{ … || … }} branch, so a
	// templated value would be stored verbatim and rejected by the backend.
	t.Run("action_source rejects dynamic values", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			"{{ .FACEBOOK_ACTION_SOURCE }}",
			`{{ .FACEBOOK_ACTION_SOURCE || website }}`,
			"env.FACEBOOK_ACTION_SOURCE",
		} {
			errors := registered.ValidateConfig(map[string]any{
				"dataset_id":    "dataset-1",
				"access_token":  "access-token-1",
				"action_source": value,
			})
			require.NotEmpty(t, errors, value)
			assert.Equal(t, "/action_source", errors[0].Path)
		}
	})

	t.Run("invalid mapped event target", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id":   "dataset-1",
			"access_token": "access-token-1",
			"events_to_events": []any{
				map[string]any{"from": "Signed Up", "to": "InvalidEvent"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/events_to_events/0/to", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("mapped event target rejects dynamic values", func(t *testing.T) {
		t.Parallel()
		for _, value := range []string{
			"{{ .FACEBOOK_EVENT_TO }}",
			`{{ .FACEBOOK_EVENT_TO || Purchase }}`,
			"env.FACEBOOK_EVENT_TO",
		} {
			errors := registered.ValidateConfig(map[string]any{
				"dataset_id":   "dataset-1",
				"access_token": "access-token-1",
				"events_to_events": []any{
					map[string]any{"from": "Signed Up", "to": value},
				},
			})
			require.NotEmpty(t, errors, value)
			assert.Equal(t, "/events_to_events/0/to", errors[0].Path)
		}
	})

	t.Run("mapped event target accepts empty value", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id":   "dataset-1",
			"access_token": "access-token-1",
			"events_to_events": []any{
				map[string]any{"from": "Signed Up", "to": ""},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("required strings reject empty values", func(t *testing.T) {
		t.Parallel()

		for _, field := range []string{"dataset_id", "access_token"} {
			t.Run(field, func(t *testing.T) {
				t.Parallel()
				config := validMinimalConfig()
				config[field] = ""

				errors := registered.ValidateConfig(config)
				require.NotEmpty(t, errors)
				assert.Equal(t, "/"+field, errors[0].Path)
			})
		}
	})

	t.Run("string fields reject values over maximum length", func(t *testing.T) {
		t.Parallel()

		// access_token allows 500; the rest allow 100.
		cases := patternFieldCases(strings.Repeat("x", 101))
		for i := range cases {
			if cases[i].name == "access_token" {
				cases[i].config["access_token"] = strings.Repeat("x", 501)
			}
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// The upstream constraints are `^(.{0,100})$` / `^(.{1,500})$` — patterns, not
	// length limits, so they forbid line breaks as well as bounding length.
	t.Run("string fields reject line breaks", func(t *testing.T) {
		t.Parallel()

		for _, tc := range patternFieldCases("line\nbreak") {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				errors := registered.ValidateConfig(tc.config)
				require.NotEmpty(t, errors)
				assert.Equal(t, tc.path, errors[0].Path)
			})
		}
	})

	// Upstream's template branch carries no length cap, so a template longer than
	// the literal limit is still valid.
	t.Run("string fields accept ui templates of any length", func(t *testing.T) {
		t.Parallel()

		long := "{{ config.value || " + strings.Repeat("x", 600) + " }}"
		for _, tc := range patternFieldCases(long) {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				assert.Empty(t, registered.ValidateConfig(tc.config))
			})
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(validMinimalConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(validFullConfig())
		assert.Empty(t, errors)
	})

	t.Run("nested array item fields follow schema optionality", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id":   "dataset-1",
			"access_token": "access-token-1",
			"events_to_events": []any{
				map[string]any{},
			},
			"blacklist_pii_properties": []any{
				map[string]any{},
			},
			"whitelist_pii_properties": []any{
				map[string]any{},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"dataset_id":         "123456789012345",
			"access_token":       "{{ .FACEBOOK_CONVERSIONS_ACCESS_TOKEN }}",
			"action_source":      "website",
			"limited_data_usage": false,
			"test_destination":   true,
			"test_event_code":    "TEST12345",
			"remove_external_id": false,
			"events_to_events": []any{
				map[string]any{"from": "Product Viewed", "to": "ViewContent"},
				map[string]any{"from": "Order Completed", "to": "Purchase"},
			},
			"blacklist_pii_properties": []any{
				map[string]any{"property": "email", "hash": true},
			},
			"whitelist_pii_properties": []any{
				map[string]any{"property": "phone"},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"cloud_source": []any{},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/cloud_source", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'cloud_source' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		config := validMinimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{"provider": "unknown"},
			},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
	// connection_mode legality is per source type, taken from this definition's
	// own ConnectionModes map rather than a shared enum.
	t.Run("connection_mode accepts a supported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "cloud"},
		})

		for _, err := range errors {
			assert.NotEqual(t, "/connection_mode/web", err.Path)
		}
	})

	t.Run("connection_mode rejects an unsupported mode", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"connection_mode": map[string]any{"web": "device"},
		})

		var found bool
		for _, err := range errors {
			if err.Path == "/connection_mode/web" {
				found = true
				assert.Contains(t, err.Message, "must be one of")
			}
		}
		assert.True(t, found, "expected /connection_mode/web to be rejected")
	})

}

func TestFacebookConversionsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := facebookconversions.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"dataset_id": "dataset-1",
				"access_token": "access-token-1"
			}`,
			APIJSON: `{
				"datasetId": "dataset-1",
				"accessToken": "access-token-1"
			}`,
		},
		{
			Name: "full",
			LocalJSON: `{
				"dataset_id": "dataset-1",
				"access_token": "access-token-1",
				"action_source": "website",
				"limited_data_usage": true,
				"test_destination": true,
				"test_event_code": "TEST12345",
				"remove_external_id": true,
				"events_to_events": [
					{"from": "Product Viewed", "to": "ViewContent"},
					{"from": "Order Completed", "to": "Purchase"}
				],
				"blacklist_pii_properties": [
					{"property": "email", "hash": true},
					{"property": "phone", "hash": false}
				],
				"whitelist_pii_properties": [
					{"property": "country"}
				]
			}`,
			APIJSON: `{
				"datasetId": "dataset-1",
				"accessToken": "access-token-1",
				"actionSource": "website",
				"limitedDataUSage": true,
				"testDestination": true,
				"testEventCode": "TEST12345",
				"removeExternalId": true,
				"eventsToEvents": [
					{"from": "Product Viewed", "to": "ViewContent"},
					{"from": "Order Completed", "to": "Purchase"}
				],
				"blacklistPiiProperties": [
					{"blacklistPiiProperties": "email", "blacklistPiiHash": true},
					{"blacklistPiiProperties": "phone", "blacklistPiiHash": false}
				],
				"whitelistPiiProperties": [
					{"whitelistPiiProperties": "country"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"dataset_id": "dataset-1",
				"access_token": "access-token-1",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"datasetId": "dataset-1",
				"accessToken": "access-token-1",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

// patternFieldCase is one pattern-validated string field, with a config that is
// otherwise valid so the only error can come from the field under test.
type patternFieldCase struct {
	name   string
	path   string
	config map[string]any
}

// patternFieldCases returns one case per pattern-validated string field, each
// carrying value in that field.
func patternFieldCases(value string) []patternFieldCase {
	return []patternFieldCase{
		{
			name: "dataset_id",
			path: "/dataset_id",
			config: map[string]any{
				"dataset_id":   value,
				"access_token": "access-token-1",
			},
		},
		{
			name: "access_token",
			path: "/access_token",
			config: map[string]any{
				"dataset_id":   "dataset-1",
				"access_token": value,
			},
		},
		{
			name: "test_event_code",
			path: "/test_event_code",
			config: map[string]any{
				"dataset_id":      "dataset-1",
				"access_token":    "access-token-1",
				"test_event_code": value,
			},
		},
		{
			name: "event from",
			path: "/events_to_events/0/from",
			config: map[string]any{
				"dataset_id":   "dataset-1",
				"access_token": "access-token-1",
				"events_to_events": []any{
					map[string]any{"from": value, "to": "Purchase"},
				},
			},
		},
		{
			name: "denylist property",
			path: "/blacklist_pii_properties/0/property",
			config: map[string]any{
				"dataset_id":   "dataset-1",
				"access_token": "access-token-1",
				"blacklist_pii_properties": []any{
					map[string]any{"property": value, "hash": true},
				},
			},
		},
		{
			name: "allowlist property",
			path: "/whitelist_pii_properties/0/property",
			config: map[string]any{
				"dataset_id":   "dataset-1",
				"access_token": "access-token-1",
				"whitelist_pii_properties": []any{
					map[string]any{"property": value},
				},
			},
		},
	}
}

func validMinimalConfig() map[string]any {
	return map[string]any{
		"dataset_id":   "dataset-1",
		"access_token": "access-token-1",
	}
}

func validFullConfig() map[string]any {
	return map[string]any{
		"dataset_id":         "dataset-1",
		"access_token":       "access-token-1",
		"action_source":      "website",
		"limited_data_usage": true,
		"test_destination":   true,
		"test_event_code":    "TEST12345",
		"remove_external_id": true,
		"events_to_events": []any{
			map[string]any{"from": "Product Viewed", "to": "ViewContent"},
			map[string]any{"from": "Order Completed", "to": "Purchase"},
		},
		"blacklist_pii_properties": []any{
			map[string]any{"property": "email", "hash": true},
			map[string]any{"property": "phone", "hash": false},
		},
		"whitelist_pii_properties": []any{
			map[string]any{"property": "country"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"marketing"},
				},
			},
		},
	}
}
