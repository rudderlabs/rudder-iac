package googleads

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/googleads/db-config.json
// supportedSourceTypes (web only).
var sourceTypes = []string{
	common.SourceTypeWeb,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb: {"device"},
}

type conversionEntry struct {
	Label string `mapstructure:"label" validate:"required,max=100"`
	Name  string `mapstructure:"name" validate:"required,max=100"`
}

type eventFilteringConfig struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,dive,max=100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,dive,max=100"`
}

type webBoolConfig struct {
	Web *bool `mapstructure:"web"`
}

// googleAdsConfig is the local YAML config model. Field set mirrors
// terraform-provider destination_google_ads.go; validation constraints mirror
// overlapping schema.json rules for those mapped fields.
type googleAdsConfig struct {
	ConversionID             string                   `mapstructure:"conversion_id" validate:"required,min=1,max=103"`
	PageLoadConversions      []conversionEntry        `mapstructure:"page_load_conversions" validate:"omitempty,dive"`
	ClickEventConversions    []conversionEntry        `mapstructure:"click_event_conversions" validate:"omitempty,dive"`
	DefaultPageConversion    string                   `mapstructure:"default_page_conversion" validate:"omitempty,max=100"`
	DynamicRemarketing       *webBoolConfig           `mapstructure:"dynamic_remarketing"`
	ConversionLinker         *bool                    `mapstructure:"conversion_linker"`
	SendPageView             *bool                    `mapstructure:"send_page_view"`
	DisableAdPersonalization *bool                    `mapstructure:"disable_ad_personalization"`
	EventFiltering           *eventFilteringConfig    `mapstructure:"event_filtering"`
	UseNativeSDK             *webBoolConfig           `mapstructure:"use_native_sdk"`
	ConsentManagement        common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Ads (API type GOOGLEADS) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("conversionID", "conversion_id"),
		converter.ArrayWithObjects("pageLoadConversions", "page_load_conversions", map[string]any{
			"conversionLabel": "label",
			"name":            "name",
		}),
		converter.ArrayWithObjects("clickEventConversions", "click_event_conversions", map[string]any{
			"conversionLabel": "label",
			"name":            "name",
		}),
		converter.Simple("defaultPageConversion", "default_page_conversion", converter.SkipZeroValue),
		converter.Simple("dynamicRemarketing.web", "dynamic_remarketing.web"),
		converter.Simple("conversionLinker", "conversion_linker"),
		converter.Simple("sendPageView", "send_page_view"),
		converter.Simple("disableAdPersonalization", "disable_ad_personalization", converter.SkipZeroValue),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "googleads",
		APIType:    "GOOGLEADS",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &googleAdsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
