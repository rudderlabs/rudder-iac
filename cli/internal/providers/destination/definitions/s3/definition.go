package s3

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/s3/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns.
// sourceTypes is the supported-source-types list (integrations-config
// supportedSourceTypes ∩ CLI event-stream ownership).
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

// s3Config is the local YAML config model. Field set comes from TF
// destination_s3.go; validation constraints mirror overlapping schema.json rules.
type s3Config struct {
	BucketName        string                   `mapstructure:"bucket_name" validate:"required,min=1,max=100"`
	Prefix            string                   `mapstructure:"prefix" validate:"omitempty,max=100"`
	AccessKeyID       string                   `mapstructure:"access_key_id" validate:"omitempty,max=100"`
	AccessKey         string                   `mapstructure:"access_key" validate:"omitempty,max=100"`
	EnableSSE         *bool                    `mapstructure:"enable_sse"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the classic Amazon S3 destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("prefix", "prefix", converter.SkipZeroValue),
		converter.Simple("accessKeyID", "access_key_id", converter.SkipZeroValue),
		converter.Simple("accessKey", "access_key", converter.SkipZeroValue),
		converter.Simple("enableSSE", "enable_sse", converter.SkipZeroValue),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "s3",
		APIType:    "S3",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"access_key_id", "access_key"},
		NewConfig: func() any {
			return &s3Config{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
