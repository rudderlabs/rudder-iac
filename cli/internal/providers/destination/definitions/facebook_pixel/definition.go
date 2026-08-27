package facebookpixel

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/facebook_pixel/db-config.json.
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
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

type eventMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_oneof=ViewContent Search AddToCart AddToWishlist InitiateCheckout AddPaymentInfo Purchase PageView Lead CompleteRegistration Contact CustomizeProduct Donate FindLocation Schedule StartTrial SubmitApplication Subscribe"`
}

type piiDenylistEntry struct {
	Property string `mapstructure:"property" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Hash     *bool  `mapstructure:"hash"`
}

type piiAllowlistEntry struct {
	Property string `mapstructure:"property" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type legacyConversionPixelID struct {
	Web []legacyConversionPixelMapping `mapstructure:"web" validate:"omitempty,dive"`
}

type legacyConversionPixelMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

// facebookPixelConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/facebook_pixel schema/defaultConfig;
// validation constraints mirror schema.json where present.
type facebookPixelConfig struct {
	PixelID                 string                   `mapstructure:"pixel_id" validate:"required,dynamic_or_pattern=single_line_100"`
	AccessToken             string                   `mapstructure:"access_token" validate:"omitempty,dynamic_or_pattern=single_line_300"`
	StandardPageCall        *bool                    `mapstructure:"standard_page_call"`
	ValueFieldIdentifier    string                   `mapstructure:"value_field_identifier" validate:"omitempty,dynamic_or_oneof=properties.value properties.price"`
	AdvancedMapping         *bool                    `mapstructure:"advanced_mapping"`
	LimitedDataUsage        *bool                    `mapstructure:"limited_data_usage"`
	TestDestination         *bool                    `mapstructure:"test_destination"`
	TestEventCode           string                   `mapstructure:"test_event_code" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	RemoveExternalID        *bool                    `mapstructure:"remove_external_id"`
	UseUpdatedMapping       *bool                    `mapstructure:"use_updated_mapping"`
	EventsToEvents          []eventMapping           `mapstructure:"events_to_events" validate:"omitempty,dive"`
	BlacklistPIIProperties  []piiDenylistEntry       `mapstructure:"blacklist_pii_properties" validate:"omitempty,dive"`
	WhitelistPIIProperties  []piiAllowlistEntry      `mapstructure:"whitelist_pii_properties" validate:"omitempty,dive"`
	EventFiltering          *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK            webBool                  `mapstructure:"use_native_sdk"`
	AutoConfig              webBool                  `mapstructure:"auto_config"`
	LegacyConversionPixelID legacyConversionPixelID  `mapstructure:"legacy_conversion_pixel_id"`
	ConnectionMode          common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement       common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Facebook Pixel destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("pixelId", "pixel_id"),
		converter.Simple("accessToken", "access_token"),
		converter.Simple("standardPageCall", "standard_page_call"),
		converter.Simple("valueFieldIdentifier", "value_field_identifier"),
		converter.Simple("advancedMapping", "advanced_mapping"),
		converter.Simple("limitedDataUSage", "limited_data_usage"),
		converter.Simple("testDestination", "test_destination"),
		converter.Simple("testEventCode", "test_event_code"),
		converter.Simple("removeExternalId", "remove_external_id"),
		converter.Simple("useUpdatedMapping", "use_updated_mapping"),
		converter.ArrayWithObjects("eventsToEvents", "events_to_events", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("blacklistPiiProperties", "blacklist_pii_properties", map[string]any{
			"blacklistPiiProperties": "property",
			"blacklistPiiHash":       "hash",
		}),
		converter.ArrayWithObjects("whitelistPiiProperties", "whitelist_pii_properties", map[string]any{
			"whitelistPiiProperties": "property",
		}),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Gated(
			converter.Simple("autoConfig.web", "auto_config.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithObjects("legacyConversionPixelId.web", "legacy_conversion_pixel_id.web", map[string]any{
				"from": "from",
				"to":   "to",
			}),
			common.SourceTypeWeb,
		),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "facebook_pixel",
		APIType:    "FACEBOOK_PIXEL",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"access_token"},
		NewConfig: func() any {
			return &facebookPixelConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
