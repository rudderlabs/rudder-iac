package salesforce

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/salesforce/db-config.json.
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

// salesforceConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/salesforce defaultConfig; validation
// constraints mirror schema.json for terraform-mapped fields.
type salesforceConfig struct {
	UserName           string                   `mapstructure:"user_name" validate:"required,dynamic_or_pattern=single_line_100"`
	Password           string                   `mapstructure:"password" validate:"required,dynamic_or_pattern=single_line_100"`
	InitialAccessToken string                   `mapstructure:"initial_access_token" validate:"required,dynamic_or_pattern=single_line_100"`
	MapProperties      *bool                    `mapstructure:"map_properties"`
	Sandbox            *bool                    `mapstructure:"sandbox"`
	UseContactID       *bool                    `mapstructure:"use_contact_id"`
	ConnectionMode     common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement  common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Salesforce destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("userName", "user_name"),
		converter.Simple("password", "password"),
		converter.Simple("initialAccessToken", "initial_access_token"),
		converter.Simple("mapProperties", "map_properties"),
		converter.Simple("sandbox", "sandbox"),
		converter.Simple("useContactId", "use_contact_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "salesforce",
		APIType:    "SALESFORCE",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"password", "initial_access_token"},
		NewConfig: func() any {
			return &salesforceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
