package adobeanalytics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	adobeanalytics "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/adobe_analytics"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func TestNewDefinitionMetadata(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(adobeanalytics.NewDefinition()))

	registered, err := registry.Get("adobe_analytics", 1)
	require.NoError(t, err)

	assert.Equal(t, "adobe_analytics", registered.Type)
	assert.Equal(t, "ADOBE_ANALYTICS", registered.APIType)
	assert.Equal(t, int64(1), registered.Version)
	assert.Equal(t, []string{}, registered.SecretKeys())

	expectedSourceTypes := []string{
		"android", "android_kotlin", "ios", "ios_swift", "web",
		"unity", "react_native", "flutter", "cordova", "cloud",
	}
	assert.Equal(t, expectedSourceTypes, registered.SupportedSourceTypes())

	expectedModes := map[string][]string{
		"android":        {"cloud", "device"},
		"android_kotlin": {"cloud"},
		"ios":            {"cloud", "device"},
		"ios_swift":      {"cloud"},
		"web":            {"cloud", "device"},
		"unity":          {"cloud"},
		"react_native":   {"cloud"},
		"flutter":        {"cloud"},
		"cordova":        {"cloud"},
		"cloud":          {"cloud"},
	}
	for sourceType, want := range expectedModes {
		modes, err := registered.ConnectionModes(sourceType)
		require.NoError(t, err)
		assert.Equal(t, want, modes, sourceType)
	}

	assert.NotContains(t, registered.SupportedSourceTypes(), "amp")
	assert.NotContains(t, registered.SupportedSourceTypes(), "shopify")
	assert.NotContains(t, registered.SupportedSourceTypes(), "warehouse")

	assert.Empty(t, registered.GatedKeyPaths())

	byAPI, err := registry.GetByAPIType("ADOBE_ANALYTICS", 1)
	require.NoError(t, err)
	assert.Equal(t, registered, byAPI)
}

func TestAdobeAnalyticsConfigValidation(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(adobeanalytics.NewDefinition()))
	registered, err := registry.Get("adobe_analytics", 1)
	require.NoError(t, err)

	t.Run("missing report_suite_ids", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/report_suite_ids", errors[0].Path)
		assert.Contains(t, errors[0].Message, "required")
	})

	t.Run("valid minimal config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
		})
		assert.Empty(t, errors)
	})

	t.Run("valid full config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"tracking_server_url":           "metrics.example.com",
			"tracking_server_secure_url":    "smetrics.example.com",
			"report_suite_ids":              "rsid1,rsid2",
			"ssl_heartbeat":                 true,
			"heartbeat_tracking_server_url": "heartbeat.example.com",
			"use_utf8_charset":              true,
			"use_secure_server_side":        true,
			"proxy_normal_url":              "cdn.example.com/aa.js",
			"proxy_heartbeat_url":           "cdn.example.com/hb.js",
			"events_to_types": []any{
				map[string]any{"from": "Video Start", "to": "initHeartbeat"},
			},
			"marketing_cloud_org_id":       "1234567890ABC@AdobeOrg",
			"drop_visitor_id":              true,
			"timestamp_option":             "disabled",
			"timestamp_optional_reporting": false,
			"no_fallback_visitor_id":       false,
			"prefer_visitor_id":            false,
			"rudder_events_to_adobe_events": []any{
				map[string]any{"from": "Product Viewed", "to": "event1"},
			},
			"track_page_name": true,
			"context_data_mapping": []any{
				map[string]any{"from": "traits.email", "to": "email"},
			},
			"context_data_prefix":         "rudder_",
			"use_legacy_link_name":        true,
			"page_name_fallback_tostring": true,
			"mobile_event_mapping": []any{
				map[string]any{"from": "screen.name", "to": "pageName"},
			},
			"send_false_values": true,
			"e_var_mapping": []any{
				map[string]any{"from": "category", "to": "1"},
			},
			"hier_mapping": []any{
				map[string]any{"from": "path", "to": "1"},
			},
			"list_mapping": []any{
				map[string]any{"from": "tags", "to": "1"},
			},
			"list_delimiter": []any{
				map[string]any{"from": "tags", "to": ","},
			},
			"custom_props_mapping": []any{
				map[string]any{"from": "plan", "to": "1"},
			},
			"props_delimiter": []any{
				map[string]any{"from": "plan", "to": "|"},
			},
			"event_merch_event_to_adobe_event": []any{
				map[string]any{"from": "Order Completed", "to": "purchase"},
			},
			"event_merch_properties": []any{"revenue", "currency"},
			"product_merch_event_to_adobe_event": []any{
				map[string]any{"from": "Product Added", "to": "scAdd"},
			},
			"product_merch_properties": []any{"price", "quantity"},
			"product_merch_evars_map": []any{
				map[string]any{"from": "sku", "to": "1"},
			},
			"product_identifier": "sku",
			"event_filtering": map[string]any{
				"whitelist": []any{"Product Viewed", "Order Completed"},
			},
			"use_native_sdk": map[string]any{
				"web":          true,
				"ios":          true,
				"android":      true,
				"react_native": false,
			},
			"consent_management": map[string]any{
				"web": []any{
					map[string]any{
						"provider": "oneTrust",
						"consents": []any{"analytics"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("valid example yaml config", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"tracking_server_url":           "metrics.example.com",
			"tracking_server_secure_url":    "smetrics.example.com",
			"report_suite_ids":              "rsid1,rsid2",
			"ssl_heartbeat":                 true,
			"heartbeat_tracking_server_url": "heartbeat.example.com",
			"marketing_cloud_org_id":        "1234567890ABC@AdobeOrg",
			"drop_visitor_id":               true,
			"timestamp_option":              "disabled",
			"prefer_visitor_id":             false,
			"rudder_events_to_adobe_events": []any{
				map[string]any{"from": "Product Viewed", "to": "event1"},
			},
			"track_page_name": true,
			"context_data_mapping": []any{
				map[string]any{"from": "traits.email", "to": "email"},
			},
			"context_data_prefix": "rudder_",
			"e_var_mapping": []any{
				map[string]any{"from": "category", "to": "1"},
			},
			"product_identifier": "name",
			"event_filtering": map[string]any{
				"blacklist": []any{"Application Opened"},
			},
			"use_native_sdk": map[string]any{
				"web": true,
			},
			"consent_management": map[string]any{
				"android_kotlin": []any{
					map[string]any{
						"provider":            "oneTrust",
						"resolution_strategy": "and",
						"consents":            []any{"analytics", "marketing"},
					},
				},
			},
		})
		assert.Empty(t, errors)
	})

	t.Run("invalid timestamp_option rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
			"timestamp_option": "bogus",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/timestamp_option", errors[0].Path)
	})

	t.Run("invalid product_identifier rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids":   "rsid1",
			"product_identifier": "bogus",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/product_identifier", errors[0].Path)
	})

	t.Run("invalid events_to_types to rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
			"events_to_types": []any{
				map[string]any{"from": "Video Start", "to": "notAHeartbeat"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/events_to_types/0/to", errors[0].Path)
	})

	t.Run("invalid list_delimiter to rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
			"list_delimiter": []any{
				map[string]any{"from": "tags", "to": "#"},
			},
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/list_delimiter/0/to", errors[0].Path)
		assert.Contains(t, errors[0].Message, "must be one of")
	})

	t.Run("report_suite_ids max length rejected", func(t *testing.T) {
		t.Parallel()
		long := make([]byte, 301)
		for i := range long {
			long[i] = 'a'
		}
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": string(long),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/report_suite_ids", errors[0].Path)
		assert.Contains(t, errors[0].Message, "300")
	})

	t.Run("tracking_server_url with ngrok rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids":    "rsid1",
			"tracking_server_url": "https://example.ngrok.io",
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/tracking_server_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "ngrok")
	})

	t.Run("tracking_server_url too long rejected", func(t *testing.T) {
		t.Parallel()
		long := make([]byte, 101)
		for i := range long {
			long[i] = 'a'
		}
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids":    "rsid1",
			"tracking_server_url": string(long),
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/tracking_server_url", errors[0].Path)
		assert.Contains(t, errors[0].Message, "100")
	})

	t.Run("unknown key rejected", func(t *testing.T) {
		t.Parallel()
		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
			"not_a_field":      true,
		})
		require.NotEmpty(t, errors)
		assert.Equal(t, "/not_a_field", errors[0].Path)
		assert.Contains(t, errors[0].Message, "unknown config field")
	})

	t.Run("unsupported consent source rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
			"consent_management": map[string]any{
				"warehouse": []any{},
			},
		})

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/warehouse", errors[0].Path)
		assert.Contains(t, errors[0].Message, "source type 'warehouse' is not supported")
	})

	t.Run("invalid consent provider rejected", func(t *testing.T) {
		t.Parallel()

		errors := registered.ValidateConfig(map[string]any{
			"report_suite_ids": "rsid1",
			"consent_management": map[string]any{
				"ios_swift": []any{
					map[string]any{"provider": "unknown"},
				},
			},
		})

		require.Len(t, errors, 1)
		assert.Equal(t, "/consent_management/ios_swift/0/provider", errors[0].Path)
		assert.Contains(t, errors[0].Message, "'provider' must be one of")
	})
}

func TestAdobeAnalyticsConversionRoundTrip(t *testing.T) {
	t.Parallel()

	def := adobeanalytics.NewDefinition()
	testutil.AssertConversion(t, def.Properties, []testutil.ConversionCase{
		{
			Name: "minimal report suite ids only",
			LocalJSON: `{
				"report_suite_ids": "rsid1"
			}`,
			APIJSON: `{
				"reportSuiteIds": "rsid1"
			}`,
		},
		{
			Name: "full TF fields with whitelist",
			LocalJSON: `{
				"tracking_server_url": "metrics.example.com",
				"tracking_server_secure_url": "smetrics.example.com",
				"report_suite_ids": "rsid1,rsid2",
				"ssl_heartbeat": true,
				"heartbeat_tracking_server_url": "heartbeat.example.com",
				"use_utf8_charset": true,
				"use_secure_server_side": true,
				"proxy_normal_url": "cdn.example.com/aa.js",
				"proxy_heartbeat_url": "cdn.example.com/hb.js",
				"events_to_types": [
					{"from": "Video Start", "to": "initHeartbeat"}
				],
				"marketing_cloud_org_id": "1234567890ABC@AdobeOrg",
				"drop_visitor_id": true,
				"timestamp_option": "disabled",
				"timestamp_optional_reporting": false,
				"no_fallback_visitor_id": false,
				"prefer_visitor_id": false,
				"rudder_events_to_adobe_events": [
					{"from": "Product Viewed", "to": "event1"}
				],
				"track_page_name": true,
				"context_data_mapping": [
					{"from": "traits.email", "to": "email"}
				],
				"context_data_prefix": "rudder_",
				"use_legacy_link_name": true,
				"page_name_fallback_tostring": true,
				"mobile_event_mapping": [
					{"from": "screen.name", "to": "pageName"}
				],
				"send_false_values": true,
				"e_var_mapping": [
					{"from": "category", "to": "1"}
				],
				"hier_mapping": [
					{"from": "path", "to": "1"}
				],
				"list_mapping": [
					{"from": "tags", "to": "1"}
				],
				"list_delimiter": [
					{"from": "tags", "to": ","}
				],
				"custom_props_mapping": [
					{"from": "plan", "to": "1"}
				],
				"props_delimiter": [
					{"from": "plan", "to": "|"}
				],
				"event_merch_event_to_adobe_event": [
					{"from": "Order Completed", "to": "purchase"}
				],
				"event_merch_properties": ["revenue", "currency"],
				"product_merch_event_to_adobe_event": [
					{"from": "Product Added", "to": "scAdd"}
				],
				"product_merch_properties": ["price", "quantity"],
				"product_merch_evars_map": [
					{"from": "sku", "to": "1"}
				],
				"product_identifier": "sku",
				"event_filtering": {
					"whitelist": ["Product Viewed", "Order Completed"]
				},
				"use_native_sdk": {
					"web": true,
					"ios": true,
					"android": true,
					"react_native": false
				}
			}`,
			APIJSON: `{
				"trackingServerUrl": "metrics.example.com",
				"trackingServerSecureUrl": "smetrics.example.com",
				"reportSuiteIds": "rsid1,rsid2",
				"sslHeartbeat": true,
				"heartbeatTrackingServerUrl": "heartbeat.example.com",
				"useUtf8Charset": true,
				"useSecureServerSide": true,
				"proxyNormalUrl": "cdn.example.com/aa.js",
				"proxyHeartbeatUrl": "cdn.example.com/hb.js",
				"eventsToTypes": [
					{"from": "Video Start", "to": "initHeartbeat"}
				],
				"marketingCloudOrgId": "1234567890ABC@AdobeOrg",
				"dropVisitorId": true,
				"timestampOption": "disabled",
				"timestampOptionalReporting": false,
				"noFallbackVisitorId": false,
				"preferVisitorId": false,
				"rudderEventsToAdobeEvents": [
					{"from": "Product Viewed", "to": "event1"}
				],
				"trackPageName": true,
				"contextDataMapping": [
					{"from": "traits.email", "to": "email"}
				],
				"contextDataPrefix": "rudder_",
				"useLegacyLinkName": true,
				"pageNameFallbackTostring": true,
				"mobileEventMapping": [
					{"from": "screen.name", "to": "pageName"}
				],
				"sendFalseValues": true,
				"eVarMapping": [
					{"from": "category", "to": "1"}
				],
				"hierMapping": [
					{"from": "path", "to": "1"}
				],
				"listMapping": [
					{"from": "tags", "to": "1"}
				],
				"listDelimiter": [
					{"from": "tags", "to": ","}
				],
				"customPropsMapping": [
					{"from": "plan", "to": "1"}
				],
				"propsDelimiter": [
					{"from": "plan", "to": "|"}
				],
				"eventMerchEventToAdobeEvent": [
					{"from": "Order Completed", "to": "purchase"}
				],
				"eventMerchProperties": [
					{"eventMerchProperties": "revenue"},
					{"eventMerchProperties": "currency"}
				],
				"productMerchEventToAdobeEvent": [
					{"from": "Product Added", "to": "scAdd"}
				],
				"productMerchProperties": [
					{"productMerchProperties": "price"},
					{"productMerchProperties": "quantity"}
				],
				"productMerchEvarsMap": [
					{"from": "sku", "to": "1"}
				],
				"productIdentifier": "sku",
				"whitelistedEvents": [
					{"eventName": "Product Viewed"},
					{"eventName": "Order Completed"}
				],
				"eventFilteringOption": "whitelistedEvents",
				"useNativeSDK": {
					"web": true,
					"ios": true,
					"android": true,
					"reactnative": false
				}
			}`,
		},
		{
			Name: "event filtering blacklist reshape",
			LocalJSON: `{
				"report_suite_ids": "rsid1",
				"event_filtering": {
					"blacklist": ["Application Opened"]
				}
			}`,
			APIJSON: `{
				"reportSuiteIds": "rsid1",
				"blacklistedEvents": [
					{"eventName": "Application Opened"}
				],
				"eventFilteringOption": "blacklistedEvents"
			}`,
		},
		{
			Name: "mapping arrays reshape",
			LocalJSON: `{
				"report_suite_ids": "rsid1",
				"e_var_mapping": [
					{"from": "category", "to": "1"}
				],
				"event_merch_properties": ["revenue"]
			}`,
			APIJSON: `{
				"reportSuiteIds": "rsid1",
				"eVarMapping": [
					{"from": "category", "to": "1"}
				],
				"eventMerchProperties": [
					{"eventMerchProperties": "revenue"}
				]
			}`,
		},
		{
			Name: "consent source boundary mappings",
			LocalJSON: `{
				"report_suite_ids": "rsid1",
				"consent_management": {
					"android_kotlin": [{"provider": "oneTrust"}],
					"ios_swift": [{"provider": "ketch"}],
					"react_native": [{"provider": "iubenda"}]
				}
			}`,
			APIJSON: `{
				"reportSuiteIds": "rsid1",
				"consentManagement": {
					"androidKotlin": [{"provider": "oneTrust"}],
					"iosSwift": [{"provider": "ketch"}],
					"reactnative": [{"provider": "iubenda"}]
				}
			}`,
		},
	})
}
