package googlepubsub

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/googlepubsub/db-config.json
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

// googlePubSubConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/googlepubsub defaultConfig; validation
// constraints mirror schema.json for overlapping terraform-mapped fields.
type googlePubSubConfig struct {
	ProjectID           string                   `mapstructure:"project_id" validate:"required,dynamic_or_pattern=single_line_100"`
	Credentials         string                   `mapstructure:"credentials" validate:"required"`
	EventToTopicMap     []eventMapping           `mapstructure:"event_to_topic_map" validate:"omitempty,dive"`
	EventToAttributeMap []eventMapping           `mapstructure:"event_to_attribute_map" validate:"omitempty,dive"`
	ConnectionMode      common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement   common.ConsentManagement `mapstructure:"consent_management"`
}

type eventMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

// NewDefinition returns the Google Cloud Pub/Sub destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("projectId", "project_id"),
		converter.Simple("credentials", "credentials"),
		converter.ArrayWithObjects("eventToTopicMap", "event_to_topic_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("eventToAttributesMap", "event_to_attribute_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "googlepubsub",
		APIType:    "GOOGLEPUBSUB",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"credentials"},
		NewConfig: func() any {
			return &googlePubSubConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
