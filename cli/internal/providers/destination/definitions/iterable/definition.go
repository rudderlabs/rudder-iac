package iterable

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/iterable/db-config.json
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

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type webString struct {
	Web string `mapstructure:"web"`
}

type webStringList struct {
	Web []string `mapstructure:"web" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
}

// Iterable is unusual: event filtering is web-gated and web-keyed, rather than
// the flat arrays plus discriminator most destinations use.
type webEventFilteringOption struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=disable whitelistedEvents blacklistedEvents"`
}

type webInitIdentifier struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=email userId"`
}

type webHandleLinks struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=open-all-new-tab open-all-same-tab external-new-tab"`
}

type webCloseButtonPosition struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=top-right top-left"`
}

// iterableConfig is the local YAML config model. Field set mirrors terraform
// destination_iterable.go mappings; validation constraints mirror overlapping
// schema.json rules.
type iterableConfig struct {
	APIKey             string `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	DataCenter         string `mapstructure:"data_center" validate:"required,dynamic_or_oneof=USDC EUDC"`
	PreferUserID       *bool  `mapstructure:"prefer_user_id"`
	MergeNestedObjects *bool  `mapstructure:"merge_nested_objects"`
	// db-config lists registerDeviceOrBrowserApiKey as the only secretKey.
	RegisterDeviceOrBrowserAPIKey string                   `mapstructure:"register_device_or_browser_api_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	MapToSingleEvent              *bool                    `mapstructure:"map_to_single_event"`
	TrackAllPages                 *bool                    `mapstructure:"track_all_pages"`
	TrackCategorizedPages         *bool                    `mapstructure:"track_categorized_pages"`
	TrackNamedPages               *bool                    `mapstructure:"track_named_pages"`
	UseNativeSDK                  webBool                  `mapstructure:"use_native_sdk"`
	InitialisationIdentifier      webInitIdentifier        `mapstructure:"initialisation_identifier"`
	GetInAppEventMapping          webStringList            `mapstructure:"get_in_app_event_mapping"`
	PurchaseEventMapping          webStringList            `mapstructure:"purchase_event_mapping"`
	SendTrackForInapp             webBool                  `mapstructure:"send_track_for_inapp"`
	AnimationDuration             webString                `mapstructure:"animation_duration"`
	DisplayInterval               webString                `mapstructure:"display_interval"`
	OnOpenScreenReaderMessage     webString                `mapstructure:"on_open_screen_reader_message"`
	OnOpenNodeToTakeFocus         webString                `mapstructure:"on_open_node_to_take_focus"`
	PackageName                   string                   `mapstructure:"package_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	RightOffset                   webString                `mapstructure:"right_offset"`
	TopOffset                     webString                `mapstructure:"top_offset"`
	BottomOffset                  webString                `mapstructure:"bottom_offset"`
	HandleLinks                   webHandleLinks           `mapstructure:"handle_links"`
	CloseButtonColor              webString                `mapstructure:"close_button_color"`
	CloseButtonSize               webString                `mapstructure:"close_button_size"`
	CloseButtonColorTopOffset     webString                `mapstructure:"close_button_color_top_offset"`
	CloseButtonColorSideOffset    webString                `mapstructure:"close_button_color_side_offset"`
	IconPath                      webString                `mapstructure:"icon_path"`
	IsRequiredToDismissMessage    webBool                  `mapstructure:"is_required_to_dismiss_message"`
	CloseButtonPosition           webCloseButtonPosition   `mapstructure:"close_button_position"`
	EventFilteringOption          webEventFilteringOption  `mapstructure:"event_filtering_option"`
	WhitelistedEvents             webStringList            `mapstructure:"whitelisted_events"`
	BlacklistedEvents             webStringList            `mapstructure:"blacklisted_events"`
	ConsentManagement             common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Iterable destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiKey", "api_key"),
		converter.Simple("dataCenter", "data_center"),
		converter.Simple("preferUserId", "prefer_user_id"),
		converter.Simple("mergeNestedObjects", "merge_nested_objects"),
		converter.Simple("registerDeviceOrBrowserApiKey", "register_device_or_browser_api_key"),
		converter.Simple("mapToSingleEvent", "map_to_single_event"),
		converter.Simple("trackAllPages", "track_all_pages"),
		converter.Simple("trackCategorisedPages", "track_categorized_pages"),
		converter.Simple("trackNamedPages", "track_named_pages"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Gated(
			converter.Simple("initialisationIdentifier.web", "initialisation_identifier.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithStrings("getInAppEventMapping.web", "eventName", "get_in_app_event_mapping.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithStrings("purchaseEventMapping.web", "eventName", "purchase_event_mapping.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("sendTrackForInapp.web", "send_track_for_inapp.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("animationDuration.web", "animation_duration.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("displayInterval.web", "display_interval.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("onOpenScreenReaderMessage.web", "on_open_screen_reader_message.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("onOpenNodeToTakeFocus.web", "on_open_node_to_take_focus.web"),
			common.SourceTypeWeb,
		),
		// packageName is in destConfig.defaultConfig — not source-type-gated.
		converter.Simple("packageName", "package_name"),
		converter.Gated(
			converter.Simple("rightOffset.web", "right_offset.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("topOffset.web", "top_offset.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("bottomOffset.web", "bottom_offset.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("handleLinks.web", "handle_links.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonColor.web", "close_button_color.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonSize.web", "close_button_size.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonColorTopOffset.web", "close_button_color_top_offset.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonColorSideOffset.web", "close_button_color_side_offset.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("iconPath.web", "icon_path.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("isRequiredToDismissMessage.web", "is_required_to_dismiss_message.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonPosition.web", "close_button_position.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("eventFilteringOption.web", "event_filtering_option.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithStrings("whitelistedEvents.web", "eventName", "whitelisted_events.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithStrings("blacklistedEvents.web", "eventName", "blacklisted_events.web"),
			common.SourceTypeWeb,
		),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "iterable",
		APIType:    "ITERABLE",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"register_device_or_browser_api_key"},
		NewConfig: func() any {
			return &iterableConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
