package am_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(am.NewDefinition()))

	registered, err := registry.Get("am", 1)
	require.NoError(t, err)

	assert.Equal(t, "am", registered.Type)
	assert.Equal(t, "AM", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{"api_secret"}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "amp", "cloud", "warehouse", "react_native", "flutter", "cordova", "shopify",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud", "device"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud", "device"},
		"ios_swift":      {"cloud"},
		"web":            {"cloud", "device"},
		"unity":          {"cloud"},
		"cloud":          {"cloud"},
		"react_native":   {"cloud", "device"},
		"flutter":        {"cloud", "device"},
		"cordova":        {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, "source type %s", sourceType)
	}
	assert.Equal(t, []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "cloud", "react_native", "flutter", "cordova",
	}, registered.ConnectionModeSourceTypeKeys())

	for _, sourceType := range []string{"amp", "warehouse", "shopify"} {
		_, err := registered.ConnectionModes(sourceType)
		require.Error(t, err)
	}

	assert.Equal(t, map[string]any{
		"track_all_pages":                    false,
		"track_categorized_pages":            true,
		"track_named_pages":                  true,
		"use_user_defined_page_event_name":   false,
		"enable_enhanced_user_operations":    false,
		"map_device_brand":                   false,
		"track_products_once":                false,
		"track_revenue_per_product":          false,
		"use_user_defined_screen_event_name": false,
		"residency_server":                   "standard",
		"sdk_version":                        map[string]any{"web": float64(2)},
		"track_session_events":               map[string]any{"web": false},
		"auto_capture": map[string]any{
			"page_views":               map[string]any{"web": false},
			"page_url_enrichment":      map[string]any{"web": false},
			"web_vitals":               map[string]any{"web": false},
			"file_downloads":           map[string]any{"web": false},
			"frustration_interactions": map[string]any{"web": false},
			"network_tracking":         map[string]any{"web": false},
			"element_interactions":     map[string]any{"web": false},
			"form_interactions":        map[string]any{"web": false},
		},
	}, registered.ConfigDefaults())

	assert.NotContains(t,
		registered.ApplyDefaults(map[string]any{"api_key": "amplitude-api-key"}),
		"sdk_version")
	assert.Equal(t,
		map[string]any{"web": float64(2)},
		registered.ApplyDefaults(map[string]any{
			"api_key":     "amplitude-api-key",
			"sdk_version": map[string]any{},
		})["sdk_version"])

	assert.Equal(t, map[string][]string{
		"sdk_version/web":                               {"web"},
		"proxy_server_url/web":                          {"web"},
		"prefer_anonymous_id_for_device_id/web":         {"web"},
		"device_id_from_url_param/web":                  {"web"},
		"force_https/web":                               {"web"},
		"track_gclid/web":                               {"web"},
		"track_referrer/web":                            {"web"},
		"save_params_referrer_once_per_session/web":     {"web"},
		"track_utm_properties/web":                      {"web"},
		"unset_params_referrer_on_new_session/web":      {"web"},
		"batch_events/web":                              {"web"},
		"attribution/web":                               {"web"},
		"track_new_campaigns/web":                       {"web"},
		"auto_capture/page_views/web":                   {"web"},
		"auto_capture/page_url_enrichment/web":          {"web"},
		"auto_capture/web_vitals/web":                   {"web"},
		"auto_capture/file_downloads/web":               {"web"},
		"auto_capture/frustration_interactions/web":     {"web"},
		"auto_capture/network_tracking/web":             {"web"},
		"auto_capture/element_interactions/web":         {"web"},
		"auto_capture/form_interactions/web":            {"web"},
		"event_upload_period_millis/web":                {"web"},
		"event_upload_period_millis/android":            {"android"},
		"event_upload_period_millis/ios":                {"ios"},
		"event_upload_period_millis/react_native":       {"react_native"},
		"event_upload_period_millis/flutter":            {"flutter"},
		"event_upload_threshold/web":                    {"web"},
		"event_upload_threshold/android":                {"android"},
		"event_upload_threshold/ios":                    {"ios"},
		"event_upload_threshold/react_native":           {"react_native"},
		"event_upload_threshold/flutter":                {"flutter"},
		"enable_location_listening/android":             {"android"},
		"enable_location_listening/react_native":        {"react_native"},
		"enable_location_listening/flutter":             {"flutter"},
		"track_session_events/web":                      {"web"},
		"track_session_events/android":                  {"android"},
		"track_session_events/ios":                      {"ios"},
		"track_session_events/react_native":             {"react_native"},
		"track_session_events/flutter":                  {"flutter"},
		"use_advertising_id_for_device_id/android":      {"android"},
		"use_advertising_id_for_device_id/react_native": {"react_native"},
		"use_advertising_id_for_device_id/flutter":      {"flutter"},
		"use_idfa_as_device_id/ios":                     {"ios"},
		"use_idfa_as_device_id/react_native":            {"react_native"},
		"use_idfa_as_device_id/flutter":                 {"flutter"},
	}, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("AM", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestAmplitudeConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(am.NewDefinition()))
	registered, err := registry.Get("am", 1)
	require.NoError(t, err)

	validMinimal := func() map[string]any {
		return map[string]any{
			"api_key":          "amplitude-api-key",
			"residency_server": "standard",
		}
	}

	t.Run("missing api_key", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"residency_server": "standard",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("defaulted minimal config is valid", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(registered.ApplyDefaults(map[string]any{
			"api_key": "amplitude-api-key",
		})))
	})

	t.Run("invalid api_key pattern rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":          "line\nbreak",
			"residency_server": "standard",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/api_key", errors[0].Path)
	})

	t.Run("api_key template accepted", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":          "{{ .AMPLITUDE_API_KEY || fallback }}",
			"residency_server": "standard",
		})
		assert.Empty(t, errors)
	})

	t.Run("invalid residency_server rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":          "amplitude-api-key",
			"residency_server": "us",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/residency_server", errors[0].Path)
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, registered.ValidateConfig(validMinimal()))
	})

	t.Run("valid full config example", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"api_key":                            "amplitude-api-key",
			"api_secret":                         "{{ .AMPLITUDE_API_SECRET || fallback }}",
			"residency_server":                   "EU",
			"group_type_trait":                   "company_id",
			"group_value_trait":                  "company_name",
			"track_all_pages":                    true,
			"track_categorized_pages":            true,
			"track_named_pages":                  false,
			"use_user_defined_page_event_name":   true,
			"user_provided_page_event_string":    "Viewed {{ name }}",
			"traits_to_increment":                []any{"lifetime_value"},
			"traits_to_set_once":                 []any{"first_seen_at"},
			"traits_to_append":                   []any{"plan"},
			"traits_to_prepend":                  []any{"segment"},
			"enable_enhanced_user_operations":    true,
			"version_name":                       "1.2.3",
			"map_device_brand":                   true,
			"track_products_once":                true,
			"track_revenue_per_product":          true,
			"use_user_defined_screen_event_name": true,
			"user_provided_screen_event_string":  "Viewed {{ name }} Screen",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"sdk_version": map[string]any{"web": 2},
			"proxy_server_url": map[string]any{
				"web": "https://amplitude-proxy.example.com",
			},
			"prefer_anonymous_id_for_device_id":     map[string]any{"web": true},
			"device_id_from_url_param":              map[string]any{"web": true},
			"force_https":                           map[string]any{"web": true},
			"track_gclid":                           map[string]any{"web": true},
			"track_referrer":                        map[string]any{"web": true},
			"save_params_referrer_once_per_session": map[string]any{"web": true},
			"track_utm_properties":                  map[string]any{"web": true},
			"unset_params_referrer_on_new_session":  map[string]any{"web": true},
			"batch_events":                          map[string]any{"web": true},
			"attribution":                           map[string]any{"web": false},
			"track_new_campaigns":                   map[string]any{"web": true},
			"auto_capture": map[string]any{
				"page_views":               map[string]any{"web": true},
				"page_url_enrichment":      map[string]any{"web": true},
				"web_vitals":               map[string]any{"web": false},
				"file_downloads":           map[string]any{"web": true},
				"frustration_interactions": map[string]any{"web": false},
				"network_tracking":         map[string]any{"web": false},
				"element_interactions":     map[string]any{"web": false},
				"form_interactions":        map[string]any{"web": false},
			},
			"event_upload_period_millis": map[string]any{
				"web":          "1000",
				"android":      "1000",
				"ios":          "1000",
				"react_native": "1000",
				"flutter":      "1000",
			},
			"event_upload_threshold": map[string]any{
				"web":          "30",
				"android":      "30",
				"ios":          "30",
				"react_native": "30",
				"flutter":      "30",
			},
			"enable_location_listening": map[string]any{
				"android":      true,
				"react_native": true,
				"flutter":      true,
			},
			"track_session_events": map[string]any{
				"web":          true,
				"android":      true,
				"ios":          true,
				"react_native": true,
				"flutter":      true,
			},
			"use_advertising_id_for_device_id": map[string]any{
				"android":      false,
				"react_native": false,
				"flutter":      false,
			},
			"use_idfa_as_device_id": map[string]any{
				"ios":          false,
				"react_native": false,
				"flutter":      false,
			},
			"use_native_sdk": map[string]any{
				"web":          true,
				"android":      true,
				"ios":          true,
				"react_native": true,
				"flutter":      true,
			},
			"connection_mode": map[string]any{
				"web":            "device",
				"android":        "device",
				"android_kotlin": "cloud",
				"ios":            "device",
				"ios_swift":      "cloud",
				"unity":          "cloud",
				"cloud":          "cloud",
				"react_native":   "device",
				"flutter":        "device",
				"cordova":        "cloud",
			},
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{
						"provider":            "oneTrust",
						"resolution_strategy": "and",
						"consents":            []any{"analytics"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("sdk_version.web must be 1 or 2", func(t *testing.T) {
		t.Parallel()
		config := validMinimal()
		config["sdk_version"] = map[string]any{"web": 3}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/sdk_version/web", errors[0].Path)
	})

	t.Run("proxy_server_url rejects http and ngrok", func(t *testing.T) {
		t.Parallel()
		for _, url := range []string{"http://proxy.example.com", "https://foo.ngrok.io", "https://foo.ngrok.io:443"} {
			config := validMinimal()
			config["proxy_server_url"] = map[string]any{"web": url}
			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors, url)
			assert.Equal(t, "/proxy_server_url/web", errors[0].Path, url)
		}
	})

	t.Run("proxy_server_url accepts https and templates", func(t *testing.T) {
		t.Parallel()
		for _, url := range []string{"https://proxy.example.com", "{{ .AMPLITUDE_PROXY_URL || https://proxy.example.com }}"} {
			config := validMinimal()
			config["proxy_server_url"] = map[string]any{"web": url}
			assert.Empty(t, registered.ValidateConfig(config), url)
		}
	})

	t.Run("numeric upload values reject non digits and accept templates", func(t *testing.T) {
		t.Parallel()

		config := validMinimal()
		config["event_upload_threshold"] = map[string]any{"web": "30s"}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/event_upload_threshold/web", errors[0].Path)

		config = validMinimal()
		config["event_upload_period_millis"] = map[string]any{"web": "{{ .AMPLITUDE_UPLOAD_PERIOD || 1000 }}"}
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("single line 200 fields reject long values and accept templates", func(t *testing.T) {
		t.Parallel()

		config := validMinimal()
		config["user_provided_page_event_string"] = strings.Repeat("a", 201)
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/user_provided_page_event_string", errors[0].Path)

		config = validMinimal()
		config["user_provided_screen_event_string"] = "{{ .AMPLITUDE_SCREEN_EVENT_NAME || Viewed Screen }}"
		assert.Empty(t, registered.ValidateConfig(config))
	})

	t.Run("event filtering lists are mutually exclusive", func(t *testing.T) {
		t.Parallel()
		config := validMinimal()
		config["event_filtering"] = map[string]any{
			"whitelist": []any{"Order Completed"},
			"blacklist": []any{"Page Viewed"},
		}
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
	})

	t.Run("connection_mode value validated per source type", func(t *testing.T) {
		t.Parallel()
		config := validMinimal()
		config["connection_mode"] = map[string]any{
			"web":   "device",
			"unity": "device",
		}
		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/connection_mode/unity", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("connection_mode rejects unsupported source keys without dropping source support", func(t *testing.T) {
		t.Parallel()
		for _, sourceType := range []string{"amp", "warehouse", "shopify"} {
			config := validMinimal()
			config["connection_mode"] = map[string]any{sourceType: "cloud"}

			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors, sourceType)
			assert.Equal(t, "/connection_mode/"+sourceType, errors[0].Path)
			assert.Contains(t, errors[0].Message, "not supported under connection_mode")
		}
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimal()
		config["not_a_field"] = true
		errors := registered.ValidateConfig(config)
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("legacy consent include-key blocks rejected", func(t *testing.T) {
		t.Parallel()
		for _, key := range []string{"one_trust_cookie_categories", "ketch_consent_purposes"} {
			config := validMinimal()
			config[key] = map[string]any{"web": []any{"analytics"}}
			errors := registered.ValidateConfig(config)
			require.NotEmpty(t, errors, key)
			assert.Equal(t, "/"+key, errors[0].Path)
			assert.Contains(t, errors[0].Message, "unknown config field")
		}
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()
		config := validMinimal()
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
		config := validMinimal()
		config["consent_management"] = map[string]any{
			"android_kotlin": []any{
				map[string]any{"provider": "unknown"},
			},
		}
		errors := registered.ValidateConfig(config)
		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/android_kotlin/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestAmplitudeConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := am.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal",
			LocalJSON: `{
				"api_key": "amplitude-api-key",
				"residency_server": "standard"
			}`,
			APIJSON: `{
				"apiKey": "amplitude-api-key",
				"residencyServer": "standard"
			}`,
		},
		{
			Name: "full config",
			LocalJSON: `{
				"api_key": "amplitude-api-key",
				"api_secret": "amplitude-secret",
				"residency_server": "EU",
				"group_type_trait": "company_id",
				"group_value_trait": "company_name",
				"track_all_pages": true,
				"track_categorized_pages": false,
				"track_named_pages": true,
				"use_user_defined_page_event_name": true,
				"user_provided_page_event_string": "Viewed {{ name }}",
				"traits_to_increment": ["lifetime_value"],
				"traits_to_set_once": ["first_seen_at"],
				"traits_to_append": ["plan"],
				"traits_to_prepend": ["segment"],
				"enable_enhanced_user_operations": true,
				"track_products_once": true,
				"track_revenue_per_product": false,
				"version_name": "1.2.3",
				"map_device_brand": true,
				"use_user_defined_screen_event_name": true,
				"user_provided_screen_event_string": "Viewed {{ name }} Screen",
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"sdk_version": {"web": 2},
				"proxy_server_url": {"web": "https://amplitude-proxy.example.com"},
				"prefer_anonymous_id_for_device_id": {"web": true},
				"device_id_from_url_param": {"web": true},
				"force_https": {"web": true},
				"track_gclid": {"web": true},
				"track_referrer": {"web": true},
				"save_params_referrer_once_per_session": {"web": true},
				"track_utm_properties": {"web": true},
				"unset_params_referrer_on_new_session": {"web": true},
				"batch_events": {"web": true},
				"attribution": {"web": false},
				"track_new_campaigns": {"web": true},
				"auto_capture": {
					"page_views": {"web": true},
					"page_url_enrichment": {"web": true},
					"web_vitals": {"web": false},
					"file_downloads": {"web": true},
					"frustration_interactions": {"web": false},
					"network_tracking": {"web": false},
					"element_interactions": {"web": false},
					"form_interactions": {"web": false}
				},
				"event_upload_period_millis": {
					"web": "1000",
					"android": "1000",
					"ios": "1000",
					"react_native": "1000",
					"flutter": "1000"
				},
				"event_upload_threshold": {
					"web": "30",
					"android": "30",
					"ios": "30",
					"react_native": "30",
					"flutter": "30"
				},
				"enable_location_listening": {
					"android": true,
					"react_native": true,
					"flutter": true
				},
				"track_session_events": {
					"web": true,
					"android": true,
					"ios": true,
					"react_native": true,
					"flutter": true
				},
				"use_advertising_id_for_device_id": {
					"android": false,
					"react_native": false,
					"flutter": false
				},
				"use_idfa_as_device_id": {
					"ios": false,
					"react_native": false,
					"flutter": false
				},
				"use_native_sdk": {
					"web": true,
					"android": true,
					"ios": true,
					"react_native": true,
					"flutter": true
				},
				"connection_mode": {
					"web": "device",
					"android": "device",
					"android_kotlin": "cloud",
					"ios": "device",
					"ios_swift": "cloud",
					"unity": "cloud",
					"cloud": "cloud",
					"react_native": "device",
					"flutter": "device",
					"cordova": "cloud"
				},
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"web": [{"provider": "custom", "resolution_strategy": "and", "consents": ["analytics", "marketing"]}]
				}
			}`,
			APIJSON: `{
				"apiKey": "amplitude-api-key",
				"apiSecret": "amplitude-secret",
				"residencyServer": "EU",
				"groupTypeTrait": "company_id",
				"groupValueTrait": "company_name",
				"trackAllPages": true,
				"trackCategorizedPages": false,
				"trackNamedPages": true,
				"useUserDefinedPageEventName": true,
				"userProvidedPageEventString": "Viewed {{ name }}",
				"traitsToIncrement": [{"traits": "lifetime_value"}],
				"traitsToSetOnce": [{"traits": "first_seen_at"}],
				"traitsToAppend": [{"traits": "plan"}],
				"traitsToPrepend": [{"traits": "segment"}],
				"enableEnhancedUserOperations": true,
				"trackProductsOnce": true,
				"trackRevenuePerProduct": false,
				"versionName": "1.2.3",
				"mapDeviceBrand": true,
				"useUserDefinedScreenEventName": true,
				"userProvidedScreenEventString": "Viewed {{ name }} Screen",
				"whitelistedEvents": [{"eventName": "Product Viewed"}, {"eventName": "Order Completed"}],
				"eventFilteringOption": "whitelistedEvents",
				"sdkVersion": {"web": 2},
				"proxyServerUrl": {"web": "https://amplitude-proxy.example.com"},
				"preferAnonymousIdForDeviceId": {"web": true},
				"deviceIdFromUrlParam": {"web": true},
				"forceHttps": {"web": true},
				"trackGclid": {"web": true},
				"trackReferrer": {"web": true},
				"saveParamsReferrerOncePerSession": {"web": true},
				"trackUtmProperties": {"web": true},
				"unsetParamsReferrerOnNewSession": {"web": true},
				"batchEvents": {"web": true},
				"attribution": {"web": false},
				"trackNewCampaigns": {"web": true},
				"enablePageViewsAutoCapture": {"web": true},
				"enablePageUrlEnrichmentAutoCapture": {"web": true},
				"enableWebVitalsAutoCapture": {"web": false},
				"enableFileDownloadsAutoCapture": {"web": true},
				"enableFrustrationInteractionsAutoCapture": {"web": false},
				"enableNetworkTrackingAutoCapture": {"web": false},
				"enableElementInteractionsAutoCapture": {"web": false},
				"enableFormInteractionsAutoCapture": {"web": false},
				"eventUploadPeriodMillis": {
					"web": "1000",
					"android": "1000",
					"ios": "1000",
					"reactnative": "1000",
					"flutter": "1000"
				},
				"eventUploadThreshold": {
					"web": "30",
					"android": "30",
					"ios": "30",
					"reactnative": "30",
					"flutter": "30"
				},
				"enableLocationListening": {
					"android": true,
					"reactnative": true,
					"flutter": true
				},
				"trackSessionEvents": {
					"web": true,
					"android": true,
					"ios": true,
					"reactnative": true,
					"flutter": true
				},
				"useAdvertisingIdForDeviceId": {
					"android": false,
					"reactnative": false,
					"flutter": false
				},
				"useIdfaAsDeviceId": {
					"ios": false,
					"reactnative": false,
					"flutter": false
				},
				"useNativeSDK": {
					"web": true,
					"android": true,
					"ios": true,
					"reactnative": true,
					"flutter": true
				},
				"connectionMode": {
					"web": "device",
					"android": "device",
					"androidKotlin": "cloud",
					"ios": "device",
					"iosSwift": "cloud",
					"unity": "cloud",
					"cloud": "cloud",
					"reactnative": "device",
					"flutter": "device",
					"cordova": "cloud"
				},
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"web": [{"provider": "custom", "resolutionStrategy": "and", "consents": [{"consent": "analytics"}, {"consent": "marketing"}]}]
				}
			}`,
		},
		{
			Name: "event filtering blacklist",
			LocalJSON: `{
				"api_key": "amplitude-api-key",
				"residency_server": "standard",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"apiKey": "amplitude-api-key",
				"residencyServer": "standard",
				"blacklistedEvents": [{"eventName": "Application Opened"}],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"api_key": "amplitude-api-key",
				"residency_server": "standard",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}],
					"shopify": [{"provider": "custom", "resolution_strategy": "or", "consents": ["marketing"]}]
				}
			}`,
			APIJSON: `{
				"apiKey": "amplitude-api-key",
				"residencyServer": "standard",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}],
					"shopify": [{"provider": "custom", "resolutionStrategy": "or", "consents": [{"consent": "marketing"}]}]
				}
			}`,
		},
	})
}
