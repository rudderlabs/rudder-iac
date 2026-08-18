package confluentcloud

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/confluent_cloud/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

// confluentCloudConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/confluent_cloud schema/defaultConfig;
// validations mirror schema.json.
type confluentCloudConfig struct {
	BootstrapServer   string                   `mapstructure:"bootstrap_server" validate:"required,pattern=single_line_100"`
	Topic             string                   `mapstructure:"topic" validate:"required,pattern=single_line_100"`
	APIKey            string                   `mapstructure:"api_key" validate:"required,pattern=single_line_100"`
	APISecret         string                   `mapstructure:"api_secret" validate:"required,pattern=single_line_100"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Confluent Cloud destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("bootstrapServer", "bootstrap_server"),
		converter.Simple("topic", "topic"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("apiSecret", "api_secret"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "confluent_cloud",
		APIType:    "CONFLUENT_CLOUD",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_secret", "api_key"},
		NewConfig: func() any {
			return &confluentCloudConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
