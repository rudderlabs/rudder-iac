package firebase

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/firebase/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeUnity,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"device"},
	common.SourceTypeAndroidKotlin: {"device"},
	common.SourceTypeIOS:           {"device"},
	common.SourceTypeIOSSwift:      {"device"},
	common.SourceTypeUnity:         {"device"},
	common.SourceTypeReactNative:   {"device"},
	common.SourceTypeFlutter:       {"device"},
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDK struct {
	Android       *bool `mapstructure:"android"`
	AndroidKotlin *bool `mapstructure:"android_kotlin"`
	IOS           *bool `mapstructure:"ios"`
	IOSSwift      *bool `mapstructure:"ios_swift"`
	Unity         *bool `mapstructure:"unity"`
	ReactNative   *bool `mapstructure:"react_native"`
	Flutter       *bool `mapstructure:"flutter"`
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff. See DEX-696 Discrepancy 3.
//
// connection_mode is deliberately absent too. db-config lists connectionMode
// under all seven source types, but schema.json declares no such property, and
// schema.json is the authority on the config surface. ConnectionModes below
// still advertises the supported modes as metadata.
//
// firebaseConfig is the local YAML config model. Field set mirrors the keys
// upstream declares in schema.json and db-config.json destConfig; validation
// constraints mirror schema.json.
type firebaseConfig struct {
	EventFiltering    *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK      *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Firebase destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.androidKotlin", "use_native_sdk.android_kotlin"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.iosSwift", "use_native_sdk.ios_swift"),
		converter.Simple("useNativeSDK.unity", "use_native_sdk.unity"),
		converter.Simple("useNativeSDK.reactnative", "use_native_sdk.react_native"),
		converter.Simple("useNativeSDK.flutter", "use_native_sdk.flutter"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "firebase",
		APIType:    "FIREBASE",
		Version:    1,
		Properties: properties,
		NewConfig: func() any {
			return &firebaseConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
