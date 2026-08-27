package marketo

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/marketo/db-config.json.
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

// marketoConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/marketo defaultConfig; validation constraints mirror schema.json
// plus Terraform-required nested mapping fields.
type marketoConfig struct {
	AccountID                 string                   `mapstructure:"account_id" validate:"required,dynamic_or_pattern=single_line_100"`
	ClientID                  string                   `mapstructure:"client_id" validate:"required,dynamic_or_pattern=single_line_100"`
	ClientSecret              string                   `mapstructure:"client_secret" validate:"required,dynamic_or_pattern=single_line_100"`
	TrackAnonymousEvents      *bool                    `mapstructure:"track_anonymous_events" default:"false"`
	CreateIfNotExist          *bool                    `mapstructure:"create_if_not_exist" default:"true"`
	RudderEventsMapping       []rudderEventMapping     `mapstructure:"rudder_events_mapping" validate:"omitempty,dive"`
	LeadTraitMapping          []fieldMapping           `mapstructure:"lead_trait_mapping" validate:"omitempty,dive"`
	CustomActivityPropertyMap []fieldMapping           `mapstructure:"custom_activity_property_map" validate:"omitempty,dive"`
	ConnectionMode            common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement         common.ConsentManagement `mapstructure:"consent_management"`
}

// Terraform marks these nested fields Required, but schema.json declares them as
// plain strings with no constraint at all. Validation follows schema.json, so
// nothing is enforced here — otherwise a remote config holding a partially
// filled mapping row would import to a spec the CLI rejects.
type rudderEventMapping struct {
	Event             string `mapstructure:"event"`
	MarketoPrimaryKey string `mapstructure:"marketo_primarykey"`
	MarketoActivityID string `mapstructure:"marketo_activity_id"`
}

type fieldMapping struct {
	From string `mapstructure:"from"`
	To   string `mapstructure:"to"`
}

// NewDefinition returns the Marketo destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("accountId", "account_id"),
		converter.Simple("clientId", "client_id"),
		converter.Simple("clientSecret", "client_secret"),
		converter.Simple("trackAnonymousEvents", "track_anonymous_events"),
		converter.Simple("createIfNotExist", "create_if_not_exist"),
		converter.ArrayWithObjects("rudderEventsMapping", "rudder_events_mapping", map[string]any{
			"event":             "event",
			"marketoPrimarykey": "marketo_primarykey",
			"marketoActivityId": "marketo_activity_id",
		}),
		converter.ArrayWithObjects("leadTraitMapping", "lead_trait_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("customActivityPropertyMap", "custom_activity_property_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "marketo",
		APIType:    "MARKETO",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"client_secret"},
		NewConfig: func() any {
			return &marketoConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
