package ga4

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/ga4/db-config.json
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
	common.SourceTypeWeb:           {"cloud", "device", "hybrid"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

type piiProperty struct {
	PIIProperty string `mapstructure:"pii_property" validate:"omitempty,max=100"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist"`
	Blacklist []string `mapstructure:"blacklist"`
}

type useNativeSDK struct {
	Web     *bool `mapstructure:"web"`
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type webCapturePageView struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=rs gtag"`
}

// ga4Config is the local YAML config model. Field set mirrors terraform-provider
// destination_google_analytics4 mappings; validation constraints mirror
// overlapping schema.json rules (required, enums, client-type conditionals).
type ga4Config struct {
	APISecret             string                   `mapstructure:"api_secret" validate:"required,min=1,max=100"`
	ClientType            string                   `mapstructure:"client_type" validate:"required,dynamic_or_oneof=gtag firebase"`
	MeasurementID         string                   `mapstructure:"measurement_id" validate:"required_if=ClientType gtag,omitempty,max=100"`
	FirebaseAppID         string                   `mapstructure:"firebase_app_id" validate:"required_if=ClientType firebase,omitempty,max=100"`
	DebugMode             *bool                    `mapstructure:"debug_mode"`
	BlockPageViewEvent    *bool                    `mapstructure:"block_page_view_event"`
	ExtendPageViewParams  *bool                    `mapstructure:"extend_page_view_params"`
	SendUserID            *bool                    `mapstructure:"send_user_id"`
	SDKBaseURL            string                   `mapstructure:"sdk_base_url"`
	ServerContainerURL    string                   `mapstructure:"server_container_url"`
	PIIPropertiesToIgnore []piiProperty            `mapstructure:"pii_properties_to_ignore" validate:"omitempty,dive"`
	EventFiltering        *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK          *useNativeSDK            `mapstructure:"use_native_sdk"`
	CapturePageView       *webCapturePageView      `mapstructure:"capture_page_view"`
	DebugView             *webBool                 `mapstructure:"debug_view"`
	OverrideClientSession *webBool                 `mapstructure:"override_client_and_session_ids"`
	ConsentManagement     common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Analytics 4 (GA4) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiSecret", "api_secret"),
		converter.Simple("typesOfClient", "client_type"),
		converter.Simple("measurementId", "measurement_id", converter.SkipZeroValue),
		converter.Simple("firebaseAppId", "firebase_app_id", converter.SkipZeroValue),
		converter.Simple("debugMode", "debug_mode"),
		converter.Simple("blockPageViewEvent", "block_page_view_event", converter.SkipZeroValue),
		converter.Gated(
			converter.Simple("extendPageViewParams", "extend_page_view_params", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Simple("sendUserId", "send_user_id", converter.SkipZeroValue),
		converter.Simple("sdkBaseUrl", "sdk_base_url", converter.SkipZeroValue),
		converter.Simple("serverContainerUrl", "server_container_url", converter.SkipZeroValue),
		converter.ArrayWithObjects("piiPropertiesToIgnore", "pii_properties_to_ignore", map[string]any{
			"piiProperty": "pii_property",
		}),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Gated(
			converter.Simple("capturePageView.web", "capture_page_view.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("debugView.web", "debug_view.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("overrideClientAndSessionId.web", "override_client_and_session_ids.web"),
			common.SourceTypeWeb,
		),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "ga4",
		APIType:    "GA4",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_secret"},
		NewConfig: func() any {
			return &ga4Config{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
