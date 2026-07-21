package tiktokads

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/tiktok_ads/db-config.json
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
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

type eventToStandard struct {
	From string `mapstructure:"from" validate:"omitempty,max=100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_oneof=AddPaymentInfo AddToCart AddToWishlist ClickButton CompletePayment CompleteRegistration Contact Download InitiateCheckout PlaceAnOrder Search SubmitForm Subscribe ViewContent CustomizeProduct FindLocation Schedule Purchase Lead ApplicationApproval SubmitApplication StartTrial"`
}

type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

type connectionMode struct {
	Web           *string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=cloud device"`
	Cloud         *string `mapstructure:"cloud" validate:"omitempty,dynamic_or_oneof=cloud"`
	IOS           *string `mapstructure:"ios" validate:"omitempty,dynamic_or_oneof=cloud"`
	IOSSwift      *string `mapstructure:"ios_swift" validate:"omitempty,dynamic_or_oneof=cloud"`
	Android       *string `mapstructure:"android" validate:"omitempty,dynamic_or_oneof=cloud"`
	AndroidKotlin *string `mapstructure:"android_kotlin" validate:"omitempty,dynamic_or_oneof=cloud"`
	Unity         *string `mapstructure:"unity" validate:"omitempty,dynamic_or_oneof=cloud"`
	ReactNative   *string `mapstructure:"react_native" validate:"omitempty,dynamic_or_oneof=cloud"`
	Flutter       *string `mapstructure:"flutter" validate:"omitempty,dynamic_or_oneof=cloud"`
	Cordova       *string `mapstructure:"cordova" validate:"omitempty,dynamic_or_oneof=cloud"`
}

// tiktokAdsConfig is the local YAML config model. Field set mirrors terraform
// destination_tiktok_ads mappings; validation constraints mirror schema.json.
type tiktokAdsConfig struct {
	PixelCode               string                   `mapstructure:"pixel_code" validate:"required,min=1,max=100"`
	AccessToken             string                   `mapstructure:"access_token"`
	Version                 string                   `mapstructure:"version" validate:"omitempty,dynamic_or_oneof=v2 v1"`
	HashUserProperties      *bool                    `mapstructure:"hash_user_properties"`
	SendCustomEvents        *bool                    `mapstructure:"send_custom_events"`
	EventsToStandard        []eventToStandard        `mapstructure:"events_to_standard" validate:"omitempty,dive"`
	EventFilteringWhitelist []string                 `mapstructure:"event_filtering_whitelist"`
	EventFilteringBlacklist []string                 `mapstructure:"event_filtering_blacklist"`
	UseNativeSDK            useNativeSDK             `mapstructure:"use_native_sdk"`
	ConnectionMode          connectionMode           `mapstructure:"connection_mode"`
	ConsentManagement       common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the TikTok Ads destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("pixelCode", "pixel_code"),
		converter.Simple("accessToken", "access_token", converter.SkipZeroValue),
		converter.Simple("version", "version"),
		converter.Simple("hashUserProperties", "hash_user_properties"),
		converter.Simple("sendCustomEvents", "send_custom_events", converter.SkipZeroValue),
		converter.ArrayWithObjects("eventsToStandard", "events_to_standard", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering_whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering_blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering_whitelist": "whitelistedEvents",
			"event_filtering_blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("connectionMode.web", "connection_mode.web", converter.SkipZeroValue),
		converter.Simple("connectionMode.cloud", "connection_mode.cloud", converter.SkipZeroValue),
		converter.Simple("connectionMode.ios", "connection_mode.ios", converter.SkipZeroValue),
		converter.Simple("connectionMode.iosSwift", "connection_mode.ios_swift", converter.SkipZeroValue),
		converter.Simple("connectionMode.android", "connection_mode.android", converter.SkipZeroValue),
		converter.Simple("connectionMode.androidKotlin", "connection_mode.android_kotlin", converter.SkipZeroValue),
		converter.Simple("connectionMode.unity", "connection_mode.unity", converter.SkipZeroValue),
		converter.Simple("connectionMode.reactnative", "connection_mode.react_native", converter.SkipZeroValue),
		converter.Simple("connectionMode.flutter", "connection_mode.flutter", converter.SkipZeroValue),
		converter.Simple("connectionMode.cordova", "connection_mode.cordova", converter.SkipZeroValue),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "tiktok_ads",
		APIType:    "TIKTOK_ADS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"access_token"},
		NewConfig: func() any {
			return &tiktokAdsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
