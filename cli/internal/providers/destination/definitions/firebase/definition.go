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

type sourceConnectionMode struct {
	Android       *string `mapstructure:"android" validate:"omitempty,oneof=device"`
	AndroidKotlin *string `mapstructure:"android_kotlin" validate:"omitempty,oneof=device"`
	IOS           *string `mapstructure:"ios" validate:"omitempty,oneof=device"`
	IOSSwift      *string `mapstructure:"ios_swift" validate:"omitempty,oneof=device"`
	Unity         *string `mapstructure:"unity" validate:"omitempty,oneof=device"`
	ReactNative   *string `mapstructure:"react_native" validate:"omitempty,oneof=device"`
	Flutter       *string `mapstructure:"flutter" validate:"omitempty,oneof=device"`
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

type oneTrustCookieCategory struct {
	OneTrustCookieCategory string `mapstructure:"one_trust_cookie_category" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type oneTrustCookieCategories struct {
	Android     []oneTrustCookieCategory `mapstructure:"android" validate:"omitempty,dive"`
	IOS         []oneTrustCookieCategory `mapstructure:"ios" validate:"omitempty,dive"`
	Unity       []oneTrustCookieCategory `mapstructure:"unity" validate:"omitempty,dive"`
	ReactNative []oneTrustCookieCategory `mapstructure:"react_native" validate:"omitempty,dive"`
	Flutter     []oneTrustCookieCategory `mapstructure:"flutter" validate:"omitempty,dive"`
}

type ketchConsentPurpose struct {
	Purpose string `mapstructure:"purpose" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type ketchConsentPurposes struct {
	Android     []ketchConsentPurpose `mapstructure:"android" validate:"omitempty,dive"`
	IOS         []ketchConsentPurpose `mapstructure:"ios" validate:"omitempty,dive"`
	Unity       []ketchConsentPurpose `mapstructure:"unity" validate:"omitempty,dive"`
	ReactNative []ketchConsentPurpose `mapstructure:"react_native" validate:"omitempty,dive"`
	Flutter     []ketchConsentPurpose `mapstructure:"flutter" validate:"omitempty,dive"`
}

// firebaseConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/firebase schema/defaultConfig, plus explicit device-only connection mode
// support from the Terraform mapping.
type firebaseConfig struct {
	EventFiltering           *eventFiltering           `mapstructure:"event_filtering"`
	ConnectionMode           *sourceConnectionMode     `mapstructure:"connection_mode"`
	UseNativeSDK             *useNativeSDK             `mapstructure:"use_native_sdk"`
	OneTrustCookieCategories *oneTrustCookieCategories `mapstructure:"one_trust_cookie_categories"`
	KetchConsentPurposes     *ketchConsentPurposes     `mapstructure:"ketch_consent_purposes"`
	ConsentManagement        common.ConsentManagement  `mapstructure:"consent_management"`
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
		converter.Simple("connectionMode.android", "connection_mode.android"),
		converter.Simple("connectionMode.androidKotlin", "connection_mode.android_kotlin"),
		converter.Simple("connectionMode.ios", "connection_mode.ios"),
		converter.Simple("connectionMode.iosSwift", "connection_mode.ios_swift"),
		converter.Simple("connectionMode.unity", "connection_mode.unity"),
		converter.Simple("connectionMode.reactnative", "connection_mode.react_native"),
		converter.Simple("connectionMode.flutter", "connection_mode.flutter"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.androidKotlin", "use_native_sdk.android_kotlin"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.iosSwift", "use_native_sdk.ios_swift"),
		converter.Simple("useNativeSDK.unity", "use_native_sdk.unity"),
		converter.Simple("useNativeSDK.reactnative", "use_native_sdk.react_native"),
		converter.Simple("useNativeSDK.flutter", "use_native_sdk.flutter"),
		converter.ArrayWithObjects("oneTrustCookieCategories.android", "one_trust_cookie_categories.android", map[string]any{
			"oneTrustCookieCategory": "one_trust_cookie_category",
		}),
		converter.ArrayWithObjects("oneTrustCookieCategories.ios", "one_trust_cookie_categories.ios", map[string]any{
			"oneTrustCookieCategory": "one_trust_cookie_category",
		}),
		converter.ArrayWithObjects("oneTrustCookieCategories.unity", "one_trust_cookie_categories.unity", map[string]any{
			"oneTrustCookieCategory": "one_trust_cookie_category",
		}),
		converter.ArrayWithObjects("oneTrustCookieCategories.reactnative", "one_trust_cookie_categories.react_native", map[string]any{
			"oneTrustCookieCategory": "one_trust_cookie_category",
		}),
		converter.ArrayWithObjects("oneTrustCookieCategories.flutter", "one_trust_cookie_categories.flutter", map[string]any{
			"oneTrustCookieCategory": "one_trust_cookie_category",
		}),
		converter.ArrayWithObjects("ketchConsentPurposes.android", "ketch_consent_purposes.android", map[string]any{
			"purpose": "purpose",
		}),
		converter.ArrayWithObjects("ketchConsentPurposes.ios", "ketch_consent_purposes.ios", map[string]any{
			"purpose": "purpose",
		}),
		converter.ArrayWithObjects("ketchConsentPurposes.unity", "ketch_consent_purposes.unity", map[string]any{
			"purpose": "purpose",
		}),
		converter.ArrayWithObjects("ketchConsentPurposes.reactnative", "ketch_consent_purposes.react_native", map[string]any{
			"purpose": "purpose",
		}),
		converter.ArrayWithObjects("ketchConsentPurposes.flutter", "ketch_consent_purposes.flutter", map[string]any{
			"purpose": "purpose",
		}),
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
