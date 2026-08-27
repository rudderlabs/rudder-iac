package s3datalake

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json guards bucketName with three negative lookaheads that RE2 cannot
	// express inline: disallow xn-- prefixes, double dots, and IPv4-looking names.
	funcs.NewPatternWithReject(
		"s3_datalake_bucket_name",
		`^[a-z0-9][a-z0-9-.]{1,61}[a-z0-9]$`,
		`(^xn--)|(^.*\.\..*$)|(^(\d+(\.|$)){4}$)`,
		"must be a valid S3 bucket name: 3-63 lowercase letters, digits, dots, or hyphens; must not start with xn--, contain consecutive dots, or look like an IPv4 address",
	)

	// schema.json constrains accessKeyID/accessKey to ^(.{1,})$ inside the
	// roleBasedAuth:false branch — non-empty and no line breaks, with no upper
	// bound, so none of the shared single_line_N patterns fit.
	funcs.NewPattern(
		"s3_datalake_access_key",
		`^(.{1,})$`,
		"must not be empty and must not contain line breaks",
	)

	// schema.json guards namespace with ^((?!pg_|PG_|pG_|Pg_).{0,64})$. RE2 has no
	// lookahead, so the reserved-prefix half becomes a reject pattern.
	funcs.NewPatternWithReject(
		"s3_datalake_namespace",
		`^(.{0,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be at most 64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)
}

// Source types from integrations-config destinations/s3_datalake/db-config.json.
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

// s3DatalakeConfig is the local YAML config model. The sync fields are flat
// because schema.json declares syncFrequency/syncStartAt as top-level API keys;
// Terraform's sync block is a provider-local shape.
type s3DatalakeConfig struct {
	BucketName string `mapstructure:"bucket_name" validate:"required,dynamic_or_pattern=s3_datalake_bucket_name"`
	UseGlue    *bool  `mapstructure:"use_glue" validate:"required"`
	Region     string `mapstructure:"region" validate:"required_if=UseGlue true"`
	Prefix     string `mapstructure:"prefix"`
	Namespace  string `mapstructure:"namespace" validate:"omitempty,dynamic_or_pattern=s3_datalake_namespace"`

	RoleBasedAuth *bool  `mapstructure:"role_based_auth" validate:"required"`
	IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"required_if=RoleBasedAuth true,omitempty,dynamic_or_pattern=single_line_100"`
	AccessKeyID   string `mapstructure:"access_key_id" validate:"required_if=RoleBasedAuth false,omitempty,dynamic_or_pattern=s3_datalake_access_key"`
	AccessKey     string `mapstructure:"access_key" validate:"required_if=RoleBasedAuth false,omitempty,dynamic_or_pattern=s3_datalake_access_key"`
	// db-config secretKeys lists password even though schema/defaultConfig/Terraform
	// do not expose it; keep it modelled so imports/specs can preserve and wrap it.
	Password string `mapstructure:"password"`

	EnableSSE *bool `mapstructure:"enable_sse" default:"false"`

	// schema.json requires only bucketName, so sync_frequency is optional here
	// even though sibling warehouse destinations mark it required.
	SyncFrequency string `mapstructure:"sync_frequency" validate:"omitempty,dynamic_or_oneof=5 10 15 30 60 180 360 720 1440" default:"180"`
	SyncStartAt   string `mapstructure:"sync_start_at"`

	SkipTracksTable           *bool                    `mapstructure:"skip_tracks_table" default:"false"`
	SkipUsersTable            *bool                    `mapstructure:"skip_users_table" default:"true"`
	TimeWindowLayout          string                   `mapstructure:"time_window_layout"`
	UnderscoreDivideNumbers   *bool                    `mapstructure:"underscore_divide_numbers" default:"false"`
	CleanupObjectStorageFiles *bool                    `mapstructure:"cleanup_object_storage_files" default:"false"`
	AllowUsersContextTraits   *bool                    `mapstructure:"allow_users_context_traits" default:"false"`
	ConnectionMode            common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement         common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Amazon S3 Datalake destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("useGlue", "use_glue"),
		converter.Simple("region", "region"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("roleBasedAuth", "role_based_auth"),
		converter.Simple("iamRoleARN", "iam_role_arn"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "access_key"),
		converter.Simple("password", "password"),
		converter.Simple("enableSSE", "enable_sse"),
		converter.Simple("syncFrequency", "sync_frequency"),
		converter.Simple("syncStartAt", "sync_start_at"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("skipUsersTable", "skip_users_table"),
		converter.Simple("timeWindowLayout", "time_window_layout"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("cleanupObjectStorageFiles", "cleanup_object_storage_files"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "s3_datalake",
		APIType:    "S3_DATALAKE",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"password", "access_key_id", "access_key"},
		NewConfig: func() any {
			return &s3DatalakeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
