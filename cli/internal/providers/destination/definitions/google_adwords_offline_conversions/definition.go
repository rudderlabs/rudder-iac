package googleadwordsofflineconversions

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/google_adwords_offline_conversions/db-config.json,
// restricted to event-stream-reachable types managed by the CLI.
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
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

type offlineConversionTypeMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,oneof=click call store"`
}

type eventNameMapping struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_100"`
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff. See DEX-696 Discrepancy 3.
//
// googleAdwordsOfflineConversionsConfig is the local YAML config model. Field
// set mirrors integrations-config destinations/google_adwords_offline_conversions
// defaultConfig; validation constraints mirror schema.json.
type googleAdwordsOfflineConversionsConfig struct {
	RudderAccountID                       string                         `mapstructure:"rudder_account_id" validate:"required"`
	CustomerID                            string                         `mapstructure:"customer_id" validate:"required,dynamic_or_pattern=single_line_100"`
	SubAccount                            *bool                          `mapstructure:"sub_account" default:"false"`
	LoginCustomerID                       string                         `mapstructure:"login_customer_id" validate:"required_if=SubAccount true,omitempty,dynamic_or_pattern=single_line_100"`
	EventsToOfflineConversionsTypeMapping []offlineConversionTypeMapping `mapstructure:"events_to_offline_conversions_type_mapping" validate:"omitempty,dive"`
	EventsToConversionsNamesMapping       []eventNameMapping             `mapstructure:"events_to_conversions_names_mapping" validate:"omitempty,dive"`
	CustomVariables                       []eventNameMapping             `mapstructure:"custom_variables" validate:"omitempty,dive"`
	UserIdentifierSource                  string                         `mapstructure:"user_identifier_source" validate:"omitempty,oneof=none UNSPECIFIED UNKNOWN FIRST_PARTY THIRD_PARTY" default:"none"`
	ConversionEnvironment                 string                         `mapstructure:"conversion_environment" validate:"omitempty,oneof=none UNSPECIFIED UNKNOWN APP WEB" default:"none"`
	DefaultUserIdentifier                 string                         `mapstructure:"default_user_identifier" validate:"omitempty,oneof=email phone" default:"email"`
	HashUserIdentifier                    *bool                          `mapstructure:"hash_user_identifier" default:"true"`
	ValidateOnly                          *bool                          `mapstructure:"validate_only" default:"false"`
	ConnectionMode                        common.ConnectionMode          `mapstructure:"connection_mode"`
	ConsentManagement                     common.ConsentManagement       `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Ads Offline Conversions destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("customerId", "customer_id"),
		converter.Simple("subAccount", "sub_account"),
		converter.Simple("loginCustomerId", "login_customer_id"),
		converter.ArrayWithObjects("eventsToOfflineConversionsTypeMapping", "events_to_offline_conversions_type_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("eventsToConversionsNamesMapping", "events_to_conversions_names_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.ArrayWithObjects("customVariables", "custom_variables", map[string]any{
			"from": "from",
			"to":   "to",
		}),
		converter.Simple("UserIdentifierSource", "user_identifier_source"),
		converter.Simple("conversionEnvironment", "conversion_environment"),
		converter.Simple("defaultUserIdentifier", "default_user_identifier"),
		converter.Simple("hashUserIdentifier", "hash_user_identifier"),
		converter.Simple("validateOnly", "validate_only"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "google_adwords_offline_conversions",
		APIType:    "GOOGLE_ADWORDS_OFFLINE_CONVERSIONS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{},
		NewConfig: func() any {
			return &googleAdwordsOfflineConversionsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
