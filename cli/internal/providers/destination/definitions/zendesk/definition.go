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

// zendeskConfig is the local YAML config model. Field set mirrors terraform
// destination_zendesk mappings; validation constraints mirror schema.json.
type zendeskConfig struct {
	Email                       string                   `mapstructure:"email" validate:"required,min=1,max=100"`
	APIToken                    string                   `mapstructure:"api_token" validate:"required,min=1,max=100"`
	Domain                      string                   `mapstructure:"domain" validate:"required,min=1,max=100"`
	CreateUsersAsVerified       *bool                    `mapstructure:"create_users_as_verified"`
	SendGroupCallsWithoutUserID *bool                    `mapstructure:"send_group_calls_without_user_id"`
	RemoveUsersFromOrganization *bool                    `mapstructure:"remove_users_from_organization"`
	SearchByExternalID          *bool                    `mapstructure:"search_by_external_id"`
	ConsentManagement           common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Zendesk destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("email", "email"),
		converter.Simple("apiToken", "api_token"),
		converter.Simple("domain", "domain"),
		converter.Simple("createUsersAsVerified", "create_users_as_verified", converter.SkipZeroValue),
		converter.Simple("sendGroupCallsWithoutUserId", "send_group_calls_without_user_id", converter.SkipZeroValue),
		converter.Simple("removeUsersFromOrganization", "remove_users_from_organization", converter.SkipZeroValue),
		converter.Simple("searchByExternalId", "search_by_external_id", converter.SkipZeroValue),
	}
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
