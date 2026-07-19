package googletagmanager

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/gtm/db-config.json
// supportedSourceTypes (web only).
var sourceTypes = []string{
	common.SourceTypeWeb,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb: {"device"},
}

type eventFilteringConfig struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,dive,max=100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,dive,max=100"`
}

type useNativeSDKConfig struct {
	Web *bool `mapstructure:"web"`
}

// googleTagManagerConfig is the local YAML config model. Field set mirrors
// terraform-provider destination_google_tag_manager.go; validation constraints
// mirror overlapping schema.json rules for those mapped fields.
type googleTagManagerConfig struct {
	ContainerID       string                   `mapstructure:"container_id" validate:"required,min=1,max=100"`
	ServerURL         string                   `mapstructure:"server_url" validate:"omitempty,pattern=url"`
	EventFiltering    *eventFilteringConfig    `mapstructure:"event_filtering"`
	UseNativeSDK      *useNativeSDKConfig      `mapstructure:"use_native_sdk"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Tag Manager (API type GTM) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("containerID", "container_id"),
		converter.Simple("serverUrl", "server_url", converter.SkipZeroValue),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "google_tag_manager",
		APIType:    "GTM",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &googleTagManagerConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
