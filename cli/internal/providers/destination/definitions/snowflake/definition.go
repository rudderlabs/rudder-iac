package snowflake

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json guards namespace with ^((?!pg_|PG_|pG_|Pg_).{0,64})$. RE2 has no
	// lookahead, so the reserved-prefix half becomes a reject pattern.
	funcs.NewPatternWithReject(
		"snowflake_namespace",
		`^(.{0,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be at most 64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)

	// Azure container naming: schema.json is ^(?=.{3,63}$)[a-z0-9]+(-[a-z0-9]+)*$.
	// RE2 has no lookahead, so the length bound moves into the reject pattern
	// while the accept pattern carries the character rule. The backend rejects
	// violations with a 400, so leaving this unenforced fails at apply instead of
	// at validate.
	funcs.NewPatternWithReject(
		"azure_container_name",
		`^[a-z0-9]+(-[a-z0-9]+)*$`,
		`^(.{0,2}|.{64,})$`,
		"must be 3-63 characters of lowercase letters, digits and single hyphens",
	)
}

// Source types from integrations-config destinations/snowflake/db-config.json
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

// excludeWindow mirrors the only genuinely nested object in the upstream config.
type excludeWindow struct {
	StartTime string `mapstructure:"start_time" validate:"omitempty"`
	EndTime   string `mapstructure:"end_time" validate:"omitempty"`
}

// snowflakeConfig is the local YAML config model. It is flat because the upstream
// config is flat: a real payload carries cloudProvider and roleBasedAuth as
// first-class keys and submits every provider's keys side by side (an AZURE
// destination still sends bucketName and iamRoleARN). Terraform groups them into
// s3/gcp/azure blocks, but that shape cannot represent such a payload — and the
// s3 definition, the verified reference, is flat for the same reason.
type snowflakeConfig struct {
	Account   string `mapstructure:"account" validate:"required,dynamic_or_pattern=single_line_100"`
	Database  string `mapstructure:"database" validate:"required,dynamic_or_pattern=single_line_100"`
	Warehouse string `mapstructure:"warehouse" validate:"required,dynamic_or_pattern=single_line_100"`
	User      string `mapstructure:"user" validate:"required,dynamic_or_pattern=single_line_100"`
	Role      string `mapstructure:"role" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Namespace string `mapstructure:"namespace" validate:"omitempty,dynamic_or_pattern=snowflake_namespace"`

	// schema.json gates password/privateKey on useKeyPairAuth. Neither carries a
	// usable literal pattern: password is `.*`, and privateKey's PEM regex has no
	// template branch, so enforcing it would reject a `{{ .VAR }}` reference.
	UseKeyPairAuth       *bool  `mapstructure:"use_key_pair_auth" validate:"required"`
	Password             string `mapstructure:"password" validate:"required_if=UseKeyPairAuth false"`
	PrivateKey           string `mapstructure:"private_key" validate:"required_if=UseKeyPairAuth true"`
	PrivateKeyPassphrase string `mapstructure:"private_key_passphrase" validate:"omitempty,pattern=single_line_100"`

	SyncFrequency string         `mapstructure:"sync_frequency" validate:"required,dynamic_or_oneof=5 10 15 30 60 180 360 720 1440"`
	SyncStartAt   string         `mapstructure:"sync_start_at" validate:"omitempty"`
	ExcludeWindow *excludeWindow `mapstructure:"exclude_window"`

	SkipTracksTable         *bool  `mapstructure:"skip_tracks_table"`
	SkipUsersTable          *bool  `mapstructure:"skip_users_table"`
	PreferAppend            *bool  `mapstructure:"prefer_append"`
	ManualSync              *bool  `mapstructure:"manual_sync"`
	JSONPaths               string `mapstructure:"json_paths" validate:"omitempty"`
	UnderscoreDivideNumbers *bool  `mapstructure:"underscore_divide_numbers"`
	AllowUsersContextTraits *bool  `mapstructure:"allow_users_context_traits"`

	// Object-storage staging. cloudProvider selects the provider, but upstream
	// keeps every provider's keys in the same flat object, so none of these are
	// gated on it.
	UseRudderStorage          *bool  `mapstructure:"use_rudder_storage" validate:"required"`
	CloudProvider             string `mapstructure:"cloud_provider" validate:"required_if=UseRudderStorage false,omitempty,dynamic_or_oneof=AWS GCP AZURE"`
	Prefix                    string `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	CleanupObjectStorageFiles *bool  `mapstructure:"cleanup_object_storage_files"`
	StorageIntegration        string `mapstructure:"storage_integration" validate:"omitempty,dynamic_or_pattern=single_line_100"`

	// AWS
	BucketName    string `mapstructure:"bucket_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	RoleBasedAuth *bool  `mapstructure:"role_based_auth"`
	IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	AccessKeyID   string `mapstructure:"access_key_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	AccessKey     string `mapstructure:"access_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	EnableSSE     *bool  `mapstructure:"enable_sse"`

	// GCP
	Credentials string `mapstructure:"credentials" validate:"omitempty"`

	// Azure
	ContainerName string `mapstructure:"container_name" validate:"omitempty,dynamic_or_pattern=azure_container_name"`
	AccountName   string `mapstructure:"account_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	AccountKey    string `mapstructure:"account_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	UseSASTokens  *bool  `mapstructure:"use_sas_tokens"`
	SASToken      string `mapstructure:"sas_token" validate:"omitempty"`

	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Snowflake warehouse destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("account", "account"),
		converter.Simple("database", "database"),
		converter.Simple("warehouse", "warehouse"),
		converter.Simple("user", "user"),
		converter.Simple("role", "role"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("useKeyPairAuth", "use_key_pair_auth"),
		converter.Simple("password", "password"),
		converter.Simple("privateKey", "private_key"),
		converter.Simple("privateKeyPassphrase", "private_key_passphrase"),
		converter.Simple("syncFrequency", "sync_frequency"),
		converter.Simple("syncStartAt", "sync_start_at"),
		converter.Simple("excludeWindow.excludeWindowStartTime", "exclude_window.start_time"),
		converter.Simple("excludeWindow.excludeWindowEndTime", "exclude_window.end_time"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("skipUsersTable", "skip_users_table"),
		converter.Simple("preferAppend", "prefer_append"),
		converter.Simple("manualSync", "manual_sync"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
		converter.Simple("useRudderStorage", "use_rudder_storage"),
		converter.Simple("cloudProvider", "cloud_provider"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("cleanupObjectStorageFiles", "cleanup_object_storage_files"),
		converter.Simple("storageIntegration", "storage_integration"),
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("roleBasedAuth", "role_based_auth"),
		converter.Simple("iamRoleARN", "iam_role_arn"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "access_key"),
		converter.Simple("enableSSE", "enable_sse"),
		converter.Simple("credentials", "credentials"),
		converter.Simple("containerName", "container_name"),
		converter.Simple("accountName", "account_name"),
		converter.Simple("accountKey", "account_key"),
		converter.Simple("useSASTokens", "use_sas_tokens"),
		converter.Simple("sasToken", "sas_token"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "snowflake",
		APIType:    "SNOWFLAKE",
		Version:    1,
		Properties: properties,
		// The full db-config secretKeys set, translated to snake_case. All eight
		// are top-level, so none depend on nested secret-path support.
		SecretKeys: []string{
			"password",
			"private_key",
			"private_key_passphrase",
			"access_key_id",
			"access_key",
			"account_key",
			"sas_token",
			"credentials",
		},
		NewConfig: func() any {
			return &snowflakeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
