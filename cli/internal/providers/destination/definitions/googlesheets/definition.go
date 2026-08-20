package googlesheets

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/googlesheets/db-config.json.
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

// googleSheetsConfig is the local YAML config model. Field set mirrors the
// integrations-config destinations/googlesheets defaultConfig; validation
// constraints mirror schema.json, with required nested event mappings from the
// Google Sheets UI/Terraform contract.
type googleSheetsConfig struct {
	Credentials       string                   `mapstructure:"credentials" validate:"required"`
	SheetID           string                   `mapstructure:"sheet_id" validate:"required"`
	SheetName         string                   `mapstructure:"sheet_name" validate:"required"`
	EventKeyMap       []eventKeyMapping        `mapstructure:"event_key_map" validate:"required,dive"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// Terraform marks both fields Required and ui-config marks the mapping form
// required, but schema.json declares them as plain strings with no constraint.
// Validation follows schema.json, so nothing is enforced here — otherwise a
// remote config holding a partially filled mapping row would import to a spec
// the CLI rejects. Mirrors the marketo definition.
type eventKeyMapping struct {
	From string `mapstructure:"from"`
	To   string `mapstructure:"to"`
}

// NewDefinition returns the Google Sheets destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("credentials", "credentials"),
		converter.Simple("sheetId", "sheet_id"),
		converter.Simple("sheetName", "sheet_name"),
		converter.ArrayWithObjects("eventKeyMap", "event_key_map", map[string]any{
			"from": "from",
			"to":   "to",
		}),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "googlesheets",
		APIType:    "GOOGLESHEETS",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"credentials"},
		NewConfig: func() any {
			return &googleSheetsConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
