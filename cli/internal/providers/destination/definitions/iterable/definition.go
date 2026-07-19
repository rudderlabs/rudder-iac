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
	Web []string `mapstructure:"web" validate:"omitempty,dive,max=100"`
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

type webPackageName struct {
	Web string `mapstructure:"web" validate:"omitempty,max=100"`
}

// iterableConfig is the local YAML config model. Field set mirrors terraform
// destination_iterable.go mappings; validation constraints mirror overlapping
// schema.json rules.
type iterableConfig struct {
	APIKey                     string                   `mapstructure:"api_key" validate:"required,min=1,max=100"`
	MapToSingleEvent           *bool                    `mapstructure:"map_to_single_event"`
	TrackAllPages              *bool                    `mapstructure:"track_all_pages"`
	TrackCategorizedPages      *bool                    `mapstructure:"track_categorized_pages"`
	TrackNamedPages            *bool                    `mapstructure:"track_named_pages"`
	UseNativeSDK               webBool                  `mapstructure:"use_native_sdk"`
	InitialisationIdentifier   webInitIdentifier        `mapstructure:"initialisation_identifier"`
	GetInAppEventMapping       webStringList            `mapstructure:"get_in_app_event_mapping"`
	PurchaseEventMapping       webStringList            `mapstructure:"purchase_event_mapping"`
	SendTrackForInapp          webBool                  `mapstructure:"send_track_for_inapp"`
	AnimationDuration          webString                `mapstructure:"animation_duration"`
	DisplayInterval            webString                `mapstructure:"display_interval"`
	OnOpenScreenReaderMessage  webString                `mapstructure:"on_open_screen_reader_message"`
	OnOpenNodeToTakeFocus      webString                `mapstructure:"on_open_node_to_take_focus"`
	PackageName                webPackageName           `mapstructure:"package_name"`
	RightOffset                webString                `mapstructure:"right_offset"`
	TopOffset                  webString                `mapstructure:"top_offset"`
	BottomOffset               webString                `mapstructure:"bottom_offset"`
	HandleLinks                webHandleLinks           `mapstructure:"handle_links"`
	CloseButtonColor           webString                `mapstructure:"close_button_color"`
	CloseButtonSize            webString                `mapstructure:"close_button_size"`
	CloseButtonColorTopOffset  webString                `mapstructure:"close_button_color_top_offset"`
	CloseButtonColorSideOffset webString                `mapstructure:"close_button_color_side_offset"`
	IconPath                   webString                `mapstructure:"icon_path"`
	IsRequiredToDismissMessage webBool                  `mapstructure:"is_required_to_dismiss_message"`
	CloseButtonPosition        webCloseButtonPosition   `mapstructure:"close_button_position"`
	ConsentManagement          common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Iterable destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiKey", "api_key", converter.SkipZeroValue),
		converter.Simple("mapToSingleEvent", "map_to_single_event"),
		converter.Simple("trackAllPages", "track_all_pages", converter.SkipZeroValue),
		converter.Simple("trackCategorisedPages", "track_categorized_pages"),
		converter.Simple("trackNamedPages", "track_named_pages"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Gated(
			converter.Simple("initialisationIdentifier.web", "initialisation_identifier.web", converter.SkipZeroValue),
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
			converter.Simple("animationDuration.web", "animation_duration.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("displayInterval.web", "display_interval.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("onOpenScreenReaderMessage.web", "on_open_screen_reader_message.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("onOpenNodeToTakeFocus.web", "on_open_node_to_take_focus.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		// packageName is in destConfig.defaultConfig — not source-type-gated.
		converter.Simple("packageName.web", "package_name.web", converter.SkipZeroValue),
		converter.Gated(
			converter.Simple("rightOffset.web", "right_offset.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("topOffset.web", "top_offset.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("bottomOffset.web", "bottom_offset.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("handleLinks.web", "handle_links.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonColor.web", "close_button_color.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonSize.web", "close_button_size.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonColorTopOffset.web", "close_button_color_top_offset.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonColorSideOffset.web", "close_button_color_side_offset.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("iconPath.web", "icon_path.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("isRequiredToDismissMessage.web", "is_required_to_dismiss_message.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("closeButtonPosition.web", "close_button_position.web", converter.SkipZeroValue),
			common.SourceTypeWeb,
		),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "iterable",
		APIType:    "ITERABLE",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &iterableConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
