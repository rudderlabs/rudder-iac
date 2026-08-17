package kinesis

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/kinesis/db-config.json
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

// kinesisConfig is the local YAML config model. Field set mirrors the
// terraform-mapped Kinesis API contract; validation combines schema.json string
// constraints with Terraform's auth-mode mutual exclusion.
type kinesisConfig struct {
	Region        string `mapstructure:"region" validate:"required,dynamic_or_pattern=single_line_100"`
	Stream        string `mapstructure:"stream" validate:"required,dynamic_or_pattern=single_line_100"`
	RoleBasedAuth *bool  `mapstructure:"role_based_auth" validate:"required"`
	IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"required_if=RoleBasedAuth true,excluded_if=RoleBasedAuth false,omitempty,dynamic_or_pattern=single_line_100"`
	AccessKeyID   string `mapstructure:"access_key_id" validate:"required_if=RoleBasedAuth false,excluded_if=RoleBasedAuth true,omitempty,dynamic_or_pattern=single_line_100"`
	AccessKey     string `mapstructure:"access_key" validate:"required_if=RoleBasedAuth false,excluded_if=RoleBasedAuth true,omitempty,dynamic_or_pattern=single_line_100"`
	UseMessageID  *bool  `mapstructure:"use_message_id"`

	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Amazon Kinesis destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("region", "region"),
		converter.Simple("stream", "stream"),
		converter.Simple("roleBasedAuth", "role_based_auth"),
		converter.Simple("iamRoleARN", "iam_role_arn"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "access_key"),
		converter.Simple("useMessageId", "use_message_id"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "kinesis",
		APIType:    "KINESIS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"access_key_id", "access_key"},
		NewConfig: func() any {
			return &kinesisConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
