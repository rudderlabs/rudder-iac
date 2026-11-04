package googleadwordsremarketinglists

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config
// destinations/google_adwords_remarketing_lists db-config.json. Upstream also
// lists shopify, which the CLI never declares: it cannot produce that token, so
// a connection could never match it.
var sourceTypes = []string{
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeCloud:     {"cloud"},
	common.SourceTypeWarehouse: {"cloud"},
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff. See DEX-696 Discrepancy 3.
//
// connection_mode is modelled even though schema.json declares no
// connectionMode property, because db-config's destConfig names it under every
// source type and the fleet models it everywhere on that basis — firebase is
// the established precedent for the same shape. Its legal values still come
// from this definition's own ConnectionModes map, so there is something to
// validate against.
//
// googleAdwordsRemarketingListsConfig is the local YAML config model. The field
// set mirrors integrations-config defaultConfig; validation mirrors schema.json.
type googleAdwordsRemarketingListsConfig struct {
	// The linked RudderStack account holding the Google Ads OAuth grant
	// (db-config supportedAccountDefinitions). schema.json declares it a
	// required string with no pattern, so `required` is the whole constraint.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required"`
	CustomerID      string `mapstructure:"customer_id" validate:"required,dynamic_or_pattern=single_line_100"`
	// Upstream marks subAccount required and also defaults it to false. A
	// required key can never take a default, so `required` wins and the spec
	// must say which it is. A pointer keeps absent distinct from false.
	SubAccount *bool `mapstructure:"sub_account" validate:"required"`
	// Required once sub_account is true (schema.json allOf).
	LoginCustomerID string `mapstructure:"login_customer_id" validate:"required_if=SubAccount true,omitempty,dynamic_or_pattern=single_line_100"`
	// listId has no schema.json entry, so it carries no constraint. It stays
	// modelled because destination update replaces the whole config object.
	ListID string `mapstructure:"list_id"`
	// Required at connect time for a cloud source (schema.json allOf).
	AudienceID string `mapstructure:"audience_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	// Identifiers Google matches on. Upstream defaults this to ["email"],
	// which ApplyDefaults cannot carry: it handles top-level scalars only.
	UserSchema             []string                 `mapstructure:"user_schema" validate:"omitempty,dive,oneof=email phone addressInfo"`
	IsHashRequired         *bool                    `mapstructure:"is_hash_required" default:"true"`
	TypeOfList             string                   `mapstructure:"type_of_list" validate:"omitempty,oneof=General userID mobileDeviceID" default:"General"`
	UserDataConsent        string                   `mapstructure:"user_data_consent" validate:"omitempty,oneof=UNSPECIFIED UNKNOWN GRANTED DENIED" default:"UNSPECIFIED"`
	PersonalizationConsent string                   `mapstructure:"personalization_consent" validate:"omitempty,oneof=UNSPECIFIED UNKNOWN GRANTED DENIED" default:"UNSPECIFIED"`
	ConnectionMode         common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement      common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Google Ads Remarketing Lists destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("customerId", "customer_id"),
		converter.Simple("subAccount", "sub_account"),
		converter.Simple("loginCustomerId", "login_customer_id"),
		converter.Simple("listId", "list_id"),
		converter.Simple("audienceId", "audience_id"),
		converter.Simple("userSchema", "user_schema"),
		converter.Simple("isHashRequired", "is_hash_required"),
		converter.Simple("typeOfList", "type_of_list"),
		converter.Simple("userDataConsent", "user_data_consent"),
		converter.Simple("personalizationConsent", "personalization_consent"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "google_adwords_remarketing_lists",
		DisplayName: "Google Ads Remarketing Lists (Customer Match)",
		APIType:     "GOOGLE_ADWORDS_REMARKETING_LISTS",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{},
		NewConfig: func() any {
			return &googleAdwordsRemarketingListsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// The one connectionMode-conditioned allOf branch in schema.json names
		// cloud only; a warehouse (rETL) source needs no extra key.
		ConnectionRequiredKeys: map[string]map[string][]string{
			common.SourceTypeCloud: {"cloud": {"audience_id"}},
		},
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
