package intercom

import (
	"reflect"

	"github.com/go-playground/validator/v10"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

func init() {
	funcs.NewPattern(
		"intercom_single_line_1_100",
		`^(.{1,100})$`,
		"must be 1-100 characters and must not contain line breaks",
	)
}

// Source types from integrations-config destinations/intercom/db-config.json.
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
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

type intercomConfig struct {
	AppID               *string                  `mapstructure:"app_id" validate:"intercom_app_id_required,omitempty,dynamic_or_pattern=intercom_single_line_1_100"`
	APIKey              *string                  `mapstructure:"api_key" validate:"intercom_api_key_required,omitempty,dynamic_or_pattern=intercom_single_line_1_100"`
	APIServer           string                   `mapstructure:"api_server" validate:"omitempty,dynamic_or_oneof=standard eu au" default:"standard"`
	APIVersion          string                   `mapstructure:"api_version" validate:"omitempty,dynamic_or_oneof=v1 v2" default:"v2"`
	SendAnonymousID     *bool                    `mapstructure:"send_anonymous_id" default:"false"`
	UpdateLastRequestAt *bool                    `mapstructure:"update_last_request_at" default:"true"`
	MobileAPIKeyAndroid string                   `mapstructure:"mobile_api_key_android" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	MobileAPIKeyIOS     string                   `mapstructure:"mobile_api_key_ios" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	EventFiltering      *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK        *useNativeSDK            `mapstructure:"use_native_sdk"`
	ConnectionMode      common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement   common.ConsentManagement `mapstructure:"consent_management"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

type useNativeSDK struct {
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
	Web     *bool `mapstructure:"web"`
}

// schema.json splits Intercom's credentials by connection mode: device-mode
// sources need app_id, cloud-mode sources need api_key. Both conditions are
// keyed on connection_mode entries, which required_if cannot resolve, so they
// read the sibling off FieldLevel.Parent(). Each tag precedes omitempty, which
// would otherwise short-circuit it on the empty value being rejected.
func intercomModeConditional(want string, sources []string) validator.Func {
	return func(fl validator.FieldLevel) bool {
		parent := fl.Parent()
		if parent.Kind() == reflect.Pointer {
			parent = parent.Elem()
		}

		field := parent.FieldByName("ConnectionMode")
		if !field.IsValid() {
			return true
		}
		connectionMode, _ := field.Interface().(common.ConnectionMode)

		var matched bool
		for _, sourceType := range sources {
			if connectionMode[sourceType] == want {
				matched = true
				break
			}
		}
		if !matched {
			return true
		}
		return nonEmptyValue(fl)
	}
}

// app_id and api_key are *string so an absent key (nil) stays distinguishable
// from a present-but-empty one, which the pattern rejects.
func nonEmptyValue(fl validator.FieldLevel) bool {
	value := fl.Field()
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	return value.String() != ""
}

var intercomDeviceSources = []string{common.SourceTypeWeb, common.SourceTypeIOS, common.SourceTypeAndroid}

// NewDefinition returns the Intercom destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("appId", "app_id"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("apiServer", "api_server"),
		converter.Simple("apiVersion", "api_version"),
		converter.Simple("sendAnonymousId", "send_anonymous_id"),
		converter.Simple("updateLastRequestAt", "update_last_request_at"),
		converter.Gated(
			converter.Simple("mobileApiKeyAndroid.android", "mobile_api_key_android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("mobileApiKeyIOS.ios", "mobile_api_key_ios"),
			common.SourceTypeIOS,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "intercom",
		APIType:    "INTERCOM",
		Version:    1,
		Properties: properties,
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "intercom_app_id_required", Func: intercomModeConditional("device", intercomDeviceSources), CallEvenIfNull: true},
			{Tag: "intercom_api_key_required", Func: intercomModeConditional("cloud", sourceTypes), CallEvenIfNull: true},
		},
		SecretKeys: []string{"api_key"},
		NewConfig: func() any {
			return &intercomConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
