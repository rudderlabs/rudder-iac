package customerio

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/customerio/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

type customerioConfig struct {
	SiteID                      string                   `mapstructure:"site_id" validate:"required,dynamic_or_pattern=single_line_100"`
	APIKey                      string                   `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	DeviceTokenEventName        string                   `mapstructure:"device_token_event_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Datacenter                  string                   `mapstructure:"datacenter" validate:"required,dynamic_or_oneof=US EU"`
	UseNativeSDK                *sdkSourceBools          `mapstructure:"use_native_sdk"`
	SendPageNameInSDK           *webBool                 `mapstructure:"send_page_name_in_sdk"`
	DataUseInApp                *webBool                 `mapstructure:"data_use_in_app"`
	AutoTrackDeviceAttributes   *mobileSourceBools       `mapstructure:"auto_track_device_attributes"`
	BackgroundQueueMinTasks     *androidString           `mapstructure:"background_queue_min_number_of_tasks"`
	BackgroundQueueSecondsDelay *androidString           `mapstructure:"background_queue_seconds_delay"`
	EventFiltering              *eventFiltering          `mapstructure:"event_filtering"`
	ConsentManagement           common.ConsentManagement `mapstructure:"consent_management"`
}

type sdkSourceBools struct {
	Web     *bool `mapstructure:"web"`
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type mobileSourceBools struct {
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

type androidString struct {
	Android string `mapstructure:"android" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

// NewDefinition returns the Customer.io destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("siteID", "site_id"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("deviceTokenEventName", "device_token_event_name"),
		converter.Simple("datacenter", "datacenter"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Gated(
			converter.Simple("sendPageNameInSDK.web", "send_page_name_in_sdk.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("dataUseInApp.web", "data_use_in_app.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("autoTrackDeviceAttributes.android", "auto_track_device_attributes.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("autoTrackDeviceAttributes.ios", "auto_track_device_attributes.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("backgroundQueueMinNumberOfTasks.android", "background_queue_min_number_of_tasks.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("backgroundQueueSecondsDelay.android", "background_queue_seconds_delay.android"),
			common.SourceTypeAndroid,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "customerio",
		APIType:    "CUSTOMERIO",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_key"},
		NewConfig: func() any {
			return &customerioConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
