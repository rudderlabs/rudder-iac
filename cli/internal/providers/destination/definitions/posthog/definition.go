package posthog

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	// schema.json yourInstance: (?!.*\.ngrok\.io)^(.{0,100})$ — allow length, reject ngrok hosts.
	funcs.NewPatternWithReject(
		"posthog_endpoint",
		`^(.{0,100})$`,
		`\.ngrok\.io`,
		"must be at most 100 characters and must not contain .ngrok.io",
	)
}

// Source types from integrations-config destinations/posthog/db-config.json
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
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud", "device"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type webPersonProfiles struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_oneof=always identified_only"`
}

type xhrHeader struct {
	Key   string `mapstructure:"key" validate:"omitempty,max=100"`
	Value string `mapstructure:"value" validate:"omitempty,max=100"`
}

type propertyBlacklistItem struct {
	Property string `mapstructure:"property"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist"`
	Blacklist []string `mapstructure:"blacklist"`
}

type useNativeSDK struct {
	Web *bool `mapstructure:"web"`
}

// posthogConfig is the local YAML config model. Field set mirrors terraform
// destination_posthog mappings; validation constraints mirror schema.json.
type posthogConfig struct {
	APIKey                        string                   `mapstructure:"api_key" validate:"required,min=1,max=100"`
	Endpoint                      string                   `mapstructure:"endpoint" validate:"omitempty,pattern=posthog_endpoint"`
	UseV2Group                    *bool                    `mapstructure:"use_v2_group"`
	EventFiltering                *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK                  *useNativeSDK            `mapstructure:"use_native_sdk"`
	Autocapture                   *webBool                 `mapstructure:"autocapture"`
	CapturePageView               *webBool                 `mapstructure:"capture_page_view"`
	DisableSessionRecording       *webBool                 `mapstructure:"disable_session_recording"`
	EnableLocalStoragePersistence *webBool                 `mapstructure:"enable_local_storage_persistence"`
	PersonProfiles                *webPersonProfiles       `mapstructure:"person_profiles"`
	XHRHeaders                    []xhrHeader              `mapstructure:"xhr_headers" validate:"omitempty,dive"`
	PropertyBlacklist             []propertyBlacklistItem  `mapstructure:"property_blacklist" validate:"omitempty,dive"`
	ConsentManagement             common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the PostHog destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("yourInstance", "endpoint", converter.SkipZeroValue),
		converter.Simple("teamApiKey", "api_key"),
		converter.Simple("useV2Group", "use_v2_group"),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Gated(
			converter.Simple("disableSessionRecording.web", "disable_session_recording.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("capturePageView.web", "capture_page_view.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("autocapture.web", "autocapture.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("enableLocalStoragePersistence.web", "enable_local_storage_persistence.web"),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.ArrayWithObjects("xhrHeaders.web", "xhr_headers", map[string]any{
				"key":   "key",
				"value": "value",
			}),
			common.SourceTypeWeb,
		),
		// terraform maps propertyBlacklist (lowercase L); upstream db-config/schema use propertyBlackList.
		converter.Gated(
			converter.ArrayWithObjects("propertyBlacklist.web", "property_blacklist", map[string]any{
				"property": "property",
			}),
			common.SourceTypeWeb,
		),
		converter.Gated(
			converter.Simple("personProfiles.web", "person_profiles.web"),
			common.SourceTypeWeb,
		),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "posthog",
		APIType:    "POSTHOG",
		Version:    1,
		Properties: properties,
		// db-config secretKeys is empty; terraform marks api_key Sensitive.
		SecretKeys: []string{"api_key"},
		NewConfig: func() any {
			return &posthogConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
