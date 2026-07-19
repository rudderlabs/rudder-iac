package adobeanalytics

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/adobe_analytics/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeCloud,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

type mappingEntry struct {
	From string `mapstructure:"from" validate:"required,max=100"`
	To   string `mapstructure:"to" validate:"required,max=100"`
}

type eventToTypeEntry struct {
	From string `mapstructure:"from" validate:"required,max=100"`
	To   string `mapstructure:"to" validate:"required,dynamic_or_oneof=initHeartbeat heartbeatPlaybackStarted heartbeatPlaybackPaused heartbeatPlaybackResumed heartbeatPlaybackCompleted heartbeatPlaybackInterrupted heartbeatContentStarted heartbeatContentComplete heartbeatAdBreakStarted heartbeatAdBreakCompleted heartbeatAdStarted heartbeatAdCompleted heartbeatAdSkipped heartbeatSeekStarted heartbeatSeekCompleted heartbeatBufferStarted heartbeatBufferCompleted heartbeatQualityUpdated heartbeatUpdatePlayhead"`
}

type delimiterEntry struct {
	From string `mapstructure:"from" validate:"required,max=100"`
	To   string `mapstructure:"to" validate:"required,pattern=adobe_analytics_delimiter"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,dive,max=100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,dive,max=100"`
}

type useNativeSDK struct {
	Web         *bool `mapstructure:"web"`
	IOS         *bool `mapstructure:"ios"`
	Android     *bool `mapstructure:"android"`
	ReactNative *bool `mapstructure:"react_native"`
}

// adobeAnalyticsConfig is the local YAML config model. Field set mirrors
// terraform-provider destination_adobe_analytics.go; validation constraints
// mirror overlapping schema.json rules for those mapped fields.
type adobeAnalyticsConfig struct {
	TrackingServerUrl             string                   `mapstructure:"tracking_server_url" validate:"omitempty,max=100"`
	TrackingServerSecureUrl       string                   `mapstructure:"tracking_server_secure_url" validate:"omitempty,max=100"`
	ReportSuiteIDs                string                   `mapstructure:"report_suite_ids" validate:"required,min=1,max=300"`
	SSLHeartbeat                  *bool                    `mapstructure:"ssl_heartbeat"`
	HeartbeatTrackingServerUrl    string                   `mapstructure:"heartbeat_tracking_server_url" validate:"omitempty,max=100"`
	UseUTF8Charset                *bool                    `mapstructure:"use_utf8_charset"`
	UseSecureServerSide           *bool                    `mapstructure:"use_secure_server_side"`
	ProxyNormalUrl                string                   `mapstructure:"proxy_normal_url" validate:"omitempty,max=100"`
	ProxyHeartbeatUrl             string                   `mapstructure:"proxy_heartbeat_url" validate:"omitempty,max=100"`
	EventsToTypes                 []eventToTypeEntry       `mapstructure:"events_to_types" validate:"omitempty,dive"`
	MarketingCloudOrgID           string                   `mapstructure:"marketing_cloud_org_id" validate:"omitempty,max=100"`
	DropVisitorID                 *bool                    `mapstructure:"drop_visitor_id"`
	TimestampOption               string                   `mapstructure:"timestamp_option" validate:"omitempty,dynamic_or_oneof=disabled hybrid optional enabled"`
	TimestampOptionalReporting    *bool                    `mapstructure:"timestamp_optional_reporting"`
	NoFallbackVisitorID           *bool                    `mapstructure:"no_fallback_visitor_id"`
	PreferVisitorID               *bool                    `mapstructure:"prefer_visitor_id"`
	RudderEventsToAdobeEvents     []mappingEntry           `mapstructure:"rudder_events_to_adobe_events" validate:"omitempty,dive"`
	TrackPageName                 *bool                    `mapstructure:"track_page_name"`
	ContextDataMapping            []mappingEntry           `mapstructure:"context_data_mapping" validate:"omitempty,dive"`
	ContextDataPrefix             string                   `mapstructure:"context_data_prefix" validate:"omitempty,max=100"`
	UseLegacyLinkName             *bool                    `mapstructure:"use_legacy_link_name"`
	PageNameFallbackToString      *bool                    `mapstructure:"page_name_fallback_tostring"`
	MobileEventMapping            []mappingEntry           `mapstructure:"mobile_event_mapping" validate:"omitempty,dive"`
	SendFalseValues               *bool                    `mapstructure:"send_false_values"`
	EVarMapping                   []mappingEntry           `mapstructure:"e_var_mapping" validate:"omitempty,dive"`
	HierMapping                   []mappingEntry           `mapstructure:"hier_mapping" validate:"omitempty,dive"`
	ListMapping                   []mappingEntry           `mapstructure:"list_mapping" validate:"omitempty,dive"`
	ListDelimiter                 []delimiterEntry         `mapstructure:"list_delimiter" validate:"omitempty,dive"`
	CustomPropsMapping            []mappingEntry           `mapstructure:"custom_props_mapping" validate:"omitempty,dive"`
	PropsDelimiter                []delimiterEntry         `mapstructure:"props_delimiter" validate:"omitempty,dive"`
	EventMerchEventToAdobeEvent   []mappingEntry           `mapstructure:"event_merch_event_to_adobe_event" validate:"omitempty,dive"`
	EventMerchProperties          []string                 `mapstructure:"event_merch_properties" validate:"omitempty,dive,max=100"`
	ProductMerchEventToAdobeEvent []mappingEntry           `mapstructure:"product_merch_event_to_adobe_event" validate:"omitempty,dive"`
	ProductMerchProperties        []string                 `mapstructure:"product_merch_properties" validate:"omitempty,dive,max=100"`
	ProductMerchEvarsMap          []mappingEntry           `mapstructure:"product_merch_evars_map" validate:"omitempty,dive"`
	ProductIdentifier             string                   `mapstructure:"product_identifier" validate:"omitempty,dynamic_or_oneof=name id sku"`
	EventFiltering                *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK                  *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConsentManagement             common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Adobe Analytics destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("trackingServerUrl", "tracking_server_url", converter.SkipZeroValue),
		converter.Simple("trackingServerSecureUrl", "tracking_server_secure_url", converter.SkipZeroValue),
		converter.Simple("reportSuiteIds", "report_suite_ids"),
		converter.Simple("sslHeartbeat", "ssl_heartbeat"),
		converter.Simple("heartbeatTrackingServerUrl", "heartbeat_tracking_server_url", converter.SkipZeroValue),
		converter.Simple("useUtf8Charset", "use_utf8_charset"),
		converter.Simple("useSecureServerSide", "use_secure_server_side"),
		converter.Simple("proxyNormalUrl", "proxy_normal_url", converter.SkipZeroValue),
		converter.Simple("proxyHeartbeatUrl", "proxy_heartbeat_url", converter.SkipZeroValue),
		converter.ArrayWithObjects("eventsToTypes", "events_to_types", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("marketingCloudOrgId", "marketing_cloud_org_id", converter.SkipZeroValue),
		converter.Simple("dropVisitorId", "drop_visitor_id"),
		converter.Simple("timestampOption", "timestamp_option"),
		converter.Simple("timestampOptionalReporting", "timestamp_optional_reporting"),
		converter.Simple("noFallbackVisitorId", "no_fallback_visitor_id"),
		converter.Simple("preferVisitorId", "prefer_visitor_id"),
		converter.ArrayWithObjects("rudderEventsToAdobeEvents", "rudder_events_to_adobe_events", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("trackPageName", "track_page_name"),
		converter.ArrayWithObjects("contextDataMapping", "context_data_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("contextDataPrefix", "context_data_prefix", converter.SkipZeroValue),
		converter.Simple("useLegacyLinkName", "use_legacy_link_name"),
		converter.Simple("pageNameFallbackTostring", "page_name_fallback_tostring"),
		converter.ArrayWithObjects("mobileEventMapping", "mobile_event_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("sendFalseValues", "send_false_values"),
		converter.ArrayWithObjects("eVarMapping", "e_var_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("hierMapping", "hier_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("listMapping", "list_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("listDelimiter", "list_delimiter", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("customPropsMapping", "custom_props_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("propsDelimiter", "props_delimiter", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("eventMerchEventToAdobeEvent", "event_merch_event_to_adobe_event", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithStrings("eventMerchProperties", "eventMerchProperties", "event_merch_properties"),
		converter.ArrayWithObjects("productMerchEventToAdobeEvent", "product_merch_event_to_adobe_event", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithStrings("productMerchProperties", "productMerchProperties", "product_merch_properties"),
		converter.ArrayWithObjects("productMerchEvarsMap", "product_merch_evars_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("productIdentifier", "product_identifier"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.reactnative", "use_native_sdk.react_native"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "adobe_analytics",
		APIType:    "ADOBE_ANALYTICS",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &adobeAnalyticsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
