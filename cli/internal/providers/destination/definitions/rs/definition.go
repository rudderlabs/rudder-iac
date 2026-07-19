package rs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/rs/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns
// (same set as S3; amp/shopify/cloud_source dropped).
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

// rsSyncConfig is the local YAML sync block (terraform sync { ... }).
type rsSyncConfig struct {
	Frequency              string `mapstructure:"frequency" validate:"required,dynamic_or_oneof=5 15 30 60 180 360 720 1440"`
	StartAt                string `mapstructure:"start_at" validate:"omitempty"`
	ExcludeWindowStartTime string `mapstructure:"exclude_window_start_time" validate:"omitempty"`
	ExcludeWindowEndTime   string `mapstructure:"exclude_window_end_time" validate:"omitempty"`
}

// rsConfig is the local YAML config model. Field set mirrors the terraform
// redshift destination surface; validation constraints mirror overlapping
// schema.json rules for those mapped fields.
//
// S3 object-storage fields are flat top-level keys (API snake_case) rather than
// terraform's s3 {} block so SecretKeys can wrap access_key_id / access_key —
// the CLI secret machinery only supports top-level keys.
type rsConfig struct {
	Host              string                   `mapstructure:"host" validate:"required,min=1,max=255"`
	Port              string                   `mapstructure:"port" validate:"required,min=1,max=100"`
	Database          string                   `mapstructure:"database" validate:"required,min=1,max=100"`
	User              string                   `mapstructure:"user" validate:"required,min=1,max=100"`
	Password          string                   `mapstructure:"password" validate:"required,min=1"`
	Namespace         string                   `mapstructure:"namespace" validate:"omitempty,max=64"`
	EnableSSE         *bool                    `mapstructure:"enable_sse"`
	UseRudderStorage  *bool                    `mapstructure:"use_rudder_storage" validate:"required"`
	Sync              rsSyncConfig             `mapstructure:"sync" validate:"required"`
	BucketName        string                   `mapstructure:"bucket_name" validate:"required_if=UseRudderStorage false,omitempty,max=100"`
	AccessKeyID       string                   `mapstructure:"access_key_id" validate:"omitempty,max=100"`
	AccessKey         string                   `mapstructure:"access_key" validate:"omitempty,max=100"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Redshift (API type RS) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("host", "host"),
		converter.Simple("port", "port"),
		converter.Simple("database", "database"),
		converter.Simple("user", "user"),
		converter.Simple("password", "password"),
		converter.Simple("namespace", "namespace", converter.SkipZeroValue),
		converter.Simple("enableSSE", "enable_sse", converter.SkipZeroValue),
		converter.Simple("useRudderStorage", "use_rudder_storage"),
		converter.Simple("syncFrequency", "sync.frequency"),
		converter.Simple("syncStartAt", "sync.start_at", converter.SkipZeroValue),
		converter.Simple("excludeWindow.excludeWindowStartTime", "sync.exclude_window_start_time", converter.SkipZeroValue),
		converter.Simple("excludeWindow.excludeWindowEndTime", "sync.exclude_window_end_time", converter.SkipZeroValue),
		converter.Simple("bucketName", "bucket_name", converter.SkipZeroValue),
		converter.Simple("accessKeyID", "access_key_id", converter.SkipZeroValue),
		converter.Simple("accessKey", "access_key", converter.SkipZeroValue),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "rs",
		APIType:    "RS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"password", "access_key_id", "access_key"},
		NewConfig: func() any {
			return &rsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
