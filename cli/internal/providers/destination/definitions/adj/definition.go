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

// schema.json guards both sides with ^(.{0,100})$ — a pattern forbidding line
// breaks, not a length cap — plus a template branch.
type adjustMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type enableInstallAttributionTracking struct {
	Android       *bool `mapstructure:"android"`
	AndroidKotlin *bool `mapstructure:"android_kotlin"`
	IOS           *bool `mapstructure:"ios"`
	IOSSwift      *bool `mapstructure:"ios_swift"`
}

// Sub-key set mirrors schema.json; db-config lists the same six source types
// under destConfig. Not gated, following the source-type block convention
// use_native_sdk shares with consent_management and connection_mode.
type useNativeSDK struct {
	Android       *bool `mapstructure:"android"`
	AndroidKotlin *bool `mapstructure:"android_kotlin"`
	IOS           *bool `mapstructure:"ios"`
	IOSSwift      *bool `mapstructure:"ios_swift"`
	Unity         *bool `mapstructure:"unity"`
	Flutter       *bool `mapstructure:"flutter"`
}

// Nested event_filtering block, matching the fleet convention (braze,
// facebook_pixel, ...). schema.json patterns each eventName the same way as
// the mapping fields.
type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

// adjustConfig is the local YAML config model. Field set mirrors terraform
// destination_adjust mappings; validation constraints mirror schema.json.
// app_token and delay use plain pattern= rather than dynamic_or_pattern: unlike
// the nested mapping and event-filter fields, their schema.json patterns carry no
// {{ path || fallback }} branch, so upstream rejects templates there too.
type adjustConfig struct {
	AppToken                         string                            `mapstructure:"app_token" validate:"required,pattern=single_line_100"`
	Delay                            string                            `mapstructure:"delay" validate:"omitempty,pattern=single_line_100"`
	Environment                      *bool                             `mapstructure:"environment" default:"false"`
	CustomMappings                   []adjustMapping                   `mapstructure:"custom_mappings" validate:"omitempty,dive"`
	PartnerParamKeys                 []adjustMapping                   `mapstructure:"partner_params_keys" validate:"omitempty,dive"`
	EnableInstallAttributionTracking *enableInstallAttributionTracking `mapstructure:"enable_install_attribution_tracking"`
	UseNativeSDK                     *useNativeSDK                     `mapstructure:"use_native_sdk"`
	EventFiltering                   *eventFiltering                   `mapstructure:"event_filtering"`
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
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("enableInstallAttributionTracking.androidKotlin", "enable_install_attribution_tracking.android_kotlin"),
			common.SourceTypeAndroidKotlin,
		),
		converter.Gated(
			converter.Simple("enableInstallAttributionTracking.ios", "enable_install_attribution_tracking.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("enableInstallAttributionTracking.iosSwift", "enable_install_attribution_tracking.ios_swift"),
			common.SourceTypeIOSSwift,
		),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.androidKotlin", "use_native_sdk.android_kotlin"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.iosSwift", "use_native_sdk.ios_swift"),
		converter.Simple("useNativeSDK.unity", "use_native_sdk.unity"),
		converter.Simple("useNativeSDK.flutter", "use_native_sdk.flutter"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
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
