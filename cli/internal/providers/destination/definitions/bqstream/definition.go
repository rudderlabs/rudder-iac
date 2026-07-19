package bqstream

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/bqstream/db-config.json
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

// bqstreamConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/bqstream defaultConfig; validation
// constraints mirror schema.json required fields.
type bqstreamConfig struct {
	ProjectID         string                   `mapstructure:"project_id" validate:"required"`
	DatasetID         string                   `mapstructure:"dataset_id" validate:"required"`
	TableID           string                   `mapstructure:"table_id" validate:"required"`
	InsertID          string                   `mapstructure:"insert_id"`
	Credentials       string                   `mapstructure:"credentials" validate:"required"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the BigQuery Stream destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("projectId", "project_id"),
		converter.Simple("datasetId", "dataset_id"),
		converter.Simple("tableId", "table_id"),
		converter.Simple("insertId", "insert_id", converter.SkipZeroValue),
		converter.Simple("credentials", "credentials"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "bqstream",
		APIType:    "BQSTREAM",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"credentials"},
		NewConfig: func() any {
			return &bqstreamConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
