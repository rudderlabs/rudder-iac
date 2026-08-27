package gcs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/gcs/db-config.json
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

// gcsConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/gcs defaultConfig plus schema-declared source-scoped config such
// as connectionMode, exposed locally as connection_mode.
type gcsConfig struct {
	BucketName        string                   `mapstructure:"bucket_name" validate:"required,dynamic_or_pattern=single_line_100"`
	Prefix            string                   `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Credentials       string                   `mapstructure:"credentials"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Cloud Storage destination definition. GCS stays
// behind the unverified destination gate; source support remains restricted to the
// CLI-owned event-stream types above.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("credentials", "credentials"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "gcs",
		APIType:    "GCS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"credentials"},
		NewConfig: func() any {
			return &gcsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
