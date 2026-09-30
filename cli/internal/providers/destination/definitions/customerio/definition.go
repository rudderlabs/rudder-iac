package customerio

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
	funcs.NewPatternWithReject(
		"customerio_write_key",
		`^(.{0,100})$`,
		`^(?:\{\{.*\}\}|env[.].*)$`,
		"must be at most 100 characters, must not contain line breaks, and must not be a template",
	)
}

// Source types from integrations-config destinations/customerio/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

type customerioConfig struct {
	SiteID               string `mapstructure:"site_id" validate:"customerio_site_id_required,omitempty,dynamic_or_pattern=single_line_100"`
	APIKey               string `mapstructure:"api_key" validate:"customerio_api_key_required,omitempty,dynamic_or_pattern=single_line_100"`
	DeviceTokenEventName string `mapstructure:"device_token_event_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Datacenter           string `mapstructure:"datacenter" validate:"required,oneof=US EU"`
	// Match backend defaults so omitted API and SDK versions do not cause a
	// perpetual diff against the persisted destination config.
	APIVersion           string         `mapstructure:"api_version" validate:"omitempty,oneof=v1 v2" default:"v2"`
	UserIDIdentifierType string         `mapstructure:"user_id_identifier_type" validate:"required_if=APIVersion v2,omitempty,oneof=id email phone cio_id"`
	SDKVersion           *webSDKVersion `mapstructure:"sdk_version"`
	WriteKey             *webString     `mapstructure:"write_key" validate:"customerio_write_key_block_required"`
	AnonymousInApp       *webBool       `mapstructure:"anonymous_in_app"`
	SendPageNameInSDK    *webBool       `mapstructure:"send_page_name_in_sdk"`
	// data_use_in_app configures the v1 web SDK. The upstream schema still
	// accepts it with v2, so the CLI documents but does not enforce that scope.
	DataUseInApp                *webBool                 `mapstructure:"data_use_in_app"`
	AutoTrackDeviceAttributes   *mobileSourceBools       `mapstructure:"auto_track_device_attributes"`
	BackgroundQueueMinTasks     *androidString           `mapstructure:"background_queue_min_number_of_tasks"`
	BackgroundQueueSecondsDelay *androidString           `mapstructure:"background_queue_seconds_delay"`
	EventFiltering              *eventFiltering          `mapstructure:"event_filtering"`
	ConnectionMode              common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement           common.ConsentManagement `mapstructure:"consent_management"`
}

type webSDKVersion struct {
	Web string `mapstructure:"web" validate:"omitempty,oneof=v1 v2" default:"v2"`
}

type webString struct {
	Web string `mapstructure:"web" validate:"customerio_write_key_required,omitempty,pattern=customerio_write_key"`
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type mobileSourceBools struct {
	Android *bool `mapstructure:"android"`
	IOS     *bool `mapstructure:"ios"`
}

type androidString struct {
	Android string `mapstructure:"android" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

// Customer.io's credential branches depend on connection_mode map entries,
// which built-in required_if cannot inspect; definition-scoped validators read
// the decoded root config instead.
func customerioConfigFromField(fl validator.FieldLevel) (customerioConfig, bool) {
	root := fl.Top()
	for root.IsValid() && root.Kind() == reflect.Pointer {
		if root.IsNil() {
			return customerioConfig{}, false
		}
		root = root.Elem()
	}
	if !root.IsValid() || !root.CanInterface() {
		return customerioConfig{}, false
	}

	config, ok := root.Interface().(customerioConfig)
	return config, ok
}

func customerioSDKVersion(config customerioConfig) string {
	if config.SDKVersion == nil || config.SDKVersion.Web == "" {
		return "v2"
	}
	return config.SDKVersion.Web
}

func isWebDeviceV2(config customerioConfig) bool {
	return config.ConnectionMode[common.SourceTypeWeb] == "device" &&
		customerioSDKVersion(config) == "v2"
}

func isWebDeviceOnly(config customerioConfig) bool {
	return len(config.ConnectionMode) == 1 && config.ConnectionMode[common.SourceTypeWeb] == "device"
}

func customerioSiteIDRequired(fl validator.FieldLevel) bool {
	config, ok := customerioConfigFromField(fl)
	return !ok || isWebDeviceOnly(config) && isWebDeviceV2(config) || fl.Field().String() != ""
}

func customerioAPIKeyRequired(fl validator.FieldLevel) bool {
	config, ok := customerioConfigFromField(fl)
	return !ok || isWebDeviceOnly(config) || fl.Field().String() != ""
}

func customerioWriteKeyBlockRequired(fl validator.FieldLevel) bool {
	config, ok := customerioConfigFromField(fl)
	if !ok || !isWebDeviceV2(config) {
		return true
	}

	field := fl.Field()
	return field.Kind() != reflect.Pointer || !field.IsNil()
}

func customerioWriteKeyRequired(fl validator.FieldLevel) bool {
	config, ok := customerioConfigFromField(fl)
	return !ok || !isWebDeviceV2(config) || fl.Field().String() != ""
}

// NewDefinition returns the Customer.io destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("siteID", "site_id"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("deviceTokenEventName", "device_token_event_name"),
		converter.Simple("datacenter", "datacenter"),
		converter.Simple("apiVersion", "api_version"),
		converter.Simple("userIdIdentifierType", "user_id_identifier_type"),
		converter.Gated(
			converter.Simple("sdkVersion.web", "sdk_version.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("writeKey.web", "write_key.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("anonymousInApp.web", "anonymous_in_app.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("sendPageNameInSDK.web", "send_page_name_in_sdk.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("dataUseInApp.web", "data_use_in_app.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("autoTrackDeviceAttributes.android", "auto_track_device_attributes.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("autoTrackDeviceAttributes.ios", "auto_track_device_attributes.ios"),
			common.SourceTypeIOS,
		),
		converter.Gated(
			converter.Simple("backgroundQueueMinNumberOfTasks.android", "background_queue_min_number_of_tasks.android"),
			common.SourceTypeAndroid,
		),
		converter.Gated(
			converter.Simple("backgroundQueueSecondsDelay.android", "background_queue_seconds_delay.android"),
			common.SourceTypeAndroid,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "customerio",
		APIType:    "CUSTOMERIO",
		Version:    1,
		Properties: properties,
		// db-config lists no secretKeys for customerio, but terraform marks api_key
		// Sensitive and it is a real credential, so it is wrapped write-only here.
		// Note the API does still return apiKey, so the value is never absent from
		// remote state — see the churn note in the PR.
		SecretKeys: []string{"api_key", "site_id"},
		NewConfig: func() any {
			return &customerioConfig{}
		},
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "customerio_site_id_required", Func: customerioSiteIDRequired},
			{Tag: "customerio_api_key_required", Func: customerioAPIKeyRequired},
			{Tag: "customerio_write_key_block_required", Func: customerioWriteKeyBlockRequired, CallEvenIfNull: true},
			{Tag: "customerio_write_key_required", Func: customerioWriteKeyRequired, CallEvenIfNull: true},
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// Declared as upstream narrows it, though rETL still refuses Customer.io:
		// its connections run the destination-specific flow (retl ClassifyFlow).
		SyncBehaviours: []string{"upsert", "mirror"},
	}
}
