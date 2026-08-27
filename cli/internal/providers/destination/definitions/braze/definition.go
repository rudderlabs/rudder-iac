package braze

import (
	"reflect"

	"github.com/go-playground/validator/v10"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// Source types from integrations-config destinations/braze/db-config.json.
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
	common.SourceTypeAndroid:       {"cloud", "device", "hybrid"},
	common.SourceTypeAndroidKotlin: {"cloud", "device", "hybrid"},
	common.SourceTypeIOS:           {"cloud", "device", "hybrid"},
	common.SourceTypeIOSSwift:      {"cloud", "device", "hybrid"},
	common.SourceTypeWeb:           {"cloud", "device", "hybrid"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud", "device"},
	common.SourceTypeFlutter:       {"cloud", "device"},
	common.SourceTypeCordova:       {"cloud"},
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
	Web           *bool `mapstructure:"web"`
	ReactNative   *bool `mapstructure:"react_native"`
	Flutter       *bool `mapstructure:"flutter"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

// brazeConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/braze schema/defaultConfig; validation constraints mirror schema.json
// where they can be expressed without making mode-dependent import paths stricter
// than the backend.
type brazeConfig struct {
	RestAPIKey                           string                   `mapstructure:"rest_api_key" validate:"braze_rest_api_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	AppKey                               string                   `mapstructure:"app_key" validate:"braze_app_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	DataCenter                           string                   `mapstructure:"data_center" validate:"required,dynamic_or_oneof=US-01 US-02 US-03 US-04 US-05 US-06 US-07 US-08 EU-01 EU-02 EU-03 AU-01"`
	EnableSubscriptionGroupInGroupCall   *bool                    `mapstructure:"enable_subscription_group_in_group_call" default:"false"`
	EnableNestedArrayOperations          *bool                    `mapstructure:"enable_nested_array_operations" default:"false"`
	SendPurchaseEventWithExtraProperties *bool                    `mapstructure:"send_purchase_event_with_extra_properties" default:"false"`
	TrackAnonymousUser                   *webBool                 `mapstructure:"track_anonymous_user"`
	SupportDedup                         *bool                    `mapstructure:"support_dedup" default:"false"`
	EnableBrazeLogging                   *webBool                 `mapstructure:"enable_braze_logging"`
	EnablePushNotification               *webBool                 `mapstructure:"enable_push_notification"`
	AllowUserSuppliedJavascript          *webBool                 `mapstructure:"allow_user_supplied_javascript"`
	EventFiltering                       *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK                         *useNativeSDK            `mapstructure:"use_native_sdk"`
	UseEcommerceRecommendedEvents        *bool                    `mapstructure:"use_ecommerce_recommended_events" default:"true"`
	UsePlatformSpecificAPIKeys           *bool                    `mapstructure:"use_platform_specific_api_keys"`
	AndroidAPIKey                        string                   `mapstructure:"android_api_key" validate:"braze_android_api_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	IOSAPIKey                            string                   `mapstructure:"ios_api_key" validate:"braze_ios_api_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	WebAPIKey                            string                   `mapstructure:"web_api_key" validate:"braze_web_api_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	ConnectionMode                       common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement                    common.ConsentManagement `mapstructure:"consent_management"`
}

// schema.json gates each API key on connection_mode — a map — combined with the
// sibling use_platform_specific_api_keys flag. required_if cannot resolve a map
// entry, so these read both off FieldLevel.Parent(), the same shape as
// iterable's packageNameConditional, scoped here via ConfigValidateFuncs.
//
// Each tag must precede omitempty in the validate list: omitempty short-circuits
// every validator after it on an empty value, which is the case these reject.
func brazeParent(fl validator.FieldLevel) (common.ConnectionMode, *bool, bool) {
	parent := fl.Parent()
	if parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}

	modeField := parent.FieldByName("ConnectionMode")
	if !modeField.IsValid() {
		return nil, nil, false
	}
	mode, _ := modeField.Interface().(common.ConnectionMode)

	var platformSpecific *bool
	if f := parent.FieldByName("UsePlatformSpecificAPIKeys"); f.IsValid() {
		platformSpecific, _ = f.Interface().(*bool)
	}
	return mode, platformSpecific, true
}

func anyMode(mode common.ConnectionMode, sourceTypes []string, want ...string) bool {
	for _, sourceType := range sourceTypes {
		for _, w := range want {
			if mode[sourceType] == w {
				return true
			}
		}
	}
	return false
}

// restApiKey is required whenever any source connects in cloud or hybrid mode.
func restAPIKeyConditional(fl validator.FieldLevel) bool {
	mode, _, ok := brazeParent(fl)
	if !ok || !anyMode(mode, sourceTypes, "cloud", "hybrid") {
		return true
	}
	return fl.Field().String() != ""
}

// appKey is the single-key alternative: required for any device or hybrid source
// when platform-specific keys are off.
func appKeyConditional(fl validator.FieldLevel) bool {
	mode, platformSpecific, ok := brazeParent(fl)
	if !ok || platformSpecific == nil || *platformSpecific {
		return true
	}
	if !anyMode(mode, sourceTypes, "device", "hybrid") {
		return true
	}
	return fl.Field().String() != ""
}

// With platform-specific keys on, react_native and flutter in device mode need
// both the Android and the iOS key, so each appears in both lists below.
var (
	brazeAndroidSources = []string{
		common.SourceTypeAndroid, common.SourceTypeAndroidKotlin,
		common.SourceTypeReactNative, common.SourceTypeFlutter,
	}
	brazeIOSSources = []string{
		common.SourceTypeIOS, common.SourceTypeIOSSwift,
		common.SourceTypeReactNative, common.SourceTypeFlutter,
	}
	brazeWebSources = []string{common.SourceTypeWeb}
)

func platformKeyConditional(sources []string) validator.Func {
	return func(fl validator.FieldLevel) bool {
		mode, platformSpecific, ok := brazeParent(fl)
		if !ok || platformSpecific == nil || !*platformSpecific {
			return true
		}
		if !anyMode(mode, sources, "device", "hybrid") {
			return true
		}
		return fl.Field().String() != ""
	}
}

// NewDefinition returns the Braze destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("restApiKey", "rest_api_key"),
		converter.Simple("appKey", "app_key"),
		converter.Simple("dataCenter", "data_center"),
		converter.Simple("enableSubscriptionGroupInGroupCall", "enable_subscription_group_in_group_call"),
		converter.Simple("enableNestedArrayOperations", "enable_nested_array_operations"),
		converter.Simple("sendPurchaseEventWithExtraProperties", "send_purchase_event_with_extra_properties"),
		converter.Gated(
			converter.Simple("trackAnonymousUser.web", "track_anonymous_user.web"),
			common.SourceTypeWeb,
		),
		converter.Simple("supportDedup", "support_dedup"),
		converter.Gated(
			converter.Simple("enableBrazeLogging.web", "enable_braze_logging.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enablePushNotification.web", "enable_push_notification.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("allowUserSuppliedJavascript.web", "allow_user_supplied_javascript.web"),
			common.SourceTypeWeb,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.androidKotlin", "use_native_sdk.android_kotlin"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Simple("useNativeSDK.iosSwift", "use_native_sdk.ios_swift"),
		converter.Simple("useNativeSDK.reactnative", "use_native_sdk.react_native"),
		converter.Simple("useNativeSDK.flutter", "use_native_sdk.flutter"),
		converter.Simple("useEcommerceRecommendedEvents", "use_ecommerce_recommended_events"),
		converter.Simple("usePlatformSpecificApiKeys", "use_platform_specific_api_keys"),
		converter.Simple("androidApiKey", "android_api_key"),
		converter.Simple("iOSApiKey", "ios_api_key"),
		converter.Simple("webApiKey", "web_api_key"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "braze",
		APIType:    "BRAZE",
		Version:    1,
		Properties: properties,
		// db-config declares only restApiKey as a secret. Terraform also marks appKey
		// sensitive, but db-config is authoritative for CLI write-only wrapping.
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "braze_rest_api_key_required", Func: restAPIKeyConditional},
			{Tag: "braze_app_key_required", Func: appKeyConditional},
			{Tag: "braze_android_api_key_required", Func: platformKeyConditional(brazeAndroidSources)},
			{Tag: "braze_ios_api_key_required", Func: platformKeyConditional(brazeIOSSources)},
			{Tag: "braze_web_api_key_required", Func: platformKeyConditional(brazeWebSources)},
		},
		SecretKeys: []string{"rest_api_key"},
		NewConfig: func() any {
			return &brazeConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
