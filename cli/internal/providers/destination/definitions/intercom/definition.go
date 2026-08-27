package intercom

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	funcs.NewPattern(
		"intercom_single_line_1_100",
		`^(.{1,100})$`,
		"must be 1-100 characters and must not contain line breaks",
	)
}

// Source types from integrations-config destinations/intercom/db-config.json.
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

type intercomConfig struct {
	AppID               *string                  `mapstructure:"app_id" validate:"omitempty,dynamic_or_pattern=intercom_single_line_1_100"`
	APIKey              *string                  `mapstructure:"api_key" validate:"omitempty,dynamic_or_pattern=intercom_single_line_1_100"`
	APIServer           string                   `mapstructure:"api_server" validate:"omitempty,dynamic_or_oneof=standard eu au" default:"standard"`
	APIVersion          string                   `mapstructure:"api_version" validate:"omitempty,dynamic_or_oneof=v1 v2" default:"v2"`
	SendAnonymousID     *bool                    `mapstructure:"send_anonymous_id" default:"false"`
	UpdateLastRequestAt *bool                    `mapstructure:"update_last_request_at" default:"true"`
	MobileAPIKeyAndroid string                   `mapstructure:"mobile_api_key_android" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	MobileAPIKeyIOS     string                   `mapstructure:"mobile_api_key_ios" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	EventFiltering      *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK        *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConnectionMode      common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement   common.ConsentManagement `mapstructure:"consent_management"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDK struct {
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
	Web     *bool `mapstructure:"web"`
}

// NewDefinition returns the Intercom destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("appId", "app_id"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("apiServer", "api_server"),
		converter.Simple("apiVersion", "api_version"),
		converter.Simple("sendAnonymousId", "send_anonymous_id"),
		converter.Simple("updateLastRequestAt", "update_last_request_at"),
		converter.Gated(
			converter.Simple("mobileApiKeyAndroid.android", "mobile_api_key_android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("mobileApiKeyIOS.ios", "mobile_api_key_ios"),
			common.SourceTypeIOS,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "intercom",
		APIType:    "INTERCOM",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_key"},
		NewConfig: func() any {
			return &intercomConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
