package snowflake

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

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
	Frequency              string `mapstructure:"frequency" validate:"required,dynamic_or_oneof=5 15 30 60 180 360 720 1440"`
	StartAt                string `mapstructure:"start_at" validate:"omitempty"`
	ExcludeWindowStartTime string `mapstructure:"exclude_window_start_time" validate:"omitempty"`
	ExcludeWindowEndTime   string `mapstructure:"exclude_window_end_time" validate:"omitempty"`
}

type roleBasedAuthConfig struct {
	// Terraform local key is i_am_role_arn (not iam_role_arn).
	IAMRoleARN string `mapstructure:"i_am_role_arn" validate:"required,min=1,max=100"`
}

type s3StorageConfig struct {
	BucketName              string               `mapstructure:"bucket_name" validate:"required,min=1,max=100"`
	AccessKeyID             string               `mapstructure:"access_key_id" validate:"required_with=AccessKey,excluded_with=RoleBasedAuthentication,omitempty,max=100"`
	AccessKey               string               `mapstructure:"access_key" validate:"required_with=AccessKeyID,excluded_with=RoleBasedAuthentication,omitempty,max=100"`
	EnableSSE               *bool                `mapstructure:"enable_sse"`
	RoleBasedAuthentication *roleBasedAuthConfig `mapstructure:"role_based_authentication" validate:"excluded_with=AccessKeyID"`
	StorageIntegration      string               `mapstructure:"storage_integration" validate:"omitempty,max=100"`
}

type gcpStorageConfig struct {
	BucketName         string `mapstructure:"bucket_name" validate:"required,min=1,max=100"`
	Credentials        string `mapstructure:"credentials" validate:"required,min=1"`
	StorageIntegration string `mapstructure:"storage_integration" validate:"required,min=1,max=100"`
}

type azureStorageConfig struct {
	ContainerName      string `mapstructure:"container_name" validate:"required,min=1,max=100"`
	AccountName        string `mapstructure:"account_name" validate:"required,min=1,max=100"`
	AccountKey         string `mapstructure:"account_key" validate:"required,min=1,max=100"`
	StorageIntegration string `mapstructure:"storage_integration" validate:"required,min=1,max=100"`
}

// snowflakeConfig is the local YAML config model. Nested s3/gcp/azure/sync
// blocks mirror terraform-provider-rudderstack destination_snowflake.go shape
// (without TF list indices). Validation constraints come from schema.json.
type snowflakeConfig struct {
	Account              string                   `mapstructure:"account" validate:"required,min=1,max=100"`
	Database             string                   `mapstructure:"database" validate:"required,min=1,max=100"`
	Warehouse            string                   `mapstructure:"warehouse" validate:"required,min=1,max=100"`
	User                 string                   `mapstructure:"user" validate:"required,min=1,max=100"`
	UseKeyPairAuth       *bool                    `mapstructure:"use_key_pair_auth" validate:"required"`
	Password             string                   `mapstructure:"password" validate:"required_if=UseKeyPairAuth false"`
	PrivateKey           string                   `mapstructure:"private_key" validate:"required_if=UseKeyPairAuth true"`
	PrivateKeyPassphrase string                   `mapstructure:"private_key_passphrase" validate:"omitempty,max=100"`
	Role                 string                   `mapstructure:"role" validate:"omitempty,max=100"`
	Namespace            string                   `mapstructure:"namespace" validate:"omitempty,max=64"`
	Sync                 syncConfig               `mapstructure:"sync"`
	SkipTracksTable      *bool                    `mapstructure:"skip_tracks_table"`
	SkipUsersTable       *bool                    `mapstructure:"skip_users_table"`
	PreferAppend         *bool                    `mapstructure:"prefer_append"`
	JSONPaths            string                   `mapstructure:"json_paths" validate:"omitempty"`
	ManualSync           *bool                    `mapstructure:"manual_sync"`
	UseRudderStorage     *bool                    `mapstructure:"use_rudder_storage" validate:"required"`
	AdditionalProperties *bool                    `mapstructure:"additional_properties"`
	Prefix               string                   `mapstructure:"prefix" validate:"omitempty,max=100"`
	S3                   *s3StorageConfig         `mapstructure:"s3"`
	GCP                  *gcpStorageConfig        `mapstructure:"gcp"`
	Azure                *azureStorageConfig      `mapstructure:"azure"`
	ConsentManagement    common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Snowflake warehouse destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("account", "account"),
		converter.Simple("database", "database"),
		converter.Simple("warehouse", "warehouse"),
		converter.Simple("user", "user"),
		converter.Simple("useKeyPairAuth", "use_key_pair_auth"),
		converter.Simple("password", "password", converter.SkipZeroValue),
		converter.Simple("privateKey", "private_key", converter.SkipZeroValue),
		converter.Simple("privateKeyPassphrase", "private_key_passphrase", converter.SkipZeroValue),
		converter.Simple("role", "role", converter.SkipZeroValue),
		converter.Simple("namespace", "namespace", converter.SkipZeroValue),
		converter.Simple("syncFrequency", "sync.frequency"),
		converter.Simple("syncStartAt", "sync.start_at", converter.SkipZeroValue),
		converter.Simple("excludeWindow.excludeWindowStartTime", "sync.exclude_window_start_time", converter.SkipZeroValue),
		converter.Simple("excludeWindow.excludeWindowEndTime", "sync.exclude_window_end_time", converter.SkipZeroValue),
		converter.Simple("skipTracksTable", "skip_tracks_table", converter.SkipZeroValue),
		converter.Simple("skipUsersTable", "skip_users_table", converter.SkipZeroValue),
		converter.Simple("preferAppend", "prefer_append", converter.SkipZeroValue),
		converter.Simple("jsonPaths", "json_paths", converter.SkipZeroValue),
		converter.Simple("manualSync", "manual_sync", converter.SkipZeroValue),
		converter.Simple("useRudderStorage", "use_rudder_storage"),
		converter.Discriminator("cloudProvider", converter.DiscriminatorValues{
			"s3":    "AWS",
			"gcp":   "GCP",
			"azure": "AZURE",
		}),
		converter.Simple("additionalProperties", "additional_properties", converter.SkipZeroValue),
		converter.Conditional("bucketName", "s3.bucket_name", converter.Equals("cloudProvider", "AWS")),
		converter.Simple("accessKeyID", "s3.access_key_id", converter.SkipZeroValue),
		converter.Simple("accessKey", "s3.access_key", converter.SkipZeroValue),
		converter.Simple("enableSSE", "s3.enable_sse", converter.SkipZeroValue),
		converter.Simple("iamRoleARN", "s3.role_based_authentication.i_am_role_arn", converter.SkipZeroValue),
		converter.Discriminator("roleBasedAuth", converter.DiscriminatorValues{
			"s3.access_key":                false,
			"s3.access_key_id":             false,
			"s3.role_based_authentication": true,
		}),
		converter.Conditional("storageIntegration", "s3.storage_integration", converter.Equals("cloudProvider", "AWS")),
		converter.Conditional("bucketName", "gcp.bucket_name", converter.Equals("cloudProvider", "GCP")),
		converter.Simple("credentials", "gcp.credentials", converter.SkipZeroValue),
		converter.Conditional("storageIntegration", "gcp.storage_integration", converter.Equals("cloudProvider", "GCP")),
		converter.Simple("containerName", "azure.container_name", converter.SkipZeroValue),
		converter.Simple("accountName", "azure.account_name", converter.SkipZeroValue),
		converter.Simple("accountKey", "azure.account_key", converter.SkipZeroValue),
		converter.Conditional("storageIntegration", "azure.storage_integration", converter.Equals("cloudProvider", "AZURE")),
		converter.Simple("prefix", "prefix", converter.SkipZeroValue),
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
