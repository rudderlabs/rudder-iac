package customerioaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Audience destination is warehouse-only (integrations-config
// destinations/customerio_audience supportedSourceTypes), which makes it the
// one definition that may declare warehouse: the CLI cannot produce that source
// token yet, so no connection can exercise this destination end to end. That is
// also why it stays in the unverified registry — promote it to the verified
// block once warehouse sources are supported and the apply cycle can be proven
// against a live stack.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// customerioAudienceConfig is the local YAML config model. Field set mirrors
// integrations-config destinations/customerio_audience defaultConfig;
// validation constraints mirror overlapping schema.json rules.
type customerioAudienceConfig struct {
	SiteID            string                   `mapstructure:"site_id" validate:"required,pattern=single_line_100"`
	APIKey            string                   `mapstructure:"api_key" validate:"required,pattern=single_line_100"`
	AppAPIKey         string                   `mapstructure:"app_api_key" validate:"required,pattern=single_line_100"`
	Region            string                   `mapstructure:"region" validate:"required,oneof=US EU"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Customer.io Audience destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("siteId", "site_id"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("appApiKey", "app_api_key"),
		converter.Simple("region", "region"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "customerio_audience",
		APIType:    "CUSTOMERIO_AUDIENCE",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"app_api_key", "api_key"},
		NewConfig: func() any {
			return &customerioAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
