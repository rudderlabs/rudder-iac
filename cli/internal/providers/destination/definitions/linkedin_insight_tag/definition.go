package linkedininsighttag

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/linkedin_insight_tag/db-config.json.
var sourceTypes = []string{
	common.SourceTypeWeb,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb: {"device"},
}

type eventToConversionIDMapping struct {
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

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff.
//
// connection_mode is deliberately absent too. db-config lists connectionMode
// under web, but schema.json declares no such property, and schema.json is the
// authority on the config surface. ConnectionModes below still advertises the
// supported mode as metadata.
//
// linkedinInsightTagConfig is the local YAML config model. Field set mirrors the
// keys upstream declares in schema.json and db-config.json destConfig; validation
// constraints mirror schema.json.
type linkedinInsightTagConfig struct {
	PartnerID              string                       `mapstructure:"partner_id" validate:"required"`
	EventToConversionIDMap []eventToConversionIDMapping `mapstructure:"event_to_conversion_id_map" validate:"omitempty,dive"`
	EventFiltering         *eventFiltering              `mapstructure:"event_filtering"`
	UseNativeSDK           *useNativeSDK                `mapstructure:"use_native_sdk"`
	ConsentManagement      common.ConsentManagement     `mapstructure:"consent_management"`
}

// NewDefinition returns the LinkedIn Insight Tag destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("partnerId", "partner_id"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.ArrayWithObjects("eventToConversionIdMap", "event_to_conversion_id_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "linkedin_insight_tag",
		APIType:    "LINKEDIN_INSIGHT_TAG",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &linkedinInsightTagConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
