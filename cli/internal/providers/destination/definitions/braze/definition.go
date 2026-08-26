package braze

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/braze/db-config.json.
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
	common.SourceTypeAndroid:       {"cloud", "device", "hybrid"},
	common.SourceTypeAndroidKotlin: {"cloud", "device", "hybrid"},
	common.SourceTypeIOS:           {"cloud", "device", "hybrid"},
	common.SourceTypeIOSSwift:      {"cloud", "device", "hybrid"},
	common.SourceTypeWeb:           {"cloud", "device", "hybrid"},
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

type useNativeSDK struct {
	Android       *bool `mapstructure:"android"`
	AndroidKotlin *bool `mapstructure:"android_kotlin"`
	IOS           *bool `mapstructure:"ios"`
	IOSSwift      *bool `mapstructure:"ios_swift"`
	Web           *bool `mapstructure:"web"`
	ReactNative   *bool `mapstructure:"react_native"`
	Flutter       *bool `mapstructure:"flutter"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

// brazeConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/braze schema/defaultConfig; validation constraints mirror schema.json
// where they can be expressed without making mode-dependent import paths stricter
// than the backend.
type brazeConfig struct {
	RestAPIKey                           string                   `mapstructure:"rest_api_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	AppKey                               string                   `mapstructure:"app_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	DataCenter                           string                   `mapstructure:"data_center" validate:"required,dynamic_or_oneof=US-01 US-02 US-03 US-04 US-05 US-06 US-07 US-08 EU-01 EU-02 EU-03 AU-01"`
	EnableSubscriptionGroupInGroupCall   *bool                    `mapstructure:"enable_subscription_group_in_group_call"`
	EnableNestedArrayOperations          *bool                    `mapstructure:"enable_nested_array_operations"`
	SendPurchaseEventWithExtraProperties *bool                    `mapstructure:"send_purchase_event_with_extra_properties"`
	TrackAnonymousUser                   *webBool                 `mapstructure:"track_anonymous_user"`
	SupportDedup                         *bool                    `mapstructure:"support_dedup"`
	EnableBrazeLogging                   *webBool                 `mapstructure:"enable_braze_logging"`
	EnablePushNotification               *webBool                 `mapstructure:"enable_push_notification"`
	AllowUserSuppliedJavascript          *webBool                 `mapstructure:"allow_user_supplied_javascript"`
	EventFiltering                       *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK                         *useNativeSDK            `mapstructure:"use_native_sdk"`
	UseEcommerceRecommendedEvents        *bool                    `mapstructure:"use_ecommerce_recommended_events"`
	UsePlatformSpecificAPIKeys           *bool                    `mapstructure:"use_platform_specific_api_keys"`
	AndroidAPIKey                        string                   `mapstructure:"android_api_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	IOSAPIKey                            string                   `mapstructure:"ios_api_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	WebAPIKey                            string                   `mapstructure:"web_api_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	ConnectionMode                       common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement                    common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Braze destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("restApiKey", "rest_api_key"),
		converter.Simple("appKey", "app_key"),
		converter.Simple("dataCenter", "data_center"),
		converter.Simple("enableSubscriptionGroupInGroupCall", "enable_subscription_group_in_group_call"),
		converter.Simple("enableNestedArrayOperations", "enable_nested_array_operations"),
		converter.Simple("sendPurchaseEventWithExtraProperties", "send_purchase_event_with_extra_properties"),
		converter.Gated(
			converter.Simple("trackAnonymousUser.web", "track_anonymous_user.web"),
			common.SourceTypeWeb,
		),
		converter.Simple("supportDedup", "support_dedup"),
		converter.Gated(
			converter.Simple("enableBrazeLogging.web", "enable_braze_logging.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enablePushNotification.web", "enable_push_notification.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("allowUserSuppliedJavascript.web", "allow_user_supplied_javascript.web"),
			common.SourceTypeWeb,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.androidKotlin", "use_native_sdk.android_kotlin"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.iosSwift", "use_native_sdk.ios_swift"),
		converter.Simple("useNativeSDK.reactnative", "use_native_sdk.react_native"),
		converter.Simple("useNativeSDK.flutter", "use_native_sdk.flutter"),
		converter.Simple("useEcommerceRecommendedEvents", "use_ecommerce_recommended_events"),
		converter.Simple("usePlatformSpecificApiKeys", "use_platform_specific_api_keys"),
		converter.Simple("androidApiKey", "android_api_key"),
		converter.Simple("iOSApiKey", "ios_api_key"),
		converter.Simple("webApiKey", "web_api_key"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "braze",
		APIType:    "BRAZE",
		Version:    1,
		Properties: properties,
		// db-config declares only restApiKey as a secret. Terraform also marks appKey
		// sensitive, but db-config is authoritative for CLI write-only wrapping.
		SecretKeys: []string{"rest_api_key"},
		NewConfig: func() any {
			return &brazeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
