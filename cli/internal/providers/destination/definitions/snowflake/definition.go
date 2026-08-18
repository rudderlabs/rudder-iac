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
}

// Source types from integrations-config destinations/snowflake/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns
// (same set as S3).
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

type syncConfig struct {
	Frequency              string `mapstructure:"frequency" validate:"required,dynamic_or_oneof=5 10 15 30 60 180 360 720 1440"`
	StartAt                string `mapstructure:"start_at" validate:"omitempty"`
	ExcludeWindowStartTime string `mapstructure:"exclude_window_start_time" validate:"omitempty"`
	ExcludeWindowEndTime   string `mapstructure:"exclude_window_end_time" validate:"omitempty"`
}

type roleBasedAuthConfig struct {
	// Local key derives from the API key iamRoleARN, matching s3 and kinesis;
	// terraform spells its own local key i_am_role_arn.
	IAMRoleARN string `mapstructure:"iam_role_arn" validate:"required,dynamic_or_pattern=single_line_100"`
}

type s3StorageConfig struct {
	BucketName              string               `mapstructure:"bucket_name" validate:"required,dynamic_or_pattern=single_line_100"`
	AccessKeyID             string               `mapstructure:"access_key_id" validate:"required_with=AccessKey,excluded_with=RoleBasedAuthentication,omitempty,dynamic_or_pattern=single_line_100"`
	AccessKey               string               `mapstructure:"access_key" validate:"required_with=AccessKeyID,excluded_with=RoleBasedAuthentication,omitempty,dynamic_or_pattern=single_line_100"`
	EnableSSE               *bool                `mapstructure:"enable_sse"`
	RoleBasedAuthentication *roleBasedAuthConfig `mapstructure:"role_based_authentication" validate:"excluded_with=AccessKeyID"`
	StorageIntegration      string               `mapstructure:"storage_integration" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type gcpStorageConfig struct {
	BucketName         string `mapstructure:"bucket_name" validate:"required,dynamic_or_pattern=single_line_100"`
	Credentials        string `mapstructure:"credentials" validate:"required"`
	StorageIntegration string `mapstructure:"storage_integration" validate:"required,dynamic_or_pattern=single_line_100"`
}

type azureStorageConfig struct {
	ContainerName      string `mapstructure:"container_name" validate:"required,dynamic_or_pattern=single_line_100"`
	AccountName        string `mapstructure:"account_name" validate:"required,dynamic_or_pattern=single_line_100"`
	AccountKey         string `mapstructure:"account_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	StorageIntegration string `mapstructure:"storage_integration" validate:"required,dynamic_or_pattern=single_line_100"`
	// useSASTokens/sasToken are in db-config defaultConfig (and useSASTokens in
	// schema.json's AZURE branch) but unmapped by terraform; modelled so a CLI
	// apply does not erase them.
	UseSASTokens *bool  `mapstructure:"use_sas_tokens"`
	SASToken     string `mapstructure:"sas_token" validate:"omitempty"`
}

// snowflakeConfig is the local YAML config model. Nested s3/gcp/azure/sync
// blocks mirror terraform-provider-rudderstack destination_snowflake.go shape
// (without TF list indices). Validation constraints come from schema.json.
type snowflakeConfig struct {
	Account   string `mapstructure:"account" validate:"required,dynamic_or_pattern=single_line_100"`
	Database  string `mapstructure:"database" validate:"required,dynamic_or_pattern=single_line_100"`
	Warehouse string `mapstructure:"warehouse" validate:"required,dynamic_or_pattern=single_line_100"`
	User      string `mapstructure:"user" validate:"required,dynamic_or_pattern=single_line_100"`
	// schema.json gates password/privateKey on useKeyPairAuth. Neither carries a
	// usable literal pattern: password is `.*`, and privateKey's PEM regex has no
	// template branch, so enforcing it would reject a `{{ .VAR }}` reference.
	UseKeyPairAuth       *bool                    `mapstructure:"use_key_pair_auth" validate:"required"`
	Password             string                   `mapstructure:"password" validate:"required_if=UseKeyPairAuth false"`
	PrivateKey           string                   `mapstructure:"private_key" validate:"required_if=UseKeyPairAuth true"`
	PrivateKeyPassphrase string                   `mapstructure:"private_key_passphrase" validate:"omitempty,pattern=single_line_100"`
	Role                 string                   `mapstructure:"role" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Namespace            string                   `mapstructure:"namespace" validate:"omitempty,dynamic_or_pattern=snowflake_namespace"`
	Sync                 syncConfig               `mapstructure:"sync"`
	SkipTracksTable      *bool                    `mapstructure:"skip_tracks_table"`
	SkipUsersTable       *bool                    `mapstructure:"skip_users_table"`
	PreferAppend         *bool                    `mapstructure:"prefer_append"`
	JSONPaths            string                   `mapstructure:"json_paths" validate:"omitempty"`
	ManualSync           *bool                    `mapstructure:"manual_sync"`
	UseRudderStorage     *bool                    `mapstructure:"use_rudder_storage" validate:"required"`
	Prefix               string                   `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	// Declared in db-config defaultConfig and schema.json but unmapped by
	// terraform; modelled so a CLI apply does not erase UI-set values.
	CleanupObjectStorageFiles *bool                    `mapstructure:"cleanup_object_storage_files"`
	UnderscoreDivideNumbers   *bool                    `mapstructure:"underscore_divide_numbers"`
	AllowUsersContextTraits   *bool                    `mapstructure:"allow_users_context_traits"`
	S3                        *s3StorageConfig         `mapstructure:"s3"`
	GCP                       *gcpStorageConfig        `mapstructure:"gcp"`
	Azure                     *azureStorageConfig      `mapstructure:"azure"`
	ConsentManagement         common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Snowflake warehouse destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("account", "account"),
		converter.Simple("database", "database"),
		converter.Simple("warehouse", "warehouse"),
		converter.Simple("user", "user"),
		converter.Simple("useKeyPairAuth", "use_key_pair_auth"),
		converter.Simple("password", "password"),
		converter.Simple("privateKey", "private_key"),
		converter.Simple("privateKeyPassphrase", "private_key_passphrase"),
		converter.Simple("role", "role"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("syncFrequency", "sync.frequency"),
		converter.Simple("syncStartAt", "sync.start_at"),
		converter.Simple("excludeWindow.excludeWindowStartTime", "sync.exclude_window_start_time"),
		converter.Simple("excludeWindow.excludeWindowEndTime", "sync.exclude_window_end_time"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("skipUsersTable", "skip_users_table"),
		converter.Simple("preferAppend", "prefer_append"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("manualSync", "manual_sync"),
		converter.Simple("useRudderStorage", "use_rudder_storage"),
		converter.Discriminator("cloudProvider", converter.DiscriminatorValues{
			"s3":    "AWS",
			"gcp":   "GCP",
			"azure": "AZURE",
		}),
		converter.Simple("cleanupObjectStorageFiles", "cleanup_object_storage_files"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
		converter.Conditional("bucketName", "s3.bucket_name", converter.Equals("cloudProvider", "AWS")),
		converter.Simple("accessKeyID", "s3.access_key_id"),
		converter.Simple("accessKey", "s3.access_key"),
		converter.Simple("enableSSE", "s3.enable_sse"),
		converter.Simple("iamRoleARN", "s3.role_based_authentication.iam_role_arn"),
		converter.Discriminator("roleBasedAuth", converter.DiscriminatorValues{
			"s3.access_key":                false,
			"s3.access_key_id":             false,
			"s3.role_based_authentication": true,
		}),
		converter.Conditional("storageIntegration", "s3.storage_integration", converter.Equals("cloudProvider", "AWS")),
		converter.Conditional("bucketName", "gcp.bucket_name", converter.Equals("cloudProvider", "GCP")),
		converter.Simple("credentials", "gcp.credentials"),
		converter.Conditional("storageIntegration", "gcp.storage_integration", converter.Equals("cloudProvider", "GCP")),
		converter.Simple("containerName", "azure.container_name"),
		converter.Simple("accountName", "azure.account_name"),
		converter.Simple("accountKey", "azure.account_key"),
		converter.Simple("useSASTokens", "azure.use_sas_tokens"),
		converter.Simple("sasToken", "azure.sas_token"),
		converter.Conditional("storageIntegration", "azure.storage_integration", converter.Equals("cloudProvider", "AZURE")),
		converter.Simple("prefix", "prefix"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "snowflake",
		APIType:    "SNOWFLAKE",
		Version:    1,
		Properties: properties,
		// Only top-level string secrets; nested s3/gcp/azure secrets cannot be
		// modeled by the CLI secret machinery (see onboarding report).
		SecretKeys: []string{"password", "private_key", "private_key_passphrase"},
		NewConfig: func() any {
			return &snowflakeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
