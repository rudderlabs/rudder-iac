package googleads

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	funcs.NewPattern(
		"googleads_conversion_id",
		`^AW-(.{0,100})$`,
		"must be a Google Ads conversion ID starting with AW-",
	)

	// sdkBaseUrl bounds its length with a lookahead RE2 cannot express; the
	// 0-500 bound is carried by a max tag instead. Empty is allowed upstream.
	funcs.NewPattern(
		"googleads_sdk_base_url",
		`^(?:https?://)?[\w.-]+(?:\.[\w.-]+)+[\w\-._~:/?#[\]@!$&'()*+,;=.]*$|^$`,
		"must be a domain URL",
	)
}

// Source types from integrations-config destinations/googleads/db-config.json
// supportedSourceTypes (web only).
var sourceTypes = []string{
	common.SourceTypeWeb,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb: {"device"},
}

// conversionEntry mirrors the pageLoadConversions / clickEventConversions item
// shape. schema.json marks neither field required, so neither is here.
type conversionEntry struct {
	Label string `mapstructure:"label" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Name  string `mapstructure:"name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type eventMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,oneof=Lead PageVisit ViewCategory Signup WatchVideo Checkout Search AddToCart purchase"`
}

type eventFilteringConfig struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type webBoolConfig struct {
	Web *bool `mapstructure:"web"`
}

// googleAdsConfig is the local YAML config model. Field set mirrors the keys
// upstream declares in db-config.json destConfig; validation constraints mirror
// schema.json.
type googleAdsConfig struct {
	ConversionID             string            `mapstructure:"conversion_id" validate:"required,dynamic_or_pattern=googleads_conversion_id"`
	V2                       *bool             `mapstructure:"v2" default:"true"`
	AllowIdentify            *bool             `mapstructure:"allow_identify" default:"false"`
	SDKBaseURL               string            `mapstructure:"sdk_base_url" validate:"omitempty,max=500,dynamic_or_pattern=googleads_sdk_base_url"`
	EventMappingFromConfig   []eventMapping    `mapstructure:"event_mapping_from_config" validate:"omitempty,dive"`
	PageLoadConversions      []conversionEntry `mapstructure:"page_load_conversions" validate:"omitempty,dive"`
	ClickEventConversions    []conversionEntry `mapstructure:"click_event_conversions" validate:"omitempty,dive"`
	DefaultPageConversion    string            `mapstructure:"default_page_conversion" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	DynamicRemarketing       *webBoolConfig    `mapstructure:"dynamic_remarketing"`
	ConversionLinker         *bool             `mapstructure:"conversion_linker" default:"true"`
	SendPageView             *bool             `mapstructure:"send_page_view" default:"true"`
	DisableAdPersonalization *bool             `mapstructure:"disable_ad_personalization" default:"false"`
	EnableConversionLabel    *bool             `mapstructure:"enable_conversion_label" default:"false"`
	AllowEnhancedConversions *bool             `mapstructure:"allow_enhanced_conversions" default:"false"`

	// Conversion and dynamic-remarketing tracking each gate a filtering toggle,
	// which in turn gates an event list. schema.json expresses that as nested
	// allOf branches; the keys are modelled unconditionally because every one of
	// them lives in destConfig.defaultConfig and would otherwise be erased.
	TrackConversions                        *bool    `mapstructure:"track_conversions" default:"true"`
	EnableConversionEventsFiltering         *bool    `mapstructure:"enable_conversion_events_filtering" default:"false"`
	EventsToTrackConversions                []string `mapstructure:"events_to_track_conversions" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	TrackDynamicRemarketing                 *bool    `mapstructure:"track_dynamic_remarketing" default:"false"`
	EnableDynamicRemarketingEventsFiltering *bool    `mapstructure:"enable_dynamic_remarketing_events_filtering" default:"false"`
	EventsToTrackDynamicRemarketing         []string `mapstructure:"events_to_track_dynamic_remarketing" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`

	EventFiltering    *eventFilteringConfig    `mapstructure:"event_filtering"`
	UseNativeSDK      *webBoolConfig           `mapstructure:"use_native_sdk"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Ads destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("conversionID", "conversion_id"),
		converter.Simple("v2", "v2"),
		converter.Simple("allowIdentify", "allow_identify"),
		converter.Simple("sdkBaseUrl", "sdk_base_url"),
		converter.ArrayWithObjects("eventMappingFromConfig", "event_mapping_from_config", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("pageLoadConversions", "page_load_conversions", map[string]any{
			"conversionLabel": "label",
			"name":            "name",
		}),
		converter.ArrayWithObjects("clickEventConversions", "click_event_conversions", map[string]any{
			"conversionLabel": "label",
			"name":            "name",
		}),
		converter.Simple("defaultPageConversion", "default_page_conversion"),
		converter.Simple("dynamicRemarketing.web", "dynamic_remarketing.web"),
		converter.Simple("conversionLinker", "conversion_linker"),
		converter.Simple("sendPageView", "send_page_view"),
		converter.Simple("disableAdPersonalization", "disable_ad_personalization"),
		converter.Simple("enableConversionLabel", "enable_conversion_label"),
		converter.Simple("allowEnhancedConversions", "allow_enhanced_conversions"),
		converter.Simple("trackConversions", "track_conversions"),
		converter.Simple("enableConversionEventsFiltering", "enable_conversion_events_filtering"),
		converter.ArrayWithStrings("eventsToTrackConversions", "eventName", "events_to_track_conversions"),
		converter.Simple("trackDynamicRemarketing", "track_dynamic_remarketing"),
		converter.Simple("enableDynamicRemarketingEventsFiltering", "enable_dynamic_remarketing_events_filtering"),
		converter.ArrayWithStrings("eventsToTrackDynamicRemarketing", "eventName", "events_to_track_dynamic_remarketing"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
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
