package gtm

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// serverUrl carries the same URL idiom ga4 uses for sdkBaseUrl. RE2 has no
	// lookahead, so the ngrok guard becomes a reject pattern; the empty
	// alternative is preserved because upstream allows an unset value.
	funcs.NewPatternWithReject(
		"gtm_server_url",
		`^(?:https?://)?[\w.-]+(?:\.[\w.-]+)+[\w\-._~:/?#[\]@!$&'()*+,;=.]*|^$`,
		`\.ngrok\.io`,
		"must be a domain URL and must not use ngrok",
	)
}

// Source types from integrations-config destinations/gtm/db-config.json
// supportedSourceTypes (web only).
var sourceTypes = []string{
	common.SourceTypeWeb,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb: {"device"},
}

type eventFilteringConfig struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDKConfig struct {
	Web *bool `mapstructure:"web"`
}

// gtmConfig is the local YAML config model. Field set mirrors the keys upstream
// declares in db-config.json destConfig; validation constraints mirror
// schema.json.
type gtmConfig struct {
	ContainerID string `mapstructure:"container_id" validate:"required,dynamic_or_pattern=single_line_100"`
	ServerURL   string `mapstructure:"server_url" validate:"omitempty,dynamic_or_pattern=gtm_server_url"`
	// environmentID and authorizationToken carry no {{ }} branch upstream, so
	// they take the plain pattern rather than the template-accepting variant.
	EnvironmentID      string                   `mapstructure:"environment_id" validate:"omitempty,pattern=single_line_100"`
	AuthorizationToken string                   `mapstructure:"authorization_token" validate:"omitempty,pattern=single_line_100"`
	EventFiltering     *eventFilteringConfig    `mapstructure:"event_filtering"`
	UseNativeSDK       *useNativeSDKConfig      `mapstructure:"use_native_sdk"`
	ConnectionMode     common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement  common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Tag Manager destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("containerID", "container_id"),
		converter.Simple("serverUrl", "server_url"),
		// environmentID and authorizationToken are in db-config defaultConfig but
		// missing from terraform; without them an apply erases whatever the UI set.
		converter.Simple("environmentID", "environment_id"),
		converter.Simple("authorizationToken", "authorization_token"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "gtm",
		APIType:    "GTM",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &gtmConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
