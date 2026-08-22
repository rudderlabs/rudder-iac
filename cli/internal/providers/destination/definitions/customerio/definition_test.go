package customerio_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
	"github.com/rudderlabs/rudder-iac/cli/internal/resources"
	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customerio.NewDefinition()))

	registered, err := registry.Get("customerio", 1)
	require.NoError(t, err)

	assert.Equal(t, "customerio", registered.Type)
	assert.Equal(t, "CUSTOMERIO", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_key"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web", "unity", "amp",
		"cloud", "warehouse", "react_native", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud", "device"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud", "device"},
		"ios_swift":      {"cloud"},
		"web":            {"cloud", "device"},
		"unity":          {"cloud"},
		"amp":            {"cloud"},
		"cloud":          {"cloud"},
		"warehouse":      {"cloud"},
		"react_native":   {"cloud"},
		"flutter":        {"cloud"},
		"cordova":        {"cloud"},
		"shopify":        {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, sourceType)
	}

	assert.Equal(t, map[string][]string{
		"auto_track_device_attributes/android":         {"android"},
		"auto_track_device_attributes/ios":             {"ios"},
		"background_queue_min_number_of_tasks/android": {"android"},
		"background_queue_seconds_delay/android":       {"android"},
		"data_use_in_app/web":                          {"web"},
		"send_page_name_in_sdk/web":                    {"web"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("CUSTOMERIO", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestCustomerioConfigValidation(t *testing.T) {
	t.Parallel()

	registered := registeredCustomerioDefinition(t)

	for _, tc := range []struct {
		name string
		key  string
		path string
	}{
		{name: "missing site_id", key: "site_id", path: "/site_id"},
		{name: "missing api_key", key: "api_key", path: "/api_key"},
		{name: "missing datacenter", key: "datacenter", path: "/datacenter"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config := minimalConfig()
			delete(config, tc.key)

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors)
			assert.Equal(t, tc.path, errors[0].Path)
			assert.Contains(t, errors[0].Message, "required")
		})
	}

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(minimalConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(fullConfig())
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(exampleConfig())
		assert.Empty(t, errors)
	})

	t.Run("invalid datacenter rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["datacenter"] = "APAC"

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/datacenter", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("datacenter accepts dynamic values", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["datacenter"] = "{{ .CUSTOMERIO_DATACENTER }}"

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("single line fields reject invalid literals", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			field string
			value any
			path  string
		}{
			{field: "site_id", value: "bad\nsite", path: "/site_id"},
			{field: "api_key", value: "bad\nkey", path: "/api_key"},
			{field: "device_token_event_name", value: "bad\nevent", path: "/device_token_event_name"},
			{field: "background_queue_min_number_of_tasks", value: map[string]any{"android": "bad\n10"}, path: "/background_queue_min_number_of_tasks/android"},
			{field: "background_queue_seconds_delay", value: map[string]any{"android": "bad\n30"}, path: "/background_queue_seconds_delay/android"},
			{field: "event_filtering", value: map[string]any{"whitelist": []any{"bad\nevent"}}, path: "/event_filtering/whitelist/0"},
		}

		for _, tc := range cases {
			config := minimalConfig()
			config[tc.field] = tc.value

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors, tc.field)
			assert.Equal(t, tc.path, errors[0].Path)
		}
	})

	t.Run("pattern fields accept templates", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["site_id"] = "{{ config.siteID || site-id-1 }}"
		config["api_key"] = "{{ config.apiKey || api-key-1 }}"
		config["device_token_event_name"] = "{{ config.event || Device Token Registered }}"
		config["background_queue_min_number_of_tasks"] = map[string]any{"android": "{{ config.minTasks || 10 }}"}
		config["background_queue_seconds_delay"] = map[string]any{"android": "{{ config.delay || 30 }}"}
		config["event_filtering"] = map[string]any{"whitelist": []any{"{{ config.event || Product Purchased }}"}}

		errors := registered.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("event filtering rejects whitelist and blacklist together", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Product Purchased"},
			"blacklist": []any{"Password Reset"},
		}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assertValidationPaths(t, errors, "/event_filtering/whitelist", "/event_filtering/blacklist")
	})

	t.Run("unknown nested source key rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["use_native_sdk"] = map[string]any{"android_kotlin": true}

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/use_native_sdk/android_kotlin", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent blocks are not supported keys", func(t *testing.T) {
		t.Parallel()

		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			config := minimalConfig()
			config[key] = map[string]any{"web": []any{}}

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["not_a_field"] = true

		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"ios_swift": []any{map[string]any{"provider": "unknown"}},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})

	t.Run("duplicate consent provider rejected", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"web": []any{
				map[string]any{"provider": "oneTrust"},
				map[string]any{"provider": "oneTrust"},
			},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/web/1/provider", errors[0].Path)
	})

	t.Run("custom consent provider requires resolution strategy", func(t *testing.T) {
		t.Parallel()

		config := minimalConfig()
		config["consent_management"] = map[string]any{
			"warehouse": []any{map[string]any{"provider": "custom"}},
		}

		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse/0/resolution_strategy", errors[0].Path)
	})
}

func TestCustomerioConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := customerio.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"datacenter": "US"
			}`,
			APIJSON: `{
				"siteID": "site-id-1",
				"apiKey": "api-key-1",
				"datacenter": "US"
			}`,
		},
		{
			Name: "full sdk and cloud config",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"device_token_event_name": "Device Token Registered",
				"datacenter": "EU",
				"use_native_sdk": {"web": true, "android": true, "ios": false},
				"send_page_name_in_sdk": {"web": true},
				"data_use_in_app": {"web": false},
				"auto_track_device_attributes": {"android": true, "ios": true},
				"background_queue_min_number_of_tasks": {"android": "10"},
				"background_queue_seconds_delay": {"android": "30"}
			}`,
			APIJSON: `{
				"siteID": "site-id-1",
				"apiKey": "api-key-1",
				"deviceTokenEventName": "Device Token Registered",
				"datacenter": "EU",
				"useNativeSDK": {"web": true, "android": true, "ios": false},
				"sendPageNameInSDK": {"web": true},
				"dataUseInApp": {"web": false},
				"autoTrackDeviceAttributes": {"android": true, "ios": true},
				"backgroundQueueMinNumberOfTasks": {"android": "10"},
				"backgroundQueueSecondsDelay": {"android": "30"}
			}`,
		},
		{
			Name: "event filtering whitelist",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"datacenter": "US",
				"event_filtering": {"whitelist": ["Product Purchased", "Signed Up"]}
			}`,
			APIJSON: `{
				"siteID": "site-id-1",
				"apiKey": "api-key-1",
				"datacenter": "US",
				"whitelistedEvents": [{"eventName": "Product Purchased"}, {"eventName": "Signed Up"}],
				"eventFilteringOption": "whitelistedEvents"
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"datacenter": "US",
				"event_filtering": {"blacklist": ["Password Reset"]}
			}`,
			APIJSON: `{
				"siteID": "site-id-1",
				"apiKey": "api-key-1",
				"datacenter": "US",
				"blacklistedEvents": [{"eventName": "Password Reset"}],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"site_id": "site-id-1",
				"api_key": "api-key-1",
				"datacenter": "US",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"siteID": "site-id-1",
				"apiKey": "api-key-1",
				"datacenter": "US",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}

func TestCustomerioSecretHandling(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customerio.NewDefinition()))

	h := destination.NewHandler(nil, registry)

	extracted, err := h.Impl.ExtractResourcesFromSpec("destinations/customerio.yaml", &destination.DestinationSpec{
		ID:                "customerio-production",
		DisplayName:       "Customer.io Production",
		Type:              "customerio",
		Enabled:           true,
		DefinitionVersion: 1,
		Config: map[string]any{
			"site_id":    "site-id-1",
			"api_key":    "customerio-api-key",
			"datacenter": "US",
		},
	})
	require.NoError(t, err)

	resource := extracted["customerio-production"]
	require.NotNil(t, resource)
	assert.Equal(t, "site-id-1", resource.Config["site_id"])
	apiKey := assertWrappedSecret(t, resource.Config, "api_key", "customerio-api-key")
	assert.NotContains(t, fmt.Sprintf("%v", apiKey), "customerio-api-key")

	remote := &destination.RemoteDestination{Destination: &client.Destination{
		ID:         "dst-customerio",
		ExternalID: "customerio-production",
		Name:       "Customer.io Production",
		Type:       "CUSTOMERIO",
		Version:    1,
		IsEnabled:  true,
		Config:     []byte(`{"siteID":"site-id-1","apiKey":"","datacenter":"US"}`),
	}}

	remoteResource, _, err := h.Impl.MapRemoteToState(remote, urnResolver{})
	require.NoError(t, err)
	remoteAPIKey := requireSecret(t, remoteResource.Config, "api_key")
	assert.True(t, remoteAPIKey.IsUnknown())
	assert.Equal(t, "site-id-1", remoteResource.Config["site_id"])

	entities, _, err := h.Impl.FormatForExport(map[string]*destination.RemoteDestination{
		"customerio-production": {Destination: &client.Destination{
			ID:        "dst-customerio",
			Name:      "Customer.io Production",
			Type:      "CUSTOMERIO",
			Version:   1,
			IsEnabled: true,
			Config:    []byte(`{"siteID":"site-id-1","apiKey":"customerio-api-key","datacenter":"US"}`),
		}},
	}, nil, urnResolver{})
	require.NoError(t, err)
	require.Len(t, entities, 1)

	spec, ok := entities[0].Content.(*specs.Spec)
	require.True(t, ok)
	config, ok := spec.Spec["config"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "(unknown)", config["api_key"])
	assert.Equal(t, "site-id-1", config["site_id"])
}

func registeredCustomerioDefinition(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(customerio.NewDefinition()))
	registered, err := registry.Get("customerio", 1)
	require.NoError(t, err)
	return registered
}

func minimalConfig() map[string]any {
	return map[string]any{
		"site_id":    "site-id-1",
		"api_key":    "api-key-1",
		"datacenter": "US",
	}
}

func fullConfig() map[string]any {
	return map[string]any{
		"site_id":                              "site-id-1",
		"api_key":                              "api-key-1",
		"device_token_event_name":              "Device Token Registered",
		"datacenter":                           "EU",
		"use_native_sdk":                       map[string]any{"web": true, "android": true, "ios": false},
		"send_page_name_in_sdk":                map[string]any{"web": true},
		"data_use_in_app":                      map[string]any{"web": false},
		"auto_track_device_attributes":         map[string]any{"android": true, "ios": true},
		"background_queue_min_number_of_tasks": map[string]any{"android": "10"},
		"background_queue_seconds_delay":       map[string]any{"android": "30"},
		"event_filtering": map[string]any{
			"whitelist": []any{"Product Purchased", "Signed Up"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
			"warehouse": []any{
				map[string]any{
					"provider":            "custom",
					"resolution_strategy": "and",
					"consents":            []any{"marketing", "analytics"},
				},
			},
		},
	}
}

func exampleConfig() map[string]any {
	return map[string]any{
		"site_id":                 "cio-site-id",
		"api_key":                 "cio-api-key",
		"datacenter":              "US",
		"device_token_event_name": "Device Token Registered",
		"use_native_sdk":          map[string]any{"web": true, "android": true, "ios": true},
		"send_page_name_in_sdk":   map[string]any{"web": true},
		"data_use_in_app":         map[string]any{"web": false},
		"auto_track_device_attributes": map[string]any{
			"android": true,
			"ios":     true,
		},
		"background_queue_min_number_of_tasks": map[string]any{"android": "10"},
		"background_queue_seconds_delay":       map[string]any{"android": "30"},
		"event_filtering": map[string]any{
			"whitelist": []any{"Product Purchased", "Signed Up"},
		},
		"consent_management": map[string]any{
			"web": []any{
				map[string]any{
					"provider": "oneTrust",
					"consents": []any{"analytics"},
				},
			},
		},
	}
}

func assertValidationPaths(t *testing.T, errors []definitions.ConfigError, paths ...string) {
	t.Helper()

	byPath := map[string]struct{}{}
	for _, err := range errors {
		byPath[err.Path] = struct{}{}
	}
	for _, path := range paths {
		assert.Contains(t, byPath, path)
	}
}

func assertWrappedSecret(t *testing.T, config map[string]any, key string, want string) *secret.String {
	t.Helper()

	s := requireSecret(t, config, key)
	assert.False(t, s.IsUnknown())
	assert.Equal(t, want, s.Reveal())
	return s
}

func requireSecret(t *testing.T, config map[string]any, key string) *secret.String {
	t.Helper()

	v, ok := config[key]
	require.True(t, ok, "missing secret key %q", key)
	s, ok := v.(*secret.String)
	require.True(t, ok, "key %q: expected *secret.String, got %T", key, v)
	require.NotNil(t, s)
	return s
}

type urnResolver struct{}

func (urnResolver) GetURNByID(string, string) (string, error) {
	return "", resources.ErrRemoteResourceExternalIdNotFound
}

func (urnResolver) ResolveToReference(string, string) (string, error) {
	return "", resources.ErrRemoteResourceExternalIdNotFound
}
