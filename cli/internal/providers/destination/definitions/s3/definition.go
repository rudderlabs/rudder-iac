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

// s3Config is the local YAML config model. Field set mirrors integrations-config
// destinations/s3 defaultConfig (schema.json / db-config.json); validation
// constraints mirror overlapping schema.json rules plus explicit auth-mode
// mutual exclusion (role ARN vs access keys).
type s3Config struct {
	BucketName    string `mapstructure:"bucket_name" validate:"required,min=1,max=100"`
	Prefix        string `mapstructure:"prefix" validate:"omitempty,max=100"`
	RoleBasedAuth *bool  `mapstructure:"role_based_auth" validate:"required"`
	// IAM role ARN is required when role-based auth is on; forbidden when off.
	IAMRoleARN string `mapstructure:"iam_role_arn" validate:"required_if=RoleBasedAuth true,max=100"`
	// Access keys are required for key-based auth and forbidden when role-based
	// auth is on. Import/export never invents these secrets — users must supply
	// them (e.g. via {{ .VAR }} + a var file) when role_based_auth is false.
	AccessKeyID       string                   `mapstructure:"access_key_id" validate:"required_if=RoleBasedAuth false,max=100"`
	AccessKey         string                   `mapstructure:"access_key" validate:"required_if=RoleBasedAuth false,max=100"`
	EnableSSE         *bool                    `mapstructure:"enable_sse"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the classic Amazon S3 destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("roleBasedAuth", "role_based_auth"),
		converter.Simple("iamRoleARN", "iam_role_arn"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "access_key"),
		converter.Simple("enableSSE", "enable_sse"),
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
