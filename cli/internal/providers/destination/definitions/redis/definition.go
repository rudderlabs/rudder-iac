package redis

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// Upstream schema rejects ngrok tunnels via a negative lookahead, which RE2
	// cannot express — use NewPatternWithReject instead.
	funcs.NewPatternWithReject(
		"redis_address",
		`^(.{0,100})$`,
		`\.ngrok\.io`,
		"must be at most 100 characters and must not contain .ngrok.io",
	)
}

// Source types from integrations-config destinations/redis/db-config.json
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

// redisConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/redis defaultConfig; validation constraints
// mirror schema.json. password and ca_certificate carry no pattern because
// schema.json constrains them to `.*`.
type redisConfig struct {
	Address       string `mapstructure:"address" validate:"required,dynamic_or_pattern=redis_address"`
	Password      string `mapstructure:"password" validate:"omitempty"`
	ClusterMode   *bool  `mapstructure:"cluster_mode"`
	Secure        *bool  `mapstructure:"secure"`
	Prefix        string `mapstructure:"prefix" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Database      string `mapstructure:"database" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	CACertificate string `mapstructure:"ca_certificate" validate:"omitempty"`
	SkipVerify    *bool  `mapstructure:"skip_verify"`
	// schema.json and defaultConfig declare useJSONModule but terraform does not
	// map it. Modelled anyway: an unmodelled key is dropped from the update
	// payload and erased upstream on the first apply.
	UseJSONModule     *bool                    `mapstructure:"use_json_module"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Redis destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("address", "address"),
		converter.Simple("password", "password"),
		converter.Simple("clusterMode", "cluster_mode"),
		converter.Simple("secure", "secure"),
		converter.Simple("prefix", "prefix"),
		converter.Simple("database", "database"),
		converter.Simple("caCertificate", "ca_certificate"),
		converter.Simple("skipVerify", "skip_verify"),
		converter.Simple("useJSONModule", "use_json_module"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "redis",
		APIType:    "REDIS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"password", "ca_certificate"},
		NewConfig: func() any {
			return &redisConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
