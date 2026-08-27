package facebookconversions

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/facebook_conversions/db-config.json.
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
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

// facebookConversionsConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/facebook_conversions defaultConfig;
// validation constraints mirror overlapping schema.json rules.
type facebookConversionsConfig struct {
	DatasetID              string                   `mapstructure:"dataset_id" validate:"required,dynamic_or_pattern=single_line_100"`
	AccessToken            string                   `mapstructure:"access_token" validate:"required,dynamic_or_pattern=single_line_500"`
	ActionSource           string                   `mapstructure:"action_source" validate:"omitempty,dynamic_or_oneof=website email app phone_call chat physical_store system_generated other" default:"website"`
	LimitedDataUsage       *bool                    `mapstructure:"limited_data_usage" default:"false"`
	TestDestination        *bool                    `mapstructure:"test_destination" default:"false"`
	TestEventCode          string                   `mapstructure:"test_event_code" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	RemoveExternalID       *bool                    `mapstructure:"remove_external_id" default:"false"`
	EventsToEvents         []eventMapping           `mapstructure:"events_to_events" validate:"omitempty,dive"`
	BlacklistPIIProperties []piiDenylistEntry       `mapstructure:"blacklist_pii_properties" validate:"omitempty,dive"`
	WhitelistPIIProperties []piiAllowlistEntry      `mapstructure:"whitelist_pii_properties" validate:"omitempty,dive"`
	ConnectionMode         common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement      common.ConsentManagement `mapstructure:"consent_management"`
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

// NewDefinition returns the Facebook Conversions destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("datasetId", "dataset_id"),
		converter.Simple("accessToken", "access_token"),
		converter.Simple("actionSource", "action_source"),
		converter.Simple("limitedDataUSage", "limited_data_usage"),
		converter.Simple("testDestination", "test_destination"),
		converter.Simple("testEventCode", "test_event_code"),
		converter.Simple("removeExternalId", "remove_external_id"),
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
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "facebook_conversions",
		APIType:    "FACEBOOK_CONVERSIONS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"access_token"},
		NewConfig: func() any {
			return &facebookConversionsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
