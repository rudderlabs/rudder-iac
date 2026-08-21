package adj

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/adj/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeUnity,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeCloud,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud", "device"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud", "device"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud", "device"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

type adjustMapping struct {
	From string `mapstructure:"from" validate:"omitempty,max=100"`
	To   string `mapstructure:"to" validate:"omitempty,max=100"`
}

type enableInstallAttributionTracking struct {
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

// adjustConfig is the local YAML config model. Field set mirrors terraform
// destination_adjust mappings; validation constraints mirror schema.json.
type adjustConfig struct {
	AppToken                         string                            `mapstructure:"app_token" validate:"required,dynamic_or_pattern=single_line_100"`
	Delay                            string                            `mapstructure:"delay" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Environment                      *bool                             `mapstructure:"environment"`
	CustomMappings                   []adjustMapping                   `mapstructure:"custom_mappings" validate:"omitempty,dive"`
	PartnerParamKeys                 []adjustMapping                   `mapstructure:"partner_params_keys" validate:"omitempty,dive"`
	EnableInstallAttributionTracking *enableInstallAttributionTracking `mapstructure:"enable_install_attribution_tracking"`
	EventFilteringWhitelist          []string                          `mapstructure:"event_filtering_whitelist"`
	EventFilteringBlacklist          []string                          `mapstructure:"event_filtering_blacklist"`
	ConsentManagement                common.ConsentManagement          `mapstructure:"consent_management"`
}

// NewDefinition returns the Adjust destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("appToken", "app_token"),
		converter.Simple("delay", "delay"),
		converter.Simple("environment", "environment"),
		converter.ArrayWithObjects("customMappings", "custom_mappings", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("partnerParamsKeys", "partner_params_keys", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Gated(
			converter.Simple("enableInstallAttributionTracking.android", "enable_install_attribution_tracking.android"),
			common.SourceTypeAndroid, common.SourceTypeAndroidKotlin,
		),
		converter.Gated(
			converter.Simple("enableInstallAttributionTracking.ios", "enable_install_attribution_tracking.ios"),
			common.SourceTypeIOS, common.SourceTypeIOSSwift,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering_whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering_blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering_whitelist": "whitelistedEvents",
			"event_filtering_blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "adj",
		APIType:    "ADJ",
		Version:    1,
		Properties: properties,
		// db-config declares no secretKeys. Terraform marks app_token Sensitive, but
		// db-config is authoritative for write-only values: the API returns app_token,
		// so wrapping it as a secret would make the destination diff on every apply
		// and export as a "{{ .VAR }}" the user has to fill in by hand.
		SecretKeys: nil,
		NewConfig: func() any {
			return &adjustConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
