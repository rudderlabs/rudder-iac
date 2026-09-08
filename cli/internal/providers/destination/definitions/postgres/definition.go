package postgres

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
	// schema.json guards namespace with ^((?!pg_|PG_|pG_|Pg_).{0,64})$. RE2 has no
	// lookahead, so the reserved-prefix half becomes a reject pattern.
	funcs.NewPatternWithReject(
		"postgres_namespace",
		`^(.{0,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be at most 64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)

	// schema.json guards host with (?!.*.ngrok.io), whose dots are unescaped and
	// so match any character — broader than a literal ".ngrok.io", and broader
	// than the escaped form redis and slack use. The reject half reproduces that
	// exactly, including the unescaped dots, so the CLI rejects every host the API
	// would rather than deferring the failure to apply. It stays unanchored in
	// effect (leading and trailing .*) because a reject pattern must match
	// broadly: end-anchoring would let a trailing-dot FQDN or a host:port suffix
	// through.
	funcs.NewPatternWithReject(
		"postgres_host",
		`^(.{1,200})$`,
		`^.*.ngrok.io.*$`,
		"must be 1-200 characters, must not contain line breaks, and must not be an ngrok host",
	)

	// schema.json states a different bucketName rule per bucketProvider branch.
	// RE2 has no lookahead, so each stack of negative lookaheads becomes a reject
	// pattern. MINIO carries only the IP-address rule.
	funcs.NewPatternWithReject(
		"postgres_s3_bucket_name",
		`^[a-z0-9][a-z0-9-.]{1,61}[a-z0-9]$`,
		`^xn--|\.\.|^(\d+(\.|$)){4}$`,
		"must be a valid S3 bucket name: lowercase, no xn-- prefix, no consecutive dots, not an IP address",
	)
	funcs.NewPatternWithReject(
		"postgres_gcs_bucket_name",
		`^[a-z0-9][a-z0-9-._]{1,61}[a-z0-9]$`,
		`^goog|google|\.\.|^(\d+(\.|$)){4}$`,
		"must be a valid GCS bucket name: lowercase, no goog prefix, must not contain google, no consecutive dots, not an IP address",
	)
	funcs.NewPatternWithReject(
		"postgres_minio_bucket_name",
		`^[a-z0-9][a-z0-9-.]{1,61}[a-z0-9]$`,
		`^(\d+(\.|$)){4}$`,
		"must be a valid MinIO bucket name: lowercase and not an IP address",
	)

	// Azure container naming: ^(?=.{3,63}$)[a-z0-9]+(-[a-z0-9]+)*$ — the length
	// bound moves into the reject pattern, the character rule stays in accept.
	funcs.NewPatternWithReject(
		"postgres_container_name",
		`^[a-z0-9]+(-[a-z0-9]+)*$`,
		`^(.{0,2}|.{64,})$`,
		"must be 3-63 characters of lowercase letters, digits and single hyphens",
	)

	funcs.NewPatternWithReject(
		"postgres_end_point",
		`^(.{1,100})$`,
		`\.ngrok\.io`,
		"must be 1-100 characters, must not contain line breaks, and must not be an ngrok host",
	)
}

// bucketNameConditional switches bucket_name between the S3, GCS and MinIO rules
// that schema.json states per bucketProvider branch. All three are gated on
// useRudderStorage=false; outside them upstream sets no rule, so a stale bucket
// name from another provider round-trips rather than erroring.
func bucketNameConditional(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}

	parent := fl.Parent()
	if parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}

	useRudderStorageField := parent.FieldByName("UseRudderStorage")
	bucketProviderField := parent.FieldByName("BucketProvider")
	if !useRudderStorageField.IsValid() || !bucketProviderField.IsValid() {
		return true
	}

	useRudderStorage, _ := useRudderStorageField.Interface().(*bool)
	if useRudderStorage == nil || *useRudderStorage {
		return true
	}

	switch bucketProviderField.String() {
	case "S3":
		return funcs.MatchPattern("postgres_s3_bucket_name", value)
	case "GCS":
		return funcs.MatchPattern("postgres_gcs_bucket_name", value)
	case "MINIO":
		return funcs.MatchPattern("postgres_minio_bucket_name", value)
	default:
		return true
	}
}

// Source types from integrations-config destinations/postgres/db-config.json.
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

// excludeWindow mirrors the only genuinely nested object in the upstream config.
type excludeWindow struct {
	StartTime string `mapstructure:"start_time" validate:"required"`
	EndTime   string `mapstructure:"end_time" validate:"required"`
}

// postgresConfig is the local YAML config model. It is flat because the upstream
// config is flat; terraform's object-storage grouping is a provider artefact.
type postgresConfig struct {
	Host     string `mapstructure:"host" validate:"required,pattern=postgres_host"`
	Database string `mapstructure:"database" validate:"required,pattern=single_line_100"`
	User     string `mapstructure:"user" validate:"required,pattern=single_line_100"`
	Password string `mapstructure:"password" validate:"required"`
	Port     string `mapstructure:"port" validate:"required,pattern=single_line_100"`

	Namespace string `mapstructure:"namespace" validate:"omitempty,pattern=postgres_namespace"`
	UseSSH    *bool  `mapstructure:"use_ssh" default:"false"`
	SSHHost   string `mapstructure:"ssh_host" validate:"required_if=UseSSH true,omitempty,pattern=single_line_100"`
	SSHPort   string `mapstructure:"ssh_port" validate:"required_if=UseSSH true,omitempty,pattern=single_line_100"`
	SSHUser   string `mapstructure:"ssh_user" validate:"required_if=UseSSH true,omitempty,pattern=single_line_100"`
	// ssh_public_key is emitted by the backend and may be long; schema.json bounds
	// it to 1000 characters when SSH is enabled.
	SSHPublicKey string `mapstructure:"ssh_public_key" validate:"required_if=UseSSH true,omitempty,pattern=single_line_1000"`

	// schema.json requires the TLS material only for verify-ca, and declares no
	// pattern for any of the three, so they carry no shape constraint here.
	SSLMode       string         `mapstructure:"ssl_mode" validate:"required,oneof=disable require verify-ca"`
	ClientKey     string         `mapstructure:"client_key" validate:"required_if=SSLMode verify-ca"`
	ClientCert    string         `mapstructure:"client_cert" validate:"required_if=SSLMode verify-ca"`
	ServerCA      string         `mapstructure:"server_ca" validate:"required_if=SSLMode verify-ca"`
	SyncFrequency string         `mapstructure:"sync_frequency" validate:"required,oneof=5 10 15 30 60 180 360 720 1440"`
	SyncStartAt   string         `mapstructure:"sync_start_at"`
	ExcludeWindow *excludeWindow `mapstructure:"exclude_window"`

	SkipTracksTable         *bool  `mapstructure:"skip_tracks_table" default:"false"`
	SkipUsersTable          *bool  `mapstructure:"skip_users_table" default:"true"`
	PreferAppend            *bool  `mapstructure:"prefer_append" default:"true"`
	JSONPaths               string `mapstructure:"json_paths"`
	AllowUsersContextTraits *bool  `mapstructure:"allow_users_context_traits" default:"false"`
	UnderscoreDivideNumbers *bool  `mapstructure:"underscore_divide_numbers" default:"false"`

	// Object-storage staging. Upstream keeps every provider's keys in the same
	// flat object, so a key is required only for the providers schema.json names
	// it under — and only while rudder-managed storage is off, so a stale
	// bucketProvider left in config cannot resurrect the requirement.
	//
	// bucket_name is the one key required for more than one provider (S3, GCS and
	// MINIO but not AZURE_BLOB). required_if cannot express that, so it is stated
	// as its inverse with required_unless: exempt when storage is rudder-managed
	// or the provider is AZURE_BLOB, required otherwise.
	UseRudderStorage          *bool  `mapstructure:"use_rudder_storage" validate:"required"`
	BucketProvider            string `mapstructure:"bucket_provider" validate:"required_if=UseRudderStorage false,omitempty,oneof=S3 GCS AZURE_BLOB MINIO"`
	BucketName                string `mapstructure:"bucket_name" validate:"required_unless=UseRudderStorage true BucketProvider AZURE_BLOB,omitempty,pattern=single_line_100,postgres_bucket_name"`
	CleanupObjectStorageFiles *bool  `mapstructure:"cleanup_object_storage_files" default:"false"`

	// S3
	RoleBasedAuth *bool  `mapstructure:"role_based_auth"`
	IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"omitempty,pattern=single_line_100"`
	AccessKeyID   string `mapstructure:"access_key_id" validate:"required_if=UseRudderStorage false BucketProvider MINIO,omitempty,pattern=single_line_100"`
	AccessKey     string `mapstructure:"access_key" validate:"omitempty,pattern=single_line_100"`

	// Azure Blob
	AccountName   string `mapstructure:"account_name" validate:"required_if=UseRudderStorage false BucketProvider AZURE_BLOB,omitempty,pattern=single_line_100"`
	AccountKey    string `mapstructure:"account_key" validate:"omitempty,pattern=single_line_100"`
	SASToken      string `mapstructure:"sas_token"`
	UseSASTokens  *bool  `mapstructure:"use_sas_tokens"`
	ContainerName string `mapstructure:"container_name" validate:"required_if=UseRudderStorage false BucketProvider AZURE_BLOB,omitempty,pattern=postgres_container_name"`

	// GCS / MinIO
	Credentials     string `mapstructure:"credentials" validate:"required_if=UseRudderStorage false BucketProvider GCS"`
	EndPoint        string `mapstructure:"end_point" validate:"required_if=UseRudderStorage false BucketProvider MINIO,omitempty,pattern=postgres_end_point"`
	SecretAccessKey string `mapstructure:"secret_access_key" validate:"required_if=UseRudderStorage false BucketProvider MINIO,omitempty,pattern=single_line_100"`
	UseSSL          *bool  `mapstructure:"use_ssl" validate:"required_if=UseRudderStorage false BucketProvider MINIO"`

	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Postgres warehouse destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("host", "host"),
		converter.Simple("database", "database"),
		converter.Simple("user", "user"),
		converter.Simple("password", "password"),
		converter.Simple("port", "port"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("useSSH", "use_ssh"),
		converter.Simple("sshHost", "ssh_host"),
		converter.Simple("sshPort", "ssh_port"),
		converter.Simple("sshUser", "ssh_user"),
		converter.Simple("sshPublicKey", "ssh_public_key"),
		converter.Simple("sslMode", "ssl_mode"),
		converter.Simple("clientKey", "client_key"),
		converter.Simple("clientCert", "client_cert"),
		converter.Simple("serverCA", "server_ca"),
		converter.Simple("syncFrequency", "sync_frequency"),
		converter.Simple("syncStartAt", "sync_start_at"),
		converter.Simple("excludeWindow.excludeWindowStartTime", "exclude_window.start_time"),
		converter.Simple("excludeWindow.excludeWindowEndTime", "exclude_window.end_time"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("skipUsersTable", "skip_users_table"),
		converter.Simple("preferAppend", "prefer_append"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("useRudderStorage", "use_rudder_storage"),
		converter.Simple("bucketProvider", "bucket_provider"),
		converter.Simple("bucketName", "bucket_name"),
		converter.Simple("cleanupObjectStorageFiles", "cleanup_object_storage_files"),
		converter.Simple("roleBasedAuth", "role_based_auth"),
		converter.Simple("iamRoleARN", "iam_role_arn"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "access_key"),
		converter.Simple("accountName", "account_name"),
		converter.Simple("accountKey", "account_key"),
		converter.Simple("sasToken", "sas_token"),
		converter.Simple("useSASTokens", "use_sas_tokens"),
		converter.Simple("containerName", "container_name"),
		converter.Simple("credentials", "credentials"),
		converter.Simple("endPoint", "end_point"),
		converter.Simple("secretAccessKey", "secret_access_key"),
		converter.Simple("useSSL", "use_ssl"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "postgres",
		APIType:    "POSTGRES",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{
			"password",
			"access_key_id",
			"access_key",
			"account_key",
			"sas_token",
			"secret_access_key",
			"credentials",
		},
		NewConfig: func() any {
			return &postgresConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "postgres_bucket_name", Func: bucketNameConditional},
		},
	}
}
