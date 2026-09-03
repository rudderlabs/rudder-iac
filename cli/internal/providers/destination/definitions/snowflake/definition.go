package snowflake

import (
	"reflect"

	"github.com/go-playground/validator/v10"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func init() {
	// Unanchored, mirroring schema.json: upstream only requires that a PEM block is
	// present, so a key carrying the trailing newline every .pem file ends with —
	// or surrounding whitespace — stays valid. Anchoring would reject both.
	funcs.NewPattern(
		"snowflake_private_key",
		`(?s)-----BEGIN (ENCRYPTED )?PRIVATE KEY-----.+-----END (ENCRYPTED )?PRIVATE KEY-----`,
		"must be a PEM encoded private key",
	)

	// schema.json guards namespace with ^((?!pg_|PG_|pG_|Pg_).{0,64})$. RE2 has no
	// lookahead, so the reserved-prefix half becomes a reject pattern.
	funcs.NewPatternWithReject(
		"snowflake_namespace",
		`^(.{0,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be at most 64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)

	// schema.json states a different bucketName rule per provider branch. RE2 has
	// no lookahead, so each becomes a reject pattern.
	funcs.NewPatternWithReject(
		"snowflake_s3_bucket_name",
		`^[a-z0-9][a-z0-9-.]{1,61}[a-z0-9]$`,
		`^xn--|\.\.|^(\d+(\.|$)){4}$`,
		"must be a valid S3 bucket name: lowercase, no xn-- prefix, no consecutive dots, not an IP address",
	)
	funcs.NewPatternWithReject(
		"snowflake_gcs_bucket_name",
		`^[a-z0-9][a-z0-9-._]{1,61}[a-z0-9]$`,
		`^goog|google|\.\.|^(\d+(\.|$)){4}$`,
		"must be a valid GCS bucket name: lowercase, no goog prefix, must not contain google, no consecutive dots, not an IP address",
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

// bucketNameConditional switches bucket_name between the S3 and GCS rules that
// schema.json states per if/then branch. Outside those branches upstream sets no
// rule, so a stale bucket name from another provider round-trips rather than
// erroring.
func bucketNameConditional(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" || definitions.IsTemplateConfigValue(value) {
		return true
	}

	parent := fl.Parent()
	if parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}

	useRudderStorageField := parent.FieldByName("UseRudderStorage")
	cloudProviderField := parent.FieldByName("CloudProvider")
	if !useRudderStorageField.IsValid() || !cloudProviderField.IsValid() {
		return true
	}

	useRudderStorage, _ := useRudderStorageField.Interface().(*bool)
	if useRudderStorage == nil || *useRudderStorage {
		return true
	}

	switch cloudProviderField.String() {
	case "AWS":
		return funcs.MatchPattern("snowflake_s3_bucket_name", value)
	case "GCP":
		return funcs.MatchPattern("snowflake_gcs_bucket_name", value)
	default:
		return true
	}
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
	StartTime string `mapstructure:"start_time" validate:"required"`
	EndTime   string `mapstructure:"end_time" validate:"required"`
}

// snowflakeConfig is the local YAML config model. It is flat because the upstream
// config is flat: a real payload carries cloudProvider and roleBasedAuth as
// first-class keys and submits every provider's keys side by side (an AZURE
// destination still sends bucketName and iamRoleARN). Terraform groups them into
// s3/gcp/azure blocks, but that shape cannot represent such a payload — and the
// s3 definition, the verified reference, is flat for the same reason.
// Provider-scoped object-storage settings. The grouping mirrors terraform's
// s3/gcp/azure blocks, but only for keys upstream declares in exactly one
// provider branch. bucketName (AWS+GCP) and storageIntegration (all three) stay
// top level: a shared key cannot sit in one group without a Conditional to pick
// the branch, and that Conditional is what silently dropped keys the last time
// this shape was attempted.
//
// Grouping also lets the branch-scoped defaults land correctly. schema.json
// declares roleBasedAuth, enableSSE and useSASTokens inside their provider's
// if/then branch, so AJV applies them only when that branch is active; a nested
// default applies only inside a block the spec carries, which matches.
type s3Storage struct {
	RoleBasedAuth *bool  `mapstructure:"role_based_auth" validate:"snowflake_s3_required"`
	IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"snowflake_s3_role_required,omitempty,dynamic_or_pattern=single_line_100"`
	AccessKeyID   string `mapstructure:"access_key_id" validate:"snowflake_s3_key_required,omitempty,pattern=single_line_100"`
	AccessKey     string `mapstructure:"access_key" validate:"snowflake_s3_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	EnableSSE     *bool  `mapstructure:"enable_sse" default:"false"`
}

type gcpStorage struct {
	Credentials string `mapstructure:"credentials" validate:"snowflake_gcp_required"`
}

type azureStorage struct {
	ContainerName string `mapstructure:"container_name" validate:"snowflake_azure_required,omitempty,dynamic_or_pattern=azure_container_name"`
	AccountName   string `mapstructure:"account_name" validate:"snowflake_azure_required,omitempty,dynamic_or_pattern=single_line_100"`
	AccountKey    string `mapstructure:"account_key" validate:"snowflake_azure_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	UseSASTokens  *bool  `mapstructure:"use_sas_tokens" default:"false"`
	SASToken      string `mapstructure:"sas_token" validate:"snowflake_azure_sas_required"`
}

// storageBranchActive reports whether upstream's if/then branch for provider is
// in force. Every storage branch is gated on useRudderStorage=false plus the
// matching cloudProvider, and both live on the root config — required_if cannot
// see them from inside a nested struct (it silently passes instead of erroring),
// so these conditions are read off FieldLevel.Top().
func storageBranchActive(fl validator.FieldLevel, provider string) bool {
	top := fl.Top()
	for top.Kind() == reflect.Pointer {
		top = top.Elem()
	}

	useRudderStorageField := top.FieldByName("UseRudderStorage")
	cloudProviderField := top.FieldByName("CloudProvider")
	if !useRudderStorageField.IsValid() || !cloudProviderField.IsValid() {
		return false
	}

	useRudderStorage, _ := useRudderStorageField.Interface().(*bool)
	if useRudderStorage == nil || *useRudderStorage {
		return false
	}
	return cloudProviderField.String() == provider
}

// storageFieldIsSet reports whether the spec states a value. go-playground
// dereferences a non-nil pointer before calling the validator, so a *bool only
// arrives as a pointer when it is nil (via CallEvenIfNull): an explicit false
// reaches here as bool(false) and must count as stated, not as absent.
func storageFieldIsSet(fl validator.FieldLevel) bool {
	field := fl.Field()
	switch field.Kind() {
	case reflect.Pointer:
		return !field.IsNil()
	case reflect.Bool:
		return true
	default:
		return !field.IsZero()
	}
}

// requiredForProvider makes a field required whenever its provider branch is
// active, mirroring that branch's then.required list.
func requiredForProvider(provider string) validator.Func {
	return func(fl validator.FieldLevel) bool {
		if !storageBranchActive(fl, provider) {
			return true
		}
		return storageFieldIsSet(fl)
	}
}

// requiredForProviderWhen adds the sibling-flag condition upstream expresses as
// an anyOf inside the branch (roleBasedAuth picks IAM role vs access keys,
// useSASTokens picks SAS token vs account key).
func requiredForProviderWhen(provider, flagField string, want bool) validator.Func {
	return func(fl validator.FieldLevel) bool {
		if !storageBranchActive(fl, provider) {
			return true
		}

		parent := fl.Parent()
		for parent.Kind() == reflect.Pointer {
			parent = parent.Elem()
		}
		field := parent.FieldByName(flagField)
		if !field.IsValid() {
			return true
		}
		flag, _ := field.Interface().(*bool)
		if flag == nil || *flag != want {
			return true
		}
		return storageFieldIsSet(fl)
	}
}

type snowflakeConfig struct {
	Account   string `mapstructure:"account" validate:"required,dynamic_or_pattern=single_line_100"`
	Database  string `mapstructure:"database" validate:"required,dynamic_or_pattern=single_line_100"`
	Warehouse string `mapstructure:"warehouse" validate:"required,dynamic_or_pattern=single_line_100"`
	User      string `mapstructure:"user" validate:"required,dynamic_or_pattern=single_line_100"`
	Role      string `mapstructure:"role" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Namespace string `mapstructure:"namespace" validate:"omitempty,dynamic_or_pattern=snowflake_namespace"`

	// schema.json gates password/privateKey on useKeyPairAuth. password's literal
	// pattern accepts any string, while privateKey has no template branch and must
	// stay PEM-shaped after CLI variables have been resolved.
	UseKeyPairAuth       *bool  `mapstructure:"use_key_pair_auth" validate:"required"`
	Password             string `mapstructure:"password" validate:"required_if=UseKeyPairAuth false"`
	PrivateKey           string `mapstructure:"private_key" validate:"required_if=UseKeyPairAuth true,omitempty,pattern=snowflake_private_key"`
	PrivateKeyPassphrase string `mapstructure:"private_key_passphrase" validate:"omitempty,pattern=single_line_100"`

	SyncFrequency string         `mapstructure:"sync_frequency" validate:"required,oneof=5 10 15 30 60 180 360 720 1440"`
	SyncStartAt   string         `mapstructure:"sync_start_at" validate:"omitempty"`
	ExcludeWindow *excludeWindow `mapstructure:"exclude_window"`

	SkipTracksTable         *bool  `mapstructure:"skip_tracks_table" default:"false"`
	SkipUsersTable          *bool  `mapstructure:"skip_users_table" default:"true"`
	PreferAppend            *bool  `mapstructure:"prefer_append" default:"true"`
	ManualSync              *bool  `mapstructure:"manual_sync" default:"false"`
	JSONPaths               string `mapstructure:"json_paths" validate:"omitempty"`
	UnderscoreDivideNumbers *bool  `mapstructure:"underscore_divide_numbers" default:"false"`
	AllowUsersContextTraits *bool  `mapstructure:"allow_users_context_traits" default:"false"`

	// Object-storage staging. Upstream keeps every provider's keys in the same
	// flat object, so conditionals only validate the active provider; stale keys
	// from another provider can still round-trip without erasure.
	UseRudderStorage          *bool  `mapstructure:"use_rudder_storage" validate:"required"`
	CloudProvider             string `mapstructure:"cloud_provider" validate:"required_if=UseRudderStorage false,omitempty,oneof=AWS GCP AZURE" default:"AWS"`
	Prefix                    string `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	CleanupObjectStorageFiles *bool  `mapstructure:"cleanup_object_storage_files" default:"false"`
	StorageIntegration        string `mapstructure:"storage_integration" validate:"required_unless=UseRudderStorage true CloudProvider AWS,omitempty,dynamic_or_pattern=single_line_100"`

	// Shared across provider branches upstream, so they stay top level:
	// bucketName is declared by AWS and GCP, storageIntegration by all three.
	BucketName string `mapstructure:"bucket_name" validate:"required_unless=UseRudderStorage true CloudProvider AZURE,omitempty,dynamic_or_pattern=single_line_100,snowflake_bucket_name"`

	S3    s3Storage    `mapstructure:"s3"`
	GCP   gcpStorage   `mapstructure:"gcp"`
	Azure azureStorage `mapstructure:"azure"`

	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
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
		converter.Simple("roleBasedAuth", "s3.role_based_auth"),
		converter.Simple("iamRoleARN", "s3.iam_role_arn"),
		converter.Simple("accessKeyID", "s3.access_key_id"),
		converter.Simple("accessKey", "s3.access_key"),
		converter.Simple("enableSSE", "s3.enable_sse"),
		converter.Simple("credentials", "gcp.credentials"),
		converter.Simple("containerName", "azure.container_name"),
		converter.Simple("accountName", "azure.account_name"),
		converter.Simple("accountKey", "azure.account_key"),
		converter.Simple("useSASTokens", "azure.use_sas_tokens"),
		converter.Simple("sasToken", "azure.sas_token"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
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
			"s3.access_key_id",
			"s3.access_key",
			"azure.account_key",
			"azure.sas_token",
			"gcp.credentials",
		},
		NewConfig: func() any {
			return &snowflakeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "snowflake_bucket_name", Func: bucketNameConditional},
			{Tag: "snowflake_s3_required", Func: requiredForProvider("AWS"), CallEvenIfNull: true},
			{Tag: "snowflake_s3_role_required", Func: requiredForProviderWhen("AWS", "RoleBasedAuth", true)},
			{Tag: "snowflake_s3_key_required", Func: requiredForProviderWhen("AWS", "RoleBasedAuth", false)},
			{Tag: "snowflake_gcp_required", Func: requiredForProvider("GCP")},
			{Tag: "snowflake_azure_required", Func: requiredForProvider("AZURE")},
			{Tag: "snowflake_azure_key_required", Func: requiredForProviderWhen("AZURE", "UseSASTokens", false)},
			{Tag: "snowflake_azure_sas_required", Func: requiredForProviderWhen("AZURE", "UseSASTokens", true)},
		},
	}
}
