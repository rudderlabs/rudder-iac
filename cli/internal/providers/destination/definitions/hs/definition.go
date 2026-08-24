package hs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/hs/db-config.json,
// restricted to the types the CLI event-stream provider owns.
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

type hsConfig struct {
	APIVersion string `mapstructure:"api_version" validate:"required,dynamic_or_oneof=newApi legacyApi"`
	// accessToken's schema pattern (^(.{1,100})$) carries no {{ }} / env.
	// branch, unlike every other single_line_100 field here — plain pattern.
	AccessToken       string                   `mapstructure:"access_token" validate:"required,pattern=single_line_100"`
	HubID             string                   `mapstructure:"hub_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	LookupField       string                   `mapstructure:"lookup_field" validate:"required_if=APIVersion newApi,omitempty,dynamic_or_pattern=single_line_100"`
	DoAssociation     *bool                    `mapstructure:"do_association"`
	HubSpotEvents     []hubSpotEvent           `mapstructure:"hubspot_events" validate:"omitempty,dive"`
	EventFiltering    *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK      *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

type hubSpotEvent struct {
	RSEventName      string          `mapstructure:"rs_event_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	HubSpotEventName string          `mapstructure:"hubspot_event_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	EventProperties  []eventProperty `mapstructure:"event_properties" validate:"omitempty,dive"`
}

type eventProperty struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

// NewDefinition returns the HubSpot destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiVersion", "api_version"),
		converter.Simple("accessToken", "access_token"),
		converter.Simple("hubID", "hub_id"),
		converter.Simple("lookupField", "lookup_field"),
		converter.Simple("doAssociation", "do_association"),
		converter.ArrayWithObjects("hubspotEvents", "hubspot_events", map[string]any{
			"rsEventName":      "rs_event_name",
			"hubspotEventName": "hubspot_event_name",
			"eventProperties":  "event_properties",
		}),
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
		Type:       "hs",
		APIType:    "HS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"access_token"},
		NewConfig: func() any {
			return &hsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
