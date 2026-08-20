package snowpipestreaming

import (
	"strings"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func init() {
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

type snowpipeStreamingConfig struct {
	Account   string `mapstructure:"account" validate:"required,dynamic_or_pattern=single_line_100"`
	Database  string `mapstructure:"database" validate:"required,dynamic_or_pattern=single_line_100"`
	Warehouse string `mapstructure:"warehouse" validate:"required,dynamic_or_pattern=single_line_100"`
	User      string `mapstructure:"user" validate:"required,dynamic_or_pattern=single_line_100"`
	Role      string `mapstructure:"role" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Namespace string `mapstructure:"namespace" validate:"required,dynamic_or_pattern=snowpipe_streaming_namespace"`

	// Terraform accepts raw key bodies and wraps them for the API; mirror that
	// behavior instead of forcing every spec to carry PEM headers.
	PrivateKey           string `mapstructure:"private_key" validate:"required"`
	PrivateKeyPassphrase string `mapstructure:"private_key_passphrase" validate:"omitempty,dynamic_or_pattern=single_line_100"`

	SkipTracksTable         *bool  `mapstructure:"skip_tracks_table"`
	JSONPaths               string `mapstructure:"json_paths"`
	EnableIceberg           *bool  `mapstructure:"enable_iceberg"`
	ExternalVolume          string `mapstructure:"external_volume" validate:"required_if=EnableIceberg true,omitempty,dynamic_or_pattern=single_line_100"`
	UnderscoreDivideNumbers *bool  `mapstructure:"underscore_divide_numbers"`
	AllowUsersContextTraits *bool  `mapstructure:"allow_users_context_traits"`

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
		privateKeyProperty(),
		converter.Simple("privateKeyPassphrase", "private_key_passphrase"),
		converter.Simple("skipTracksTable", "skip_tracks_table"),
		converter.Simple("jsonPaths", "json_paths"),
		converter.Simple("enableIceberg", "enable_iceberg"),
		converter.Simple("externalVolume", "external_volume"),
		converter.Simple("underscoreDivideNumbers", "underscore_divide_numbers"),
		converter.Simple("allowUsersContextTraits", "allow_users_context_traits"),
	}
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

func privateKeyProperty() converter.ConfigProperty {
	return converter.ConfigProperty{
		LocalKey: "private_key",
		FromLocalFunc: func(config, local string) (string, error) {
			v := gjson.Get(local, "private_key")
			if !v.Exists() || v.Value() == nil || v.String() == "" {
				return config, nil
			}
			return sjson.Set(config, "privateKey", wrapPEMKey(v.String()))
		},
		ToLocalFunc: func(local, config string) (string, error) {
			r := gjson.Get(config, "privateKey")
			if !r.Exists() {
				return local, nil
			}
			return sjson.Set(local, "private_key", r.Value())
		},
	}
}

func wrapPEMKey(key string) string {
	if strings.HasPrefix(key, "-----BEGIN") {
		return key
	}
	return "-----BEGIN PRIVATE KEY-----\n" + key + "\n-----END PRIVATE KEY-----"
}
