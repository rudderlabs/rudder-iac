package snowpipestreaming

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json requires privateKey to carry PEM headers and declares no template
	// branch. Unanchored, mirroring upstream: it only requires that a PEM block is
	// present, so trailing newlines or surrounding whitespace stay valid.
	funcs.NewPattern(
		"snowpipe_streaming_private_key",
		`(?s)-----BEGIN (ENCRYPTED )?PRIVATE KEY-----.+-----END (ENCRYPTED )?PRIVATE KEY-----`,
		"must be a PEM encoded private key",
	)

	// schema.json guards namespace with ^((?!pg_|PG_|pG_|Pg_).{1,64})$.
	// RE2 has no lookahead, so the reserved-prefix half becomes a reject pattern.
	funcs.NewPatternWithReject(
		"snowpipe_streaming_namespace",
		`^(.{1,64})$`,
		`^(pg_|PG_|pG_|Pg_)`,
		"must be 1-64 characters, must not contain line breaks, and must not start with a pg_ prefix",
	)
}

// Source types from integrations-config destinations/snowpipe_streaming/db-config.json.
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

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff. See DEX-696 Discrepancy 3.
type snowpipeStreamingConfig struct {
	Account   string `mapstructure:"account" validate:"required,dynamic_or_pattern=single_line_100"`
	Database  string `mapstructure:"database" validate:"required,dynamic_or_pattern=single_line_100"`
	Warehouse string `mapstructure:"warehouse" validate:"required,dynamic_or_pattern=single_line_100"`
	User      string `mapstructure:"user" validate:"required,dynamic_or_pattern=single_line_100"`
	Role      string `mapstructure:"role" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Namespace string `mapstructure:"namespace" validate:"required,dynamic_or_pattern=snowpipe_streaming_namespace"`

	PrivateKey           string `mapstructure:"private_key" validate:"required,pattern=snowpipe_streaming_private_key"`
	PrivateKeyPassphrase string `mapstructure:"private_key_passphrase" validate:"omitempty,pattern=single_line_100"`

	SkipTracksTable         *bool  `mapstructure:"skip_tracks_table" default:"false"`
	JSONPaths               string `mapstructure:"json_paths"`
	EnableIceberg           *bool  `mapstructure:"enable_iceberg" default:"false"`
	ExternalVolume          string `mapstructure:"external_volume" validate:"required_if=EnableIceberg true,omitempty,pattern=single_line_100"`
	UnderscoreDivideNumbers *bool  `mapstructure:"underscore_divide_numbers" default:"false"`
	AllowUsersContextTraits *bool  `mapstructure:"allow_users_context_traits" default:"false"`

	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Snowpipe Streaming destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("account", "account"),
		converter.Simple("database", "database"),
		converter.Simple("warehouse", "warehouse"),
		converter.Simple("user", "user"),
		converter.Simple("role", "role"),
		converter.Simple("namespace", "namespace"),
		converter.Simple("privateKey", "private_key"),
		converter.Simple("privateKeyPassphrase", "private_key_passphrase"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("enableIceberg", "enable_iceberg"),
		converter.Simple("externalVolume", "external_volume"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "snowpipe_streaming",
		APIType:    "SNOWPIPE_STREAMING",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"private_key", "private_key_passphrase"},
		NewConfig: func() any {
			return &snowpipeStreamingConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
