package zendesk

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/zendesk/db-config.json
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
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

// zendeskConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/zendesk defaultConfig; validation
// constraints mirror schema.json.
type zendeskConfig struct {
	Email    string `mapstructure:"email" validate:"required,dynamic_or_pattern=single_line_100"`
	APIToken string `mapstructure:"api_token" validate:"required,dynamic_or_pattern=single_line_100"`
	Domain   string `mapstructure:"domain" validate:"required,dynamic_or_pattern=single_line_100"`
	// schema.json declares sourceName but terraform does not map it. Modelled
	// anyway: an unmodelled key is dropped from the update payload and erased
	// upstream on the first apply.
	SourceName                  string                   `mapstructure:"source_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	CreateUsersAsVerified       *bool                    `mapstructure:"create_users_as_verified"`
	SendGroupCallsWithoutUserID *bool                    `mapstructure:"send_group_calls_without_user_id"`
	RemoveUsersFromOrganization *bool                    `mapstructure:"remove_users_from_organization"`
	SearchByExternalID          *bool                    `mapstructure:"search_by_external_id"`
	ConnectionMode              common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement           common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Zendesk destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("email", "email"),
		converter.Simple("apiToken", "api_token"),
		converter.Simple("domain", "domain"),
		converter.Simple("sourceName", "source_name"),
		converter.Simple("createUsersAsVerified", "create_users_as_verified"),
		converter.Simple("sendGroupCallsWithoutUserId", "send_group_calls_without_user_id"),
		converter.Simple("removeUsersFromOrganization", "remove_users_from_organization"),
		converter.Simple("searchByExternalId", "search_by_external_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "zendesk",
		APIType:    "ZENDESK",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_token"},
		NewConfig: func() any {
			return &zendeskConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
