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
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

type customerioConfig struct {
	SiteID               string `mapstructure:"site_id" validate:"required,dynamic_or_pattern=single_line_100"`
	APIKey               string `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	DeviceTokenEventName string `mapstructure:"device_token_event_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Datacenter           string `mapstructure:"datacenter" validate:"required,oneof=US EU"`
	// The v2 API path: both keys are declared by schema.json and db-config, and
	// were unmodelled, so update erased whatever the UI had set.
	// Since integrations-config #2661 the backend persists api_version, applying
	// the schema default "v1" when the key is absent (the destinations e2e caught
	// the extra key upstream). Declare the same default so a spec omitting it
	// matches what the backend stores instead of diffing on every apply.
	APIVersion                  string                   `mapstructure:"api_version" validate:"omitempty,oneof=v1 v2" default:"v1"`
	UserIDIdentifierType        string                   `mapstructure:"user_id_identifier_type" validate:"required_if=APIVersion v2,omitempty,oneof=id email phone cio_id"`
	UseNativeSDK                *sdkSourceBools          `mapstructure:"use_native_sdk"`
	SendPageNameInSDK           *webBool                 `mapstructure:"send_page_name_in_sdk"`
	DataUseInApp                *webBool                 `mapstructure:"data_use_in_app"`
	AutoTrackDeviceAttributes   *mobileSourceBools       `mapstructure:"auto_track_device_attributes"`
	BackgroundQueueMinTasks     *androidString           `mapstructure:"background_queue_min_number_of_tasks"`
	BackgroundQueueSecondsDelay *androidString           `mapstructure:"background_queue_seconds_delay"`
	EventFiltering              *eventFiltering          `mapstructure:"event_filtering"`
	ConnectionMode              common.ConnectionMode    `mapstructure:"connection_mode"`
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
		converter.Simple("apiVersion", "api_version"),
		converter.Simple("userIdIdentifierType", "user_id_identifier_type"),
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
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "customerio",
		APIType:    "CUSTOMERIO",
		Version:    1,
		Properties: properties,
		// db-config lists no secretKeys for customerio, but terraform marks api_key
		// Sensitive and it is a real credential, so it is wrapped write-only here.
		// Note the API does still return apiKey, so the value is never absent from
		// remote state — see the churn note in the PR.
		SecretKeys: []string{"api_key"},
		NewConfig: func() any {
			return &customerioConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
