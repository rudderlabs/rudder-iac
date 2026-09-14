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

// accessKeyIDRequired covers the two branches that require access_key_id: MinIO
// always, and S3 unless role-based auth is on. schema.json declares the key in
// both branches, so it stays top-level — nesting one key under two providers
// would need the API's single flat accessKeyID routed by bucket_provider, and a
// stale value from a third provider would have nowhere to land.
func accessKeyIDRequired(fl validator.FieldLevel) bool {
	if fl.Field().String() != "" {
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
	case "MINIO":
		return false
	case "S3":
		s3Field := parent.FieldByName("S3")
		if !s3Field.IsValid() {
			return true
		}
		roleBasedField := s3Field.FieldByName("RoleBasedAuth")
		if !roleBasedField.IsValid() {
			return true
		}
		roleBased, _ := roleBasedField.Interface().(*bool)
		return roleBased != nil && *roleBased
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

// Provider-scoped object-storage settings. The local YAML groups keys by the
// storage provider that owns them, while converter mappings keep the upstream
// POSTGRES API payload flat. Shared selector/staging keys stay top-level because
// the API uses them across branches and update replaces the whole config object.
type s3Storage struct {
	RoleBasedAuth *bool  `mapstructure:"role_based_auth"`
	IAMRoleARN    string `mapstructure:"iam_role_arn" validate:"postgres_s3_role_required,omitempty,pattern=single_line_100"`
	AccessKey     string `mapstructure:"access_key" validate:"postgres_s3_key_required,omitempty,pattern=single_line_100"`
}

// SSH tunnel settings, grouped because schema.json declares all four only
// inside the useSSH branch. use_ssh stays top level as the selector.
type sshConfig struct {
	Host string `mapstructure:"host" validate:"postgres_ssh_required,omitempty,pattern=single_line_100"`
	Port string `mapstructure:"port" validate:"postgres_ssh_required,omitempty,pattern=single_line_100"`
	User string `mapstructure:"user" validate:"postgres_ssh_required,omitempty,pattern=single_line_100"`
	// public_key is emitted by the backend and may be long; schema.json bounds it
	// to 1000 characters when SSH is enabled.
	PublicKey string `mapstructure:"public_key" validate:"postgres_ssh_required,omitempty,pattern=single_line_1000"`
}

// postgresSSHRequired reads the selector from the top-level config because the
// SSH fields live in a nested block: fl.Parent() is sshConfig, which does not
// carry use_ssh, so required_if would silently never fire.
func postgresSSHRequired(fl validator.FieldLevel) bool {
	if fl.Field().String() != "" {
		return true
	}

	root := fl.Top()
	for root.Kind() == reflect.Pointer {
		root = root.Elem()
	}
	if root.Kind() != reflect.Struct {
		return true
	}

	field := root.FieldByName("UseSSH")
	if !field.IsValid() {
		return true
	}

	useSSH, _ := field.Interface().(*bool)
	return useSSH == nil || !*useSSH
}

type gcsStorage struct {
	Credentials string `mapstructure:"credentials" validate:"postgres_gcs_required"`
}

type azureStorage struct {
	AccountName   string `mapstructure:"account_name" validate:"postgres_azure_required,omitempty,pattern=single_line_100"`
	AccountKey    string `mapstructure:"account_key" validate:"postgres_azure_key_required,omitempty,pattern=single_line_100"`
	SASToken      string `mapstructure:"sas_token" validate:"postgres_azure_sas_required"`
	UseSASTokens  *bool  `mapstructure:"use_sas_tokens"`
	ContainerName string `mapstructure:"container_name" validate:"postgres_azure_required,omitempty,pattern=postgres_container_name"`
}

type minioStorage struct {
	EndPoint        string `mapstructure:"end_point" validate:"postgres_minio_required,omitempty,pattern=postgres_end_point"`
	SecretAccessKey string `mapstructure:"secret_access_key" validate:"postgres_minio_required,omitempty,pattern=single_line_100"`
	UseSSL          *bool  `mapstructure:"use_ssl" validate:"postgres_minio_required"`
}

// storageBranchActive reports whether the given bucketProvider branch is in
// force. Every storage branch is gated on useRudderStorage=false plus the
// provider selector, so outside that pair no branch rule applies.
func storageBranchActive(fl validator.FieldLevel, provider string) bool {
	top := fl.Top()
	for top.Kind() == reflect.Pointer {
		top = top.Elem()
	}

	useRudderStorageField := top.FieldByName("UseRudderStorage")
	bucketProviderField := top.FieldByName("BucketProvider")
	if !useRudderStorageField.IsValid() || !bucketProviderField.IsValid() {
		return false
	}

	useRudderStorage, _ := useRudderStorageField.Interface().(*bool)
	if useRudderStorage == nil || *useRudderStorage {
		return false
	}
	return bucketProviderField.String() == provider
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
// useSASTokens picks SAS token vs account key). schema.json defaults neither
// flag, so an omitted one reads as false and selects the access-key side.
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
		selected := flag != nil && *flag
		if selected != want {
			return true
		}
		return storageFieldIsSet(fl)
	}
}

type postgresConfig struct {
	Host     string `mapstructure:"host" validate:"required,pattern=postgres_host"`
	Database string `mapstructure:"database" validate:"required,pattern=single_line_100"`
	User     string `mapstructure:"user" validate:"required,pattern=single_line_100"`
	Password string `mapstructure:"password" validate:"required"`
	Port     string `mapstructure:"port" validate:"required,pattern=single_line_100"`

	Namespace string    `mapstructure:"namespace" validate:"omitempty,pattern=postgres_namespace"`
	UseSSH    *bool     `mapstructure:"use_ssh" default:"false"`
	SSH       sshConfig `mapstructure:"ssh"`

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

	UseRudderStorage          *bool  `mapstructure:"use_rudder_storage" validate:"required"`
	BucketProvider            string `mapstructure:"bucket_provider" validate:"required_if=UseRudderStorage false,omitempty,oneof=S3 GCS AZURE_BLOB MINIO"`
	BucketName                string `mapstructure:"bucket_name" validate:"required_unless=UseRudderStorage true BucketProvider AZURE_BLOB,omitempty,pattern=single_line_100,postgres_bucket_name"`
	AccessKeyID               string `mapstructure:"access_key_id" validate:"postgres_access_key_id_required,omitempty,pattern=single_line_100"`
	CleanupObjectStorageFiles *bool  `mapstructure:"cleanup_object_storage_files" default:"false"`

	S3    s3Storage    `mapstructure:"s3"`
	GCS   gcsStorage   `mapstructure:"gcs"`
	Azure azureStorage `mapstructure:"azure"`
	MinIO minioStorage `mapstructure:"minio"`

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
		converter.Simple("sshHost", "ssh.host"),
		converter.Simple("sshPort", "ssh.port"),
		converter.Simple("sshUser", "ssh.user"),
		converter.Simple("sshPublicKey", "ssh.public_key"),
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
		converter.Simple("roleBasedAuth", "s3.role_based_auth"),
		converter.Simple("iamRoleARN", "s3.iam_role_arn"),
		converter.Simple("accessKeyID", "access_key_id"),
		converter.Simple("accessKey", "s3.access_key"),
		converter.Simple("accountName", "azure.account_name"),
		converter.Simple("accountKey", "azure.account_key"),
		converter.Simple("sasToken", "azure.sas_token"),
		converter.Simple("useSASTokens", "azure.use_sas_tokens"),
		converter.Simple("containerName", "azure.container_name"),
		converter.Simple("credentials", "gcs.credentials"),
		converter.Simple("endPoint", "minio.end_point"),
		converter.Simple("secretAccessKey", "minio.secret_access_key"),
		converter.Simple("useSSL", "minio.use_ssl"),
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
			"s3.access_key",
			"azure.account_key",
			"azure.sas_token",
			"gcs.credentials",
			"minio.secret_access_key",
		},
		NewConfig: func() any {
			return &postgresConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "postgres_bucket_name", Func: bucketNameConditional},
			{Tag: "postgres_ssh_required", Func: postgresSSHRequired},
			{Tag: "postgres_access_key_id_required", Func: accessKeyIDRequired},
			{Tag: "postgres_s3_role_required", Func: requiredForProviderWhen("S3", "RoleBasedAuth", true)},
			{Tag: "postgres_s3_key_required", Func: requiredForProviderWhen("S3", "RoleBasedAuth", false)},
			{Tag: "postgres_gcs_required", Func: requiredForProvider("GCS")},
			{Tag: "postgres_azure_required", Func: requiredForProvider("AZURE_BLOB")},
			{Tag: "postgres_azure_key_required", Func: requiredForProviderWhen("AZURE_BLOB", "UseSASTokens", false)},
			{Tag: "postgres_azure_sas_required", Func: requiredForProviderWhen("AZURE_BLOB", "UseSASTokens", true)},
			{Tag: "postgres_minio_required", Func: requiredForProvider("MINIO"), CallEvenIfNull: true},
		},
	}
}
