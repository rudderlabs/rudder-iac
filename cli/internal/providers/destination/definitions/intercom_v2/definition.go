package intercomv2

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/intercom_v2
// db-config.json. Upstream also lists amp, which the CLI never declares: it
// cannot produce that token, so a connection could never match it.
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
	common.SourceTypeWarehouse,
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
	common.SourceTypeWarehouse:     {"cloud"},
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff. See DEX-696 Discrepancy 3.
//
// intercomV2Config is the local YAML config model. The field set mirrors
// integrations-config destinations/intercom_v2 defaultConfig; validation
// mirrors schema.json.
type intercomV2Config struct {
	// The linked RudderStack account holding the Intercom access token.
	// schema.json declares it a required string with no pattern, so `required`
	// is the whole constraint.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required"`
	// Intercom regional API host. Upstream defaults it to US and also marks it
	// required; a required key can never take a default, so `required` wins and
	// the spec must name the region.
	APIServer string `mapstructure:"api_server" validate:"required,oneof=US Europe Australia"`
	// Whether RudderStack sends anonymousId for a user it cannot identify.
	SendAnonymousID   *bool                    `mapstructure:"send_anonymous_id" default:"false"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Intercom V2 destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("apiServer", "api_server"),
		converter.Simple("sendAnonymousId", "send_anonymous_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "intercom_v2",
		DisplayName: "Intercom V2",
		APIType:     "INTERCOM_V2",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{},
		NewConfig: func() any {
			return &intercomV2Config{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		// Mirror is the only behaviour upstream accepts, even though this
		// destination takes event stream sources too: those connect by source
		// type and never negotiate a sync behaviour.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
