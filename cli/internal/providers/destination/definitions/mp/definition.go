package mp

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	funcs.NewPattern(
		"mp_percentage",
		`^(100|[1-9]?[0-9])$`,
		"must be an integer percentage from 0 to 100",
	)
}

// Source types from integrations-config destinations/mp/db-config.json.
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
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
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

type webBool struct {
	Web *bool `mapstructure:"web"`
}

type webString struct {
	Web string `mapstructure:"web" validate:"omitempty,dynamic_or_pattern=mp_percentage"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,excluded_with=Blacklist,dive,dynamic_or_pattern=single_line_100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,excluded_with=Whitelist,dive,dynamic_or_pattern=single_line_100"`
}

// mpConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/mp schema/defaultConfig, with terraform mappings ported where
// present and schema/defaultConfig-only account fields modelled to avoid erasing
// UI-set values during whole-config updates.
type mpConfig struct {
	Token                          string                   `mapstructure:"token" validate:"required,dynamic_or_pattern=single_line_100"`
	DataResidency                  string                   `mapstructure:"data_residency" validate:"required,oneof=us eu in"`
	IdentityMergeAPI               string                   `mapstructure:"identity_merge_api" validate:"required,oneof=simplified original"`
	ServiceAccountUserName         string                   `mapstructure:"service_account_user_name"`
	ServiceAccountSecret           string                   `mapstructure:"service_account_secret"`
	ProjectID                      string                   `mapstructure:"project_id"`
	UserDeletionAPI                string                   `mapstructure:"user_deletion_api" validate:"omitempty,oneof=engage task" default:"engage"`
	GDPRAPIToken                   string                   `mapstructure:"gdpr_api_token" validate:"required_if=UserDeletionAPI task,omitempty,dynamic_or_pattern=single_line_100"`
	StrictMode                     *bool                    `mapstructure:"strict_mode" default:"false"`
	IgnoreDNT                      *bool                    `mapstructure:"ignore_dnt" default:"false"`
	UseUserDefinedPageEventName    *bool                    `mapstructure:"use_user_defined_page_event_name" default:"false"`
	UserDefinedPageEventTemplate   string                   `mapstructure:"user_defined_page_event_template" validate:"required_if=UseUserDefinedPageEventName true,omitempty,dynamic_or_pattern=single_line_200" default:"Viewed {{ category }} {{ name }} page"`
	UseUserDefinedScreenEventName  *bool                    `mapstructure:"use_user_defined_screen_event_name" default:"false"`
	UserDefinedScreenEventTemplate string                   `mapstructure:"user_defined_screen_event_template" validate:"required_if=UseUserDefinedScreenEventName true,omitempty,dynamic_or_pattern=single_line_200" default:"Viewed {{ category }} {{ name }} screen"`
	DropTraitsInTrackEvent         *bool                    `mapstructure:"drop_traits_in_track_event" default:"false"`
	People                         *bool                    `mapstructure:"people" default:"false"`
	SetAllTraitsByDefault          *bool                    `mapstructure:"set_all_traits_by_default" default:"false"`
	SuperProperties                []string                 `mapstructure:"super_properties" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	SetOnceProperties              []string                 `mapstructure:"set_once_properties" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	UnionProperties                []string                 `mapstructure:"union_properties" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	AppendProperties               []string                 `mapstructure:"append_properties" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	PeopleProperties               []string                 `mapstructure:"people_properties" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	EventIncrements                []string                 `mapstructure:"event_increments" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	PropIncrements                 []string                 `mapstructure:"prop_increments" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	GroupKeySettings               []string                 `mapstructure:"group_key_settings" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	ConsolidatedPageCalls          *bool                    `mapstructure:"consolidated_page_calls" default:"true"`
	TrackCategorizedPages          *bool                    `mapstructure:"track_categorized_pages" default:"false"`
	TrackNamedPages                *bool                    `mapstructure:"track_named_pages" default:"false"`
	SourceName                     string                   `mapstructure:"source_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	SessionReplayPercentage        *webString               `mapstructure:"session_replay_percentage"`
	CrossSubdomainCookie           *bool                    `mapstructure:"cross_subdomain_cookie" default:"false"`
	PersistenceType                string                   `mapstructure:"persistence_type" validate:"omitempty,oneof=none cookie localStorage" default:"cookie"`
	PersistenceName                string                   `mapstructure:"persistence_name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	SecureCookie                   *bool                    `mapstructure:"secure_cookie" default:"false"`
	EventFiltering                 *eventFiltering          `mapstructure:"event_filtering"`
	UseNativeSDK                   *webBool                 `mapstructure:"use_native_sdk"`
	UseNewMapping                  *bool                    `mapstructure:"use_new_mapping" default:"false"`
	ConnectionMode                 common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement              common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Mixpanel destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("token", "token"),
		converter.Simple("dataResidency", "data_residency"),
		converter.Simple("identityMergeApi", "identity_merge_api"),
		converter.Simple("serviceAccountUserName", "service_account_user_name"),
		converter.Simple("serviceAccountSecret", "service_account_secret"),
		converter.Simple("projectId", "project_id"),
		converter.Simple("userDeletionApi", "user_deletion_api"),
		converter.Simple("gdprApiToken", "gdpr_api_token"),
		converter.Simple("strictMode", "strict_mode"),
		converter.Simple("ignoreDnt", "ignore_dnt"),
		converter.Simple("useUserDefinedPageEventName", "use_user_defined_page_event_name"),
		converter.Simple("userDefinedPageEventTemplate", "user_defined_page_event_template"),
		converter.Simple("useUserDefinedScreenEventName", "use_user_defined_screen_event_name"),
		converter.Simple("userDefinedScreenEventTemplate", "user_defined_screen_event_template"),
		converter.Simple("dropTraitsInTrackEvent", "drop_traits_in_track_event"),
		converter.Simple("people", "people"),
		converter.Simple("setAllTraitsByDefault", "set_all_traits_by_default"),
		converter.ArrayWithStrings("superProperties", "property", "super_properties"),
		converter.ArrayWithStrings("setOnceProperties", "property", "set_once_properties"),
		converter.ArrayWithStrings("unionProperties", "property", "union_properties"),
		converter.ArrayWithStrings("appendProperties", "property", "append_properties"),
		converter.ArrayWithStrings("peopleProperties", "property", "people_properties"),
		converter.ArrayWithStrings("eventIncrements", "property", "event_increments"),
		converter.ArrayWithStrings("propIncrements", "property", "prop_increments"),
		converter.ArrayWithStrings("groupKeySettings", "groupKey", "group_key_settings"),
		converter.Simple("consolidatedPageCalls", "consolidated_page_calls"),
		converter.Simple("trackCategorizedPages", "track_categorized_pages"),
		converter.Simple("trackNamedPages", "track_named_pages"),
		converter.Simple("sourceName", "source_name"),
		converter.Gated(
			converter.Simple("sessionReplayPercentage.web", "session_replay_percentage.web"),
			common.SourceTypeWeb,
		),
		converter.Simple("crossSubdomainCookie", "cross_subdomain_cookie"),
		converter.Simple("persistenceType", "persistence_type"),
		converter.Simple("persistenceName", "persistence_name"),
		converter.Simple("secureCookie", "secure_cookie"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("useNativeSDK.web", "use_native_sdk.web"),
		converter.Simple("useNewMapping", "use_new_mapping"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "mp",
		APIType:    "MP",
		Version:    1,
		Properties: properties,
		// Deliberately broader than db-config secretKeys, which lists only
		// gdprApiToken. token is Sensitive in terraform, and serviceAccountSecret
		// is a credential by name and function though neither source classifies it
		// (terraform does not model it at all). The cost is accepted: the API
		// returns both, so each reads back as an unknown secret and re-applies on
		// every plan — the same trade recorded for customerio and posthog.
		SecretKeys: []string{"token", "gdpr_api_token", "service_account_secret"},
		NewConfig: func() any {
			return &mpConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
