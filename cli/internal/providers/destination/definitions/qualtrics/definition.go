package qualtrics

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/qualtrics/db-config.json.
var sourceTypes = []string{
	common.SourceTypeWeb,
	common.SourceTypeAndroid,
	common.SourceTypeIOS,
}

var connectionModes = map[string][]string{
	common.SourceTypeWeb:     {"device"},
	common.SourceTypeAndroid: {"device"},
	common.SourceTypeIOS:     {"device"},
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDK struct {
	Web     *bool `mapstructure:"web"`
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

type enableGenericPageTitle struct {
	Web *bool `mapstructure:"web"`
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff.
//
// connection_mode is deliberately absent too. db-config lists connectionMode
// under every supported source type, but schema.json declares no such property,
// and schema.json is the authority on the config surface. ConnectionModes below
// still advertises the supported modes as metadata.
//
// qualtricsConfig is the local YAML config model. Field set mirrors schema.json
// and db-config.json destConfig; validation constraints mirror schema.json.
type qualtricsConfig struct {
	ProjectID              string                   `mapstructure:"project_id" validate:"required"`
	BrandID                string                   `mapstructure:"brand_id" validate:"required"`
	EnableGenericPageTitle *enableGenericPageTitle  `mapstructure:"enable_generic_page_title"`
	UseNativeSDK           *useNativeSDK            `mapstructure:"use_native_sdk"`
	EventFiltering         *eventFiltering          `mapstructure:"event_filtering"`
	ConsentManagement      common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Qualtrics destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("projectId", "project_id"),
		converter.Simple("brandId", "brand_id"),
		converter.Gated(
			converter.Simple("enableGenericPageTitle.web", "enable_generic_page_title.web"),
			common.SourceTypeWeb,
		),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "qualtrics",
		APIType:    "QUALTRICS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{},
		NewConfig: func() any {
			return &qualtricsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
