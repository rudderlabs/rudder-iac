package am

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json proxyServerUrl.web forbids http:// and .ngrok.io using negative lookaheads.
	funcs.NewPatternWithReject(
		"am_proxy_server_url",
		`^.*$`,
		`^http://|^.*\.ngrok\.io.*$`,
		"must not start with http:// and must not contain .ngrok.io",
	)

	funcs.NewPattern(
		"am_numeric_string_100",
		`^([0-9]{0,100})$`,
		"must contain only digits and be at most 100 characters",
	)
}

// Source types from integrations-config destinations/am/db-config.json, minus
// amp, warehouse and shopify: the CLI maps those tokens but cannot produce
// them, since an event stream source's type is constrained to the SDK
// definitions and SourceTypeToken only reaches warehouse through a source
// category the sole call site never sets. Declaring them would advertise
// support no connection could ever match.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeCloud,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud", "device"},
	common.SourceTypeFlutter:       {"cloud", "device"},
	common.SourceTypeCordova:       {"cloud"},
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type sdkVersion struct {
	Web *int `mapstructure:"web" validate:"omitempty,oneof=1 2" default:"2"`
}

type proxyServerURL struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_pattern=am_proxy_server_url"`
}

type mobileBool struct {
	Android     *bool `mapstructure:"android"`
	ReactNative *bool `mapstructure:"react_native"`
	Flutter     *bool `mapstructure:"flutter"`
}

type mobileString struct {
	Web         string `mapstructure:"web" validate:"omitempty,dynamic_or_pattern=am_numeric_string_100"`
	Android     string `mapstructure:"android" validate:"omitempty,dynamic_or_pattern=am_numeric_string_100"`
	IOS         string `mapstructure:"ios" validate:"omitempty,dynamic_or_pattern=am_numeric_string_100"`
	ReactNative string `mapstructure:"react_native" validate:"omitempty,dynamic_or_pattern=am_numeric_string_100"`
	Flutter     string `mapstructure:"flutter" validate:"omitempty,dynamic_or_pattern=am_numeric_string_100"`
}

type sdkBools struct {
	Web         *bool `mapstructure:"web"`
	IOS         *bool `mapstructure:"ios"`
	Android     *bool `mapstructure:"android"`
	ReactNative *bool `mapstructure:"react_native"`
	Flutter     *bool `mapstructure:"flutter"`
}

type trackSessionEvents struct {
	Web         *bool `mapstructure:"web" default:"false"`
	Android     *bool `mapstructure:"android"`
	IOS         *bool `mapstructure:"ios"`
	ReactNative *bool `mapstructure:"react_native"`
	Flutter     *bool `mapstructure:"flutter"`
}

type idfaBool struct {
	IOS         *bool `mapstructure:"ios"`
	ReactNative *bool `mapstructure:"react_native"`
	Flutter     *bool `mapstructure:"flutter"`
}

type autoCaptureSetting struct {
	Web *bool `mapstructure:"web" default:"false"`
}

type autoCapture struct {
	PageViews               *autoCaptureSetting `mapstructure:"page_views"`
	PageURLEnrichment       *autoCaptureSetting `mapstructure:"page_url_enrichment"`
	WebVitals               *autoCaptureSetting `mapstructure:"web_vitals"`
	FileDownloads           *autoCaptureSetting `mapstructure:"file_downloads"`
	FrustrationInteractions *autoCaptureSetting `mapstructure:"frustration_interactions"`
	NetworkTracking         *autoCaptureSetting `mapstructure:"network_tracking"`
	ElementInteractions     *autoCaptureSetting `mapstructure:"element_interactions"`
	FormInteractions        *autoCaptureSetting `mapstructure:"form_interactions"`
}

// amplitudeConfig is the local YAML config model. Field set ports terraform's
// Amplitude mapping and adds schema/defaultConfig fields terraform omits so
// destination updates do not erase control-plane-managed configuration.
type amplitudeConfig struct {
	APIKey                           string                   `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	APISecret                        string                   `mapstructure:"api_secret" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	GroupTypeTrait                   string                   `mapstructure:"group_type_trait" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	GroupValueTrait                  string                   `mapstructure:"group_value_trait" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	TrackAllPages                    *bool                    `mapstructure:"track_all_pages" default:"false"`
	TrackCategorizedPages            *bool                    `mapstructure:"track_categorized_pages" default:"true"`
	TrackNamedPages                  *bool                    `mapstructure:"track_named_pages" default:"true"`
	UseUserDefinedPageEventName      *bool                    `mapstructure:"use_user_defined_page_event_name" default:"false"`
	UserProvidedPageEventString      string                   `mapstructure:"user_provided_page_event_string" validate:"omitempty,dynamic_or_pattern=single_line_200"`
	TraitsToIncrement                []string                 `mapstructure:"traits_to_increment" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	TraitsToSetOnce                  []string                 `mapstructure:"traits_to_set_once" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	TraitsToAppend                   []string                 `mapstructure:"traits_to_append" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	TraitsToPrepend                  []string                 `mapstructure:"traits_to_prepend" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	EnableEnhancedUserOperations     *bool                    `mapstructure:"enable_enhanced_user_operations" default:"false"`
	VersionName                      string                   `mapstructure:"version_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	MapDeviceBrand                   *bool                    `mapstructure:"map_device_brand" default:"false"`
	TrackProductsOnce                *bool                    `mapstructure:"track_products_once" default:"false"`
	TrackRevenuePerProduct           *bool                    `mapstructure:"track_revenue_per_product" default:"false"`
	UseUserDefinedScreenEventName    *bool                    `mapstructure:"use_user_defined_screen_event_name" default:"false"`
	UserProvidedScreenEventString    string                   `mapstructure:"user_provided_screen_event_string" validate:"omitempty,dynamic_or_pattern=single_line_200"`
	EventFiltering                   *eventFiltering          `mapstructure:"event_filtering"`
	SDKVersion                       *sdkVersion              `mapstructure:"sdk_version"`
	ProxyServerURL                   *proxyServerURL          `mapstructure:"proxy_server_url"`
	PreferAnonymousIDForDeviceID     *webBool                 `mapstructure:"prefer_anonymous_id_for_device_id"`
	DeviceIDFromURLParam             *webBool                 `mapstructure:"device_id_from_url_param"`
	ForceHTTPS                       *webBool                 `mapstructure:"force_https"`
	TrackGCLID                       *webBool                 `mapstructure:"track_gclid"`
	TrackReferrer                    *webBool                 `mapstructure:"track_referrer"`
	SaveParamsReferrerOncePerSession *webBool                 `mapstructure:"save_params_referrer_once_per_session"`
	TrackUTMProperties               *webBool                 `mapstructure:"track_utm_properties"`
	UnsetParamsReferrerNewSession    *webBool                 `mapstructure:"unset_params_referrer_on_new_session"`
	BatchEvents                      *webBool                 `mapstructure:"batch_events"`
	Attribution                      *webBool                 `mapstructure:"attribution"`
	TrackNewCampaigns                *webBool                 `mapstructure:"track_new_campaigns"`
	AutoCapture                      *autoCapture             `mapstructure:"auto_capture"`
	EventUploadPeriodMillis          *mobileString            `mapstructure:"event_upload_period_millis"`
	EventUploadThreshold             *mobileString            `mapstructure:"event_upload_threshold"`
	EnableLocationListening          *mobileBool              `mapstructure:"enable_location_listening"`
	TrackSessionEvents               *trackSessionEvents      `mapstructure:"track_session_events"`
	UseAdvertisingIDForDeviceID      *mobileBool              `mapstructure:"use_advertising_id_for_device_id"`
	UseIDFAAsDeviceID                *idfaBool                `mapstructure:"use_idfa_as_device_id"`
	UseNativeSDK                     *sdkBools                `mapstructure:"use_native_sdk"`
	ConnectionMode                   common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement                common.ConsentManagement `mapstructure:"consent_management"`
	ResidencyServer                  string                   `mapstructure:"residency_server" validate:"required,oneof=standard EU"`
}

// NewDefinition returns the Amplitude destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiKey", "api_key"),
		converter.Simple("apiSecret", "api_secret"),
		converter.Simple("groupTypeTrait", "group_type_trait"),
		converter.Simple("groupValueTrait", "group_value_trait"),
		converter.Simple("trackAllPages", "track_all_pages"),
		converter.Simple("trackCategorizedPages", "track_categorized_pages"),
		converter.Simple("trackNamedPages", "track_named_pages"),
		converter.Simple("useUserDefinedPageEventName", "use_user_defined_page_event_name"),
		converter.Simple("userProvidedPageEventString", "user_provided_page_event_string"),
		converter.ArrayWithStrings("traitsToIncrement", "traits", "traits_to_increment"),
		converter.ArrayWithStrings("traitsToSetOnce", "traits", "traits_to_set_once"),
		converter.ArrayWithStrings("traitsToAppend", "traits", "traits_to_append"),
		converter.ArrayWithStrings("traitsToPrepend", "traits", "traits_to_prepend"),
		converter.Simple("enableEnhancedUserOperations", "enable_enhanced_user_operations"),
		converter.Simple("trackProductsOnce", "track_products_once"),
		converter.Simple("trackRevenuePerProduct", "track_revenue_per_product"),
		converter.Simple("versionName", "version_name"),
		converter.Simple("mapDeviceBrand", "map_device_brand"),
		converter.Simple("useUserDefinedScreenEventName", "use_user_defined_screen_event_name"),
		converter.Simple("userProvidedScreenEventString", "user_provided_screen_event_string"),
		converter.Simple("residencyServer", "residency_server"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Gated(
			converter.Simple("sdkVersion.web", "sdk_version.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("proxyServerUrl.web", "proxy_server_url.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("preferAnonymousIdForDeviceId.web", "prefer_anonymous_id_for_device_id.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("deviceIdFromUrlParam.web", "device_id_from_url_param.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("forceHttps.web", "force_https.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("trackGclid.web", "track_gclid.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("trackReferrer.web", "track_referrer.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("saveParamsReferrerOncePerSession.web", "save_params_referrer_once_per_session.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("trackUtmProperties.web", "track_utm_properties.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("unsetParamsReferrerOnNewSession.web", "unset_params_referrer_on_new_session.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("batchEvents.web", "batch_events.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("attribution.web", "attribution.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("trackNewCampaigns.web", "track_new_campaigns.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enablePageViewsAutoCapture.web", "auto_capture.page_views.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enablePageUrlEnrichmentAutoCapture.web", "auto_capture.page_url_enrichment.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableWebVitalsAutoCapture.web", "auto_capture.web_vitals.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableFileDownloadsAutoCapture.web", "auto_capture.file_downloads.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableFrustrationInteractionsAutoCapture.web", "auto_capture.frustration_interactions.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableNetworkTrackingAutoCapture.web", "auto_capture.network_tracking.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableElementInteractionsAutoCapture.web", "auto_capture.element_interactions.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableFormInteractionsAutoCapture.web", "auto_capture.form_interactions.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("eventUploadPeriodMillis.web", "event_upload_period_millis.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("eventUploadPeriodMillis.android", "event_upload_period_millis.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("eventUploadPeriodMillis.ios", "event_upload_period_millis.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("eventUploadPeriodMillis.reactnative", "event_upload_period_millis.react_native"),
			common.SourceTypeReactNative,
		),
		converter.Gated(
			converter.Simple("eventUploadPeriodMillis.flutter", "event_upload_period_millis.flutter"),
			common.SourceTypeFlutter,
		),
		converter.Gated(
			converter.Simple("eventUploadThreshold.web", "event_upload_threshold.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("eventUploadThreshold.android", "event_upload_threshold.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("eventUploadThreshold.ios", "event_upload_threshold.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("eventUploadThreshold.reactnative", "event_upload_threshold.react_native"),
			common.SourceTypeReactNative,
		),
		converter.Gated(
			converter.Simple("eventUploadThreshold.flutter", "event_upload_threshold.flutter"),
			common.SourceTypeFlutter,
		),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.reactnative", "use_native_sdk.react_native"),
		converter.Simple("useNativeSDK.flutter", "use_native_sdk.flutter"),
		converter.Gated(
			converter.Simple("enableLocationListening.android", "enable_location_listening.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("enableLocationListening.reactnative", "enable_location_listening.react_native"),
			common.SourceTypeReactNative,
		),
		converter.Gated(
			converter.Simple("enableLocationListening.flutter", "enable_location_listening.flutter"),
			common.SourceTypeFlutter,
		),
		converter.Gated(
			converter.Simple("trackSessionEvents.web", "track_session_events.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("trackSessionEvents.android", "track_session_events.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("trackSessionEvents.ios", "track_session_events.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("trackSessionEvents.reactnative", "track_session_events.react_native"),
			common.SourceTypeReactNative,
		),
		converter.Gated(
			converter.Simple("trackSessionEvents.flutter", "track_session_events.flutter"),
			common.SourceTypeFlutter,
		),
		converter.Gated(
			converter.Simple("useAdvertisingIdForDeviceId.android", "use_advertising_id_for_device_id.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("useAdvertisingIdForDeviceId.reactnative", "use_advertising_id_for_device_id.react_native"),
			common.SourceTypeReactNative,
		),
		converter.Gated(
			converter.Simple("useAdvertisingIdForDeviceId.flutter", "use_advertising_id_for_device_id.flutter"),
			common.SourceTypeFlutter,
		),
		converter.Gated(
			converter.Simple("useIdfaAsDeviceId.ios", "use_idfa_as_device_id.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("useIdfaAsDeviceId.reactnative", "use_idfa_as_device_id.react_native"),
			common.SourceTypeReactNative,
		),
		converter.Gated(
			converter.Simple("useIdfaAsDeviceId.flutter", "use_idfa_as_device_id.flutter"),
			common.SourceTypeFlutter,
		),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "am",
		APIType:    "AM",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_secret"},
		NewConfig: func() any {
			return &amplitudeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
