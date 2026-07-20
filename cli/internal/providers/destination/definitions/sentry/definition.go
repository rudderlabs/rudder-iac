package sentry

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json allowUrls/denyUrls: (?!.*\.ngrok\.io).*$ — reject ngrok hosts.
	funcs.NewPatternWithReject(
		"sentry_url",
		`^.*$`,
		`\.ngrok\.io`,
		"must not contain .ngrok.io",
	)
}

// Source types from integrations-config destinations/sentry/db-config.json
// supportedSourceTypes (web only).
var sourceTypes = []string{
	common.SourceTypeWeb,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb: {"device"},
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,dive,max=100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,dive,max=100"`
}

type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

// sentryConfig is the local YAML config model. Field set mirrors terraform
// destination_sentry.go mappings; validation constraints mirror schema.json.
type sentryConfig struct {
	DSN                   string                   `mapstructure:"dsn" validate:"required,min=1,max=300"`
	Environment           string                   `mapstructure:"environment"`
	CustomVersionProperty string                   `mapstructure:"custom_version_property"`
	Release               string                   `mapstructure:"release"`
	ServerName            string                   `mapstructure:"server_name"`
	Logger                string                   `mapstructure:"logger"`
	DebugMode             *bool                    `mapstructure:"debug_mode"`
	IgnoreErrors          []string                 `mapstructure:"ignore_errors" validate:"omitempty,dive,max=100"`
	IncludePaths          []string                 `mapstructure:"include_paths" validate:"omitempty,dive,max=100"`
	AllowURLs             []string                 `mapstructure:"allow_urls" validate:"omitempty,dive,pattern=sentry_url"`
	DenyURLs              []string                 `mapstructure:"deny_urls" validate:"omitempty,dive,pattern=sentry_url"`
	EventFiltering        *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK          *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConsentManagement     common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Sentry destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("dsn", "dsn"),
		converter.Simple("environment", "environment", converter.SkipZeroValue),
		converter.Simple("customVersionProperty", "custom_version_property", converter.SkipZeroValue),
		converter.Simple("release", "release", converter.SkipZeroValue),
		converter.Simple("serverName", "server_name", converter.SkipZeroValue),
		converter.Simple("logger", "logger", converter.SkipZeroValue),
		converter.Simple("debugMode", "debug_mode"),
		converter.ArrayWithStrings("ignoreErrors", "ignoreErrors", "ignore_errors"),
		converter.ArrayWithStrings("includePaths", "includePaths", "include_paths"),
		converter.ArrayWithStrings("allowUrls", "allowUrls", "allow_urls"),
		converter.ArrayWithStrings("denyUrls", "denyUrls", "deny_urls"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		// Terraform omits Discriminator; CLI sets eventFilteringOption from whitelist/blacklist presence (GA4/PostHog pattern).
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "sentry",
		APIType:    "SENTRY",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &sentryConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
