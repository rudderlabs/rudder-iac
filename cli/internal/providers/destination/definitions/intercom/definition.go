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

type intercomConfig struct {
	AppID               *string                  `mapstructure:"app_id" validate:"omitempty,dynamic_or_pattern=intercom_single_line_1_100"`
	APIKey              *string                  `mapstructure:"api_key" validate:"omitempty,dynamic_or_pattern=intercom_single_line_1_100"`
	APIServer           string                   `mapstructure:"api_server" validate:"omitempty,dynamic_or_oneof=standard eu au"`
	APIVersion          string                   `mapstructure:"api_version" validate:"omitempty,dynamic_or_oneof=v1 v2"`
	SendAnonymousID     *bool                    `mapstructure:"send_anonymous_id"`
	UpdateLastRequestAt *bool                    `mapstructure:"update_last_request_at"`
	MobileAPIKeyAndroid string                   `mapstructure:"mobile_api_key_android" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	MobileAPIKeyIOS     string                   `mapstructure:"mobile_api_key_ios" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	EventFiltering      *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK        *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConnectionMode      *connectionMode          `mapstructure:"connection_mode"`
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

type connectionMode struct {
	Android       string `mapstructure:"android" validate:"omitempty,dynamic_or_oneof=cloud device"`
	AndroidKotlin string `mapstructure:"android_kotlin" validate:"omitempty,dynamic_or_oneof=cloud"`
	IOS           string `mapstructure:"ios" validate:"omitempty,dynamic_or_oneof=cloud device"`
	IOSSwift      string `mapstructure:"ios_swift" validate:"omitempty,dynamic_or_oneof=cloud"`
	Web           string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=cloud device"`
	Unity         string `mapstructure:"unity" validate:"omitempty,dynamic_or_oneof=cloud"`
	AMP           string `mapstructure:"amp" validate:"omitempty,dynamic_or_oneof=cloud"`
	Cloud         string `mapstructure:"cloud" validate:"omitempty,dynamic_or_oneof=cloud"`
	Warehouse     string `mapstructure:"warehouse" validate:"omitempty,dynamic_or_oneof=cloud"`
	ReactNative   string `mapstructure:"react_native" validate:"omitempty,dynamic_or_oneof=cloud"`
	Flutter       string `mapstructure:"flutter" validate:"omitempty,dynamic_or_oneof=cloud"`
	Cordova       string `mapstructure:"cordova" validate:"omitempty,dynamic_or_oneof=cloud"`
	Shopify       string `mapstructure:"shopify" validate:"omitempty,dynamic_or_oneof=cloud"`
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
		converter.Simple("connectionMode.web", "connection_mode.web"),
		converter.Simple("connectionMode.android", "connection_mode.android"),
		converter.Simple("connectionMode.androidKotlin", "connection_mode.android_kotlin"),
		converter.Simple("connectionMode.ios", "connection_mode.ios"),
		converter.Simple("connectionMode.iosSwift", "connection_mode.ios_swift"),
		converter.Simple("connectionMode.unity", "connection_mode.unity"),
		converter.Simple("connectionMode.amp", "connection_mode.amp"),
		converter.Simple("connectionMode.cloud", "connection_mode.cloud"),
		converter.Simple("connectionMode.warehouse", "connection_mode.warehouse"),
		converter.Simple("connectionMode.reactnative", "connection_mode.react_native"),
		converter.Simple("connectionMode.flutter", "connection_mode.flutter"),
		converter.Simple("connectionMode.cordova", "connection_mode.cordova"),
		converter.Simple("connectionMode.shopify", "connection_mode.shopify"),
	}
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
