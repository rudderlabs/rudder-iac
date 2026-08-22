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
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_oneof=AddPaymentInfo AddToCart AddToWishlist ClickButton CompletePayment CompleteRegistration Contact Download InitiateCheckout PlaceAnOrder Search SubmitForm Subscribe ViewContent CustomizeProduct FindLocation Schedule Purchase Lead ApplicationApproval SubmitApplication StartTrial"`
}

// useNativeSDK is web-only upstream: db-config lists the key under `web` alone,
// and schema.json declares no other source type for it.
type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

// tiktokAdsConfig is the local YAML config model. Field set mirrors terraform
// destination_tiktok_ads mappings; validation constraints mirror schema.json.
type tiktokAdsConfig struct {
	PixelCode               string                   `mapstructure:"pixel_code" validate:"required,dynamic_or_pattern=single_line_100"`
	AccessToken             string                   `mapstructure:"access_token"`
	Version                 string                   `mapstructure:"version" validate:"omitempty,dynamic_or_oneof=v2 v1"`
	HashUserProperties      *bool                    `mapstructure:"hash_user_properties"`
	SendCustomEvents        *bool                    `mapstructure:"send_custom_events"`
	EventsToStandard        []eventToStandard        `mapstructure:"events_to_standard" validate:"omitempty,dive"`
	EventFilteringWhitelist []string                 `mapstructure:"event_filtering_whitelist" validate:"omitempty,excluded_with=EventFilteringBlacklist,dive,dynamic_or_pattern=single_line_100"`
	EventFilteringBlacklist []string                 `mapstructure:"event_filtering_blacklist" validate:"omitempty,excluded_with=EventFilteringWhitelist,dive,dynamic_or_pattern=single_line_100"`
	UseNativeSDK            useNativeSDK             `mapstructure:"use_native_sdk"`
	ConsentManagement       common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the TikTok Ads destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("pixelCode", "pixel_code"),
		converter.Simple("accessToken", "access_token"),
		converter.Simple("version", "version"),
		converter.Simple("hashUserProperties", "hash_user_properties"),
		converter.Simple("sendCustomEvents", "send_custom_events"),
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
