package iterable

import (
	"reflect"

	"github.com/go-playground/validator/v10"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
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

// Iterable is unusual upstream: the event-filter keys are web-scoped objects
// (whitelistedEvents.web, eventFilteringOption.web) rather than the flat
// arrays most destinations use. The local surface still follows the fleet
// convention — a nested event_filtering block with mutual exclusion and a
// derived option — with the web scoping expressed in the converters (Gated)
// instead of in the local key names.
type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type webInitIdentifier struct {
	Web string `mapstructure:"web" validate:"omitempty,oneof=email userId"`
}

type webHandleLinks struct {
	Web string `mapstructure:"web" validate:"omitempty,oneof=open-all-new-tab open-all-same-tab external-new-tab"`
}

type webCloseButtonPosition struct {
	Web string `mapstructure:"web" validate:"omitempty,oneof=top-right top-left"`
}

// iterableConfig is the local YAML config model. Field set mirrors terraform
// destination_iterable.go mappings; validation constraints mirror overlapping
// schema.json rules.
type iterableConfig struct {
	APIKey             string `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	DataCenter         string `mapstructure:"data_center" validate:"required,oneof=USDC EUDC"`
	PreferUserID       *bool  `mapstructure:"prefer_user_id" default:"true"`
	MergeNestedObjects *bool  `mapstructure:"merge_nested_objects" default:"true"`
	// db-config lists registerDeviceOrBrowserApiKey as the only secretKey.
	RegisterDeviceOrBrowserAPIKey string                   `mapstructure:"register_device_or_browser_api_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	MapToSingleEvent              *bool                    `mapstructure:"map_to_single_event" default:"true"`
	TrackAllPages                 *bool                    `mapstructure:"track_all_pages" default:"false"`
	TrackCategorizedPages         *bool                    `mapstructure:"track_categorized_pages" default:"true"`
	TrackNamedPages               *bool                    `mapstructure:"track_named_pages" default:"true"`
	UseNativeSDK                  webBool                  `mapstructure:"use_native_sdk"`
	InitialisationIdentifier      webInitIdentifier        `mapstructure:"initialisation_identifier"`
	GetInAppEventMapping          webStringList            `mapstructure:"get_in_app_event_mapping"`
	PurchaseEventMapping          webStringList            `mapstructure:"purchase_event_mapping"`
	SendTrackForInapp             webBool                  `mapstructure:"send_track_for_inapp"`
	AnimationDuration             webString                `mapstructure:"animation_duration"`
	DisplayInterval               webString                `mapstructure:"display_interval"`
	OnOpenScreenReaderMessage     webString                `mapstructure:"on_open_screen_reader_message"`
	OnOpenNodeToTakeFocus         webString                `mapstructure:"on_open_node_to_take_focus"`
	PackageName                   string                   `mapstructure:"package_name" validate:"iterable_package_name_required,omitempty,dynamic_or_pattern=single_line_100"`
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
	EventFiltering                *eventFiltering          `mapstructure:"event_filtering"`
	ConnectionMode                common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement             common.ConsentManagement `mapstructure:"consent_management"`
}

// packageNameConditional enforces schema.json's anyOf branch: when
// connection_mode.web is device, package_name is required. The condition is
// keyed on a map entry, which required_if cannot resolve, so it reads the
// sibling field off FieldLevel.Parent() — the same shape as ga4's
// sdkBaseURLConditional, and registered the same way, scoped to this
// definition via ConfigValidateFuncs.
//
// The tag must precede omitempty in the validate list: omitempty short-circuits
// every validator after it on an empty value, which is exactly the case this
// needs to reject.
func packageNameConditional(fl validator.FieldLevel) bool {
	parent := fl.Parent()
	if parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}

	field := parent.FieldByName("ConnectionMode")
	if !field.IsValid() {
		return true
	}
	connectionMode, _ := field.Interface().(common.ConnectionMode)
	if connectionMode["web"] != "device" {
		return true
	}
	return fl.Field().String() != ""
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
			converter.ArrayWithStrings("whitelistedEvents.web", "eventName", "event_filtering.whitelist"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithStrings("blacklistedEvents.web", "eventName", "event_filtering.blacklist"),
			common.SourceTypeWeb,
		),
		// Derived, never user-set; ungated because a Discriminator carries no
		// local key (the lists it derives from are gated above).
		converter.Discriminator("eventFilteringOption.web", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "iterable",
		APIType:    "ITERABLE",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"register_device_or_browser_api_key"},
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "iterable_package_name_required", Func: packageNameConditional},
		},
		NewConfig: func() any {
			return &iterableConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
