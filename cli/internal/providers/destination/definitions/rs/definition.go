package rs

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json guards namespace with ^((?!pg_|PG_|pG_|Pg_).{0,64})$.
	// RE2 has no lookahead, so the reserved-prefix half becomes a reject pattern.
	funcs.NewPatternWithReject(
		"rs_namespace",
		`^(.{0,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be at most 64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)

	// UI/schema guards password-mode host with (?!.*\.ngrok\.io). A broad reject
	// keeps CLI validation aligned with the API instead of deferring failures to apply.
	funcs.NewPatternWithReject(
		"rs_host",
		`^(.{1,255})$`,
		`^.*\.ngrok\.io.*$`,
		"must be 1-255 characters, must not contain line breaks, and must not be an ngrok host",
	)

	funcs.NewPattern(
		"rs_single_line_255",
		`^(.{1,255})$`,
		"must be 1-255 characters and must not contain line breaks",
	)

	// schema.json uses negative lookaheads for S3 bucket names. RE2 cannot compile
	// those inline, so the disallowed cases live in the reject pattern.
	funcs.NewPatternWithReject(
		"rs_bucket_name",
		`^[a-z0-9][a-z0-9-.]{1,61}[a-z0-9]$`,
		`(^xn--)|(^.*\.\..*$)|(^(\d+(\.|$)){4}$)`,
		"must be a valid S3 bucket name: 3-63 lowercase letters, digits, dots, or hyphens; must not start with xn--, contain consecutive dots, or look like an IPv4 address",
	)

	funcs.NewPattern(
		"rs_prefix",
		`^[^\s]{0,100}$`,
		"must be at most 100 characters and must not contain whitespace",
	)
}

// Source types from integrations-config destinations/rs/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeReactNative,
	common.SourceTypeCloudSource,
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
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeCloudSource:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

// excludeWindow mirrors the only genuinely nested object in the upstream config.
type excludeWindow struct {
	StartTime string `mapstructure:"start_time" validate:"required"`
	EndTime   string `mapstructure:"end_time" validate:"required"`
}

// rsConfig is the local YAML config model. It is flat because the upstream
// Redshift warehouse config is flat except excludeWindow; terraform's sync and
// s3 blocks are provider-specific artifacts and cannot represent every persisted
// config key without erasing values on whole-config updates.
type rsConfig struct {
	// use_iam_for_auth and use_serverless are required even though schema.json
	// lists neither: its branches are gated on the key being present
	// (if.required), so an absent value satisfies upstream by requiring nothing.
	// required_if cannot express "absent or false", so relaxing them would stop
	// host/port/password and cluster_id/workgroup_name being enforced at all and
	// push those failures to the API. Both are checkboxes defaulted to false
	// upstream, so a real config always carries them.
	UseIAMForAuth *bool  `mapstructure:"use_iam_for_auth" validate:"required"`
	Host          string `mapstructure:"host" validate:"required_if=UseIAMForAuth false,omitempty,dynamic_or_pattern=rs_host"`
	Port          string `mapstructure:"port" validate:"required_if=UseIAMForAuth false,omitempty,dynamic_or_pattern=single_line_100"`
	Database      string `mapstructure:"database" validate:"required,dynamic_or_pattern=single_line_100"`
	User          string `mapstructure:"user" validate:"required,dynamic_or_pattern=single_line_100"`
	Password      string `mapstructure:"password" validate:"required_if=UseIAMForAuth false"`

	IAMRoleARNForAuth string `mapstructure:"iam_role_arn_for_auth" validate:"required_if=UseIAMForAuth true,omitempty,dynamic_or_pattern=single_line_100"`
	ClusterRegion     string `mapstructure:"cluster_region" validate:"required_if=UseIAMForAuth true,omitempty,dynamic_or_pattern=rs_single_line_255"`
	UseServerless     *bool  `mapstructure:"use_serverless" validate:"required_if=UseIAMForAuth true"`
	ClusterID         string `mapstructure:"cluster_id" validate:"required_if=UseIAMForAuth true UseServerless false,omitempty,dynamic_or_pattern=rs_single_line_255"`
	WorkgroupName     string `mapstructure:"workgroup_name" validate:"required_if=UseIAMForAuth true UseServerless true,omitempty,dynamic_or_pattern=rs_single_line_255"`

	Namespace string `mapstructure:"namespace" validate:"omitempty,dynamic_or_pattern=rs_namespace"`
	UseSSH    *bool  `mapstructure:"use_ssh"`
	SSHHost   string `mapstructure:"ssh_host" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=single_line_100"`
	SSHPort   string `mapstructure:"ssh_port" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=single_line_100"`
	SSHUser   string `mapstructure:"ssh_user" validate:"required_if=UseSSH true,omitempty,dynamic_or_pattern=single_line_100"`
	// ssh_public_key is emitted by the backend and may be long; schema.json requires
	// presence when SSH is enabled but declares no literal pattern.
	SSHPublicKey string `mapstructure:"ssh_public_key" validate:"required_if=UseSSH true"`

	SyncFrequency string         `mapstructure:"sync_frequency" validate:"required,dynamic_or_oneof=5 10 15 30 60 180 360 720 1440"`
	SyncStartAt   string         `mapstructure:"sync_start_at"`
	ExcludeWindow *excludeWindow `mapstructure:"exclude_window"`

	SkipTracksTable           *bool  `mapstructure:"skip_tracks_table"`
	SkipUsersTable            *bool  `mapstructure:"skip_users_table"`
	PreferAppend              *bool  `mapstructure:"prefer_append"`
	JSONPaths                 string `mapstructure:"json_paths"`
	UnderscoreDivideNumbers   *bool  `mapstructure:"underscore_divide_numbers"`
	AllowUsersContextTraits   *bool  `mapstructure:"allow_users_context_traits"`
	UseRudderStorage          *bool  `mapstructure:"use_rudder_storage" validate:"required"`
	BucketName                string `mapstructure:"bucket_name" validate:"required_if=UseRudderStorage false,omitempty,dynamic_or_pattern=rs_bucket_name"`
	IAMRoleARN                string `mapstructure:"iam_role_arn" validate:"required_if=UseRudderStorage false RoleBasedAuth true,omitempty,dynamic_or_pattern=single_line_100"`
	RoleBasedAuth             *bool  `mapstructure:"role_based_auth"`
	AccessKeyID               string `mapstructure:"access_key_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	AccessKey                 string `mapstructure:"access_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Prefix                    string `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=rs_prefix"`
	EnableSSE                 *bool  `mapstructure:"enable_sse"`
	CleanupObjectStorageFiles *bool  `mapstructure:"cleanup_object_storage_files"`

	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Redshift (API type RS) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("host", "host"),
		converter.Simple("port", "port"),
		converter.Simple("database", "database"),
		converter.Simple("user", "user"),
		converter.Simple("useIAMForAuth", "use_iam_for_auth"),
		converter.Simple("password", "password"),
		converter.Simple("iamRoleARNForAuth", "iam_role_arn_for_auth"),
		converter.Simple("clusterId", "cluster_id"),
		converter.Simple("clusterRegion", "cluster_region"),
		converter.Simple("useServerless", "use_serverless"),
		converter.Simple("workgroupName", "workgroup_name"),
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("iamRoleARN", "iam_role_arn"),
		converter.Simple("roleBasedAuth", "role_based_auth"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "access_key"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("useSSH", "use_ssh"),
		converter.Simple("sshHost", "ssh_host"),
		converter.Simple("sshPort", "ssh_port"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("skipUsersTable", "skip_users_table"),
		converter.Simple("sshUser", "ssh_user"),
		converter.Simple("sshPublicKey", "ssh_public_key"),
		converter.Simple("syncFrequency", "sync_frequency"),
		converter.Simple("syncStartAt", "sync_start_at"),
		converter.Simple("enableSSE", "enable_sse"),
		converter.Simple("preferAppend", "prefer_append"),
		converter.Simple("excludeWindow.excludeWindowStartTime", "exclude_window.start_time"),
		converter.Simple("excludeWindow.excludeWindowEndTime", "exclude_window.end_time"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("useRudderStorage", "use_rudder_storage"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("cleanupObjectStorageFiles", "cleanup_object_storage_files"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
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
