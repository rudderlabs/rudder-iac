package activecampaign

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	activeCampaignAPIURLPattern       = `^(?:http(s)?://)?[\w.-]+(?:\.[\w\.-]+)+[\w\-._~:/?#[\]@!\$&'\(\)\*\+,;=.]+$`
	activeCampaignAPIURLRejectPattern = `^.*\.ngrok\.io.*$`
)

func init() {
	funcs.NewPatternWithReject(
		"active_campaign_api_url",
		activeCampaignAPIURLPattern,
		activeCampaignAPIURLRejectPattern,
		"must be a valid URL and must not use ngrok",
	)
}

// Source types from integrations-config destinations/active_campaign/db-config.json.
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
	common.SourceTypeWeb:           {"cloud", "device", "hybrid"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

// useNativeSDK is web-only upstream: schema.json declares just the web key, and
// web is the one source type offering device and hybrid modes.
type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

// activeCampaignConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/active_campaign defaultConfig; validation
// constraints mirror schema.json after stripping UI template/env branches.
type activeCampaignConfig struct {
	APIURL            string                   `mapstructure:"api_url" validate:"required,dynamic_or_pattern=active_campaign_api_url"`
	APIKey            string                   `mapstructure:"api_key" validate:"required,dynamic_or_pattern=single_line_100"`
	ActID             string                   `mapstructure:"actid" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	EventKey          string                   `mapstructure:"event_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	UseNativeSDK      *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the ActiveCampaign destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiUrl", "api_url"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("actid", "actid"),
		converter.Simple("eventKey", "event_key"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "active_campaign",
		APIType:    "ACTIVE_CAMPAIGN",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_key", "event_key"},
		NewConfig: func() any {
			return &activeCampaignConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
