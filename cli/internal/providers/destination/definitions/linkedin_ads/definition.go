package linkedinads

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/linkedIn_ads/db-config.json.
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
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

// linkedinAdsConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/linkedIn_ads defaultConfig; validation
// constraints mirror schema.json for overlapping terraform-mapped fields.
type linkedinAdsConfig struct {
	RudderAccountID   string                   `mapstructure:"rudder_account_id" validate:"required"`
	HashData          *bool                    `mapstructure:"hash_data" validate:"required"`
	AdAccountID       string                   `mapstructure:"ad_account_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	DeduplicationKey  string                   `mapstructure:"deduplication_key" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	ConversionMapping []conversionMapping      `mapstructure:"conversion_mapping" validate:"omitempty,dive"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// Upstream constrains both ends to `^(.{1,100})$`; `required` supplies the
// non-empty half so the shared single-line pattern covers the rest.
type conversionMapping struct {
	From string `mapstructure:"from" validate:"required,dynamic_or_pattern=single_line_100"`
	To   string `mapstructure:"to" validate:"required,dynamic_or_pattern=single_line_100"`
}

// NewDefinition returns the LinkedIn Ads destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("hashData", "hash_data"),
		converter.Simple("adAccountId", "ad_account_id"),
		converter.Simple("deduplicationKey", "deduplication_key"),
		converter.ArrayWithObjects("conversionMapping", "conversion_mapping", map[string]any{
			"from": "from",
			"to":   "to",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "linkedin_ads",
		APIType:    "LINKEDIN_ADS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{},
		NewConfig: func() any {
			return &linkedinAdsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
