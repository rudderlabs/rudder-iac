package vwo

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/vwo/db-config.json
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

// vwoConfig is the local YAML config model. Field set mirrors the
// terraform-provider VWO mapping contract plus shared source-scoped settings.
type vwoConfig struct {
	AccountID              string                   `mapstructure:"account_id" validate:"required,dynamic_or_pattern=single_line_100"`
	IsSPA                  *bool                    `mapstructure:"is_spa" default:"false"`
	SendExperimentTrack    *bool                    `mapstructure:"send_experiment_track" default:"false"`
	SendExperimentIdentify *bool                    `mapstructure:"send_experiment_identify" default:"false"`
	LibraryTolerance       string                   `mapstructure:"library_tolerance" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	SettingsTolerance      string                   `mapstructure:"settings_tolerance" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	UseExistingJquery      *bool                    `mapstructure:"use_existing_jquery" default:"false"`
	EventFiltering         *eventFilteringConfig    `mapstructure:"event_filtering"`
	UseNativeSDK           *useNativeSDKConfig      `mapstructure:"use_native_sdk"`
	ConnectionMode         common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement      common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the VWO destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("accountId", "account_id"),
		converter.Simple("isSPA", "is_spa"),
		converter.Simple("sendExperimentTrack", "send_experiment_track"),
		converter.Simple("sendExperimentIdentify", "send_experiment_identify"),
		converter.Simple("libraryTolerance", "library_tolerance"),
		converter.Simple("settingsTolerance", "settings_tolerance"),
		converter.Simple("useExistingJquery", "use_existing_jquery"),
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
		Type:       "vwo",
		APIType:    "VWO",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &vwoConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
