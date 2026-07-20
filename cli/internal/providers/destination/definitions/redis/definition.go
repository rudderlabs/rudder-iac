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
		`^.{0,100}$`,
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

// redisConfig is the local YAML config model. Field set mirrors terraform
// destination_redis mappings; validation constraints mirror schema.json.
type redisConfig struct {
	Address           string                   `mapstructure:"address" validate:"required,pattern=redis_address"`
	Password          string                   `mapstructure:"password" validate:"omitempty"`
	ClusterMode       *bool                    `mapstructure:"cluster_mode"`
	Secure            *bool                    `mapstructure:"secure"`
	Prefix            string                   `mapstructure:"prefix" validate:"omitempty,max=100"`
	Database          string                   `mapstructure:"database" validate:"omitempty,max=100"`
	CACertificate     string                   `mapstructure:"ca_certificate" validate:"omitempty"`
	SkipVerify        *bool                    `mapstructure:"skip_verify"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Redis destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("address", "address"),
		converter.Simple("password", "password", converter.SkipZeroValue),
		converter.Simple("clusterMode", "cluster_mode"),
		converter.Simple("secure", "secure"),
		converter.Simple("prefix", "prefix", converter.SkipZeroValue),
		converter.Simple("database", "database", converter.SkipZeroValue),
		converter.Simple("caCertificate", "ca_certificate", converter.SkipZeroValue),
		converter.Simple("skipVerify", "skip_verify"),
	}
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
