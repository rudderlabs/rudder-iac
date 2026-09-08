package bingadsofflineconversions

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	funcs.NewPattern(
		"bingads_offline_conversions_numeric_id",
		`^[0-9]+$`,
		"must contain only digits",
	)
}

// This destination is warehouse-only, following the customerio_audience exception:
// warehouse is otherwise not reachable from event-stream sources, so keep it
// unverified until an account-linked warehouse apply flow can be proven live.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates legacy include-key consent blocks into consentManagement
// on write, so modelling them would create non-converging plans.
type bingAdsOfflineConversionsConfig struct {
	RudderAccountID   string                   `mapstructure:"rudder_account_id" validate:"required"`
	CustomerAccountID string                   `mapstructure:"customer_account_id" validate:"required,dynamic_or_pattern=bingads_offline_conversions_numeric_id"`
	CustomerID        string                   `mapstructure:"customer_id" validate:"required,dynamic_or_pattern=bingads_offline_conversions_numeric_id"`
	IsHashRequired    *bool                    `mapstructure:"is_hash_required" default:"false"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Bing Ads Offline Conversions destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("customerAccountId", "customer_account_id"),
		converter.Simple("customerId", "customer_id"),
		converter.Simple("isHashRequired", "is_hash_required"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "bingads_offline_conversions",
		APIType:    "BINGADS_OFFLINE_CONVERSIONS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{},
		NewConfig: func() any {
			return &bingAdsOfflineConversionsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
