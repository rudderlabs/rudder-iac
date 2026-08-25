package ga4

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
	// schema.json requires the gtag measurement ID to carry the G- prefix.
	funcs.NewPattern(
		"ga4_measurement_id",
		`^(G-.{1,100})$`,
		"must start with 'G-' and be at most 101 characters",
	)

	// sdkBaseUrl's format is constrained only under typesOfClient=gtag >
	// connectionMode.web=device (schema.json declares no sdkBaseUrl property
	// outside that branch); RE2 has no lookahead, so the ngrok guard becomes a
	// reject pattern, and the empty alternative is preserved because upstream
	// allows an unset value.
	funcs.NewPatternWithReject(
		"ga4_sdk_base_url",
		`^(?:https?://)?[\w.-]+(?:\.[\w.-]+)+[\w\-._~:/?#[\]@!$&'()*+,;=.]*|^$`,
		`\.ngrok\.io`,
		"must be a domain URL and must not use ngrok",
	)
}

// sdkBaseURLConditional enforces the named pattern given as its tag param
// (ga4_sdk_base_url) only when client_type=gtag and connection_mode.web=device
// — outside that condition, schema.json imposes no format constraint on
// sdk_base_url at all. Templates are still accepted unconditionally within the
// branch, matching dynamic_or_pattern elsewhere in this destination.
//
// The condition spans two sibling fields (client_type and connection_mode.web),
// which a named pattern or a built-in required_if-style tag cannot express —
// required_if only compares against direct struct fields, and connection_mode
// is a map. This custom tag reads both via FieldLevel.Parent() instead.
// It is registered via NewDefinition's ConfigValidateFuncs, scoped to this
// definition alone, rather than a global default validator: it is GA4-specific,
// not a reusable convention other destinations opt into.
func sdkBaseURLConditional(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	parent := fl.Parent()
	if parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}

	clientType := parent.FieldByName("ClientType").String()
	connectionMode, _ := parent.FieldByName("ConnectionMode").Interface().(common.ConnectionMode)

	if clientType != "gtag" || connectionMode["web"] != "device" {
		return true
	}

	if definitions.IsTemplateConfigValue(value) {
		return true
	}
	return funcs.MatchPattern(fl.Param(), value)
}

// Source types from integrations-config destinations/ga4/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeCloud,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud", "device"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud", "device"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device", "hybrid"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

type piiProperty struct {
	PIIProperty string `mapstructure:"pii_property" validate:"omitempty,dynamic_or_pattern=single_line_100"`
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

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type webCapturePageView struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=rs gtag"`
}

// ga4Config is the local YAML config model. Field set mirrors terraform-provider
// destination_google_analytics4 mappings; validation constraints mirror
// overlapping schema.json rules (required, enums, client-type conditionals).
type ga4Config struct {
	APISecret             string              `mapstructure:"api_secret" validate:"required,dynamic_or_pattern=single_line_100"`
	ClientType            string              `mapstructure:"client_type" validate:"required,dynamic_or_oneof=gtag firebase"`
	MeasurementID         string              `mapstructure:"measurement_id" validate:"required_if=ClientType gtag,omitempty,dynamic_or_pattern=ga4_measurement_id"`
	FirebaseAppID         string              `mapstructure:"firebase_app_id" validate:"required_if=ClientType firebase,omitempty,dynamic_or_pattern=single_line_100"`
	DebugMode             *bool               `mapstructure:"debug_mode"`
	SDKBaseURL            string              `mapstructure:"sdk_base_url" validate:"omitempty,ga4_sdk_base_url_conditional=ga4_sdk_base_url"`
	ServerContainerURL    string              `mapstructure:"server_container_url"`
	PIIPropertiesToIgnore []piiProperty       `mapstructure:"pii_properties_to_ignore" validate:"omitempty,dive"`
	EventFiltering        *eventFiltering     `mapstructure:"event_filtering"`
	UseNativeSDK          *useNativeSDK       `mapstructure:"use_native_sdk"`
	CapturePageView       *webCapturePageView `mapstructure:"capture_page_view"`
	ExtendPageViewParams  *webBool            `mapstructure:"extend_page_view_params"`
	// schema.json declares useNativeSDKToSend under web but terraform does not map
	// it; modelled so a CLI apply does not erase a UI-set value.
	UseNativeSDKToSend    *webBool                 `mapstructure:"use_native_sdk_to_send"`
	DebugView             *webBool                 `mapstructure:"debug_view"`
	OverrideClientSession *webBool                 `mapstructure:"override_client_and_session_ids"`
	ConnectionMode        common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement     common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Analytics 4 (GA4) destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("apiSecret", "api_secret"),
		converter.Simple("typesOfClient", "client_type"),
		converter.Simple("measurementId", "measurement_id"),
		converter.Simple("firebaseAppId", "firebase_app_id"),
		converter.Simple("debugMode", "debug_mode"),
		converter.Simple("sdkBaseUrl", "sdk_base_url"),
		converter.Simple("serverContainerUrl", "server_container_url"),
		converter.ArrayWithObjects("piiPropertiesToIgnore", "pii_properties_to_ignore", map[string]any{
			"piiProperty": "pii_property",
		}),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNativeSDK.android", "use_native_sdk.android"),
		converter.Simple("useNativeSDK.ios", "use_native_sdk.ios"),
		converter.Gated(
			converter.Simple("capturePageView.web", "capture_page_view.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("debugView.web", "debug_view.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("overrideClientAndSessionId.web", "override_client_and_session_ids.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("extendPageViewParams.web", "extend_page_view_params.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("useNativeSDKToSend.web", "use_native_sdk_to_send.web"),
			common.SourceTypeWeb,
		),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "ga4",
		APIType:    "GA4",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"api_secret"},
		NewConfig: func() any {
			return &ga4Config{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		ConfigValidateFuncs: []rules.CustomValidateFunc{
			{Tag: "ga4_sdk_base_url_conditional", Func: sdkBaseURLConditional},
		},
	}
}
