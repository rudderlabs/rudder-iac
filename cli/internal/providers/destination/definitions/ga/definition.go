package ga

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	funcs.NewPattern(
		"google_analytics_tracking_id",
		`^(UA|YT|MO)-\d+-\d{0,100}$`,
		"must be a Universal Analytics tracking ID such as UA-123456-1",
	)
}

// Source types from integrations-config destinations/ga/db-config.json.
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

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type webString struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type webStringList struct {
	Web []string `mapstructure:"web" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
}

type fieldMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type serverSideIdentify struct {
	EventCategory string `mapstructure:"event_category" validate:"required_with=EventAction,omitempty,dynamic_or_pattern=single_line_100"`
	EventAction   string `mapstructure:"event_action" validate:"required_with=EventCategory,omitempty,dynamic_or_pattern=single_line_100"`
}

// googleAnalyticsConfig is the local YAML config model for legacy Google Analytics (Universal Analytics).
type googleAnalyticsConfig struct {
	TrackingID                  string                   `mapstructure:"tracking_id" validate:"required,dynamic_or_pattern=google_analytics_tracking_id"`
	RudderDeleteAccountID       string                   `mapstructure:"rudder_delete_account_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	DoubleClick                 *bool                    `mapstructure:"double_click"`
	EnhancedLinkAttribution     *bool                    `mapstructure:"enhanced_link_attribution"`
	IncludeSearch               *bool                    `mapstructure:"include_search"`
	ServerSideIdentify          *serverSideIdentify      `mapstructure:"server_side_identify"`
	DisableMD5                  *bool                    `mapstructure:"disable_md5"`
	AnonymizeIP                 *bool                    `mapstructure:"anonymize_ip"`
	EnhancedEcommerce           *bool                    `mapstructure:"enhanced_ecommerce"`
	NonInteraction              *bool                    `mapstructure:"non_interaction"`
	SendUserID                  *bool                    `mapstructure:"send_user_id"`
	EventFiltering              *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK                *useNativeSDK            `mapstructure:"use_native_sdk"`
	TrackCategorizedPages       *webBool                 `mapstructure:"track_categorized_pages"`
	TrackNamedPages             *webBool                 `mapstructure:"track_named_pages"`
	UseRichEventNames           *webBool                 `mapstructure:"use_rich_event_names"`
	SampleRate                  *webString               `mapstructure:"sample_rate"`
	SiteSpeedSampleRate         *webString               `mapstructure:"site_speed_sample_rate"`
	ResetCustomDimensionsOnPage *webStringList           `mapstructure:"reset_custom_dimensions_on_page"`
	SetAllMappedProps           *webBool                 `mapstructure:"set_all_mapped_props"`
	Domain                      *webString               `mapstructure:"domain"`
	Optimize                    *webString               `mapstructure:"optimize"`
	UseGoogleAMPClientID        *webBool                 `mapstructure:"use_google_amp_client_id"`
	NamedTracker                *webBool                 `mapstructure:"named_tracker"`
	Dimensions                  []fieldMapping           `mapstructure:"dimensions" validate:"omitempty,dive"`
	Metrics                     []fieldMapping           `mapstructure:"metrics" validate:"omitempty,dive"`
	ContentGroupings            []fieldMapping           `mapstructure:"content_groupings" validate:"omitempty,dive"`
	CustomMappings              []fieldMapping           `mapstructure:"custom_mappings" validate:"omitempty,dive"`
	ConsentManagement           common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the legacy Google Analytics (Universal Analytics) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("trackingID", "tracking_id"),
		converter.Simple("rudderDeleteAccountId", "rudder_delete_account_id"),
		converter.Simple("doubleClick", "double_click"),
		converter.Simple("enhancedLinkAttribution", "enhanced_link_attribution"),
		converter.Simple("includeSearch", "include_search"),
		converter.Simple("serverSideIdentifyEventCategory", "server_side_identify.event_category"),
		converter.Simple("serverSideIdentifyEventAction", "server_side_identify.event_action"),
		converter.Discriminator("enableServerSideIdentify", converter.DiscriminatorValues{
			"server_side_identify.event_category": true,
		}),
		converter.Simple("disableMd5", "disable_md5"),
		converter.Simple("anonymizeIp", "anonymize_ip"),
		converter.Simple("enhancedEcommerce", "enhanced_ecommerce"),
		converter.Simple("nonInteraction", "non_interaction"),
		converter.Simple("sendUserId", "send_user_id"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Gated(
			converter.Simple("trackCategorizedPages.web", "track_categorized_pages.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("trackNamedPages.web", "track_named_pages.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("useRichEventNames.web", "use_rich_event_names.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("sampleRate.web", "sample_rate.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("siteSpeedSampleRate.web", "site_speed_sample_rate.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithStrings("resetCustomDimensionsOnPage.web", "resetCustomDimensionsOnPage", "reset_custom_dimensions_on_page.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("setAllMappedProps.web", "set_all_mapped_props.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("domain.web", "domain.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("optimize.web", "optimize.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("useGoogleAmpClientId.web", "use_google_amp_client_id.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("namedTracker.web", "named_tracker.web"),
			common.SourceTypeWeb,
		),
		converter.ArrayWithObjects("dimensions", "dimensions", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("metrics", "metrics", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("contentGroupings", "content_groupings", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("customMappings", "custom_mappings", map[string]any{
			"from": "from",
			"to":   "to",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "ga",
		APIType:    "GA",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &googleAnalyticsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
