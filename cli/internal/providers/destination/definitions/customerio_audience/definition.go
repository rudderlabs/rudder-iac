package customerioaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Audience destination is warehouse-only (integrations-config
// destinations/customerio_audience supportedSourceTypes).
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
	SiteID            string                   `mapstructure:"site_id" validate:"required,min=1,max=100"`
	APIKey            string                   `mapstructure:"api_key" validate:"required,min=1,max=100"`
	AppAPIKey         string                   `mapstructure:"app_api_key" validate:"required,min=1,max=100"`
	Region            string                   `mapstructure:"region" validate:"required,dynamic_or_oneof=US EU"`
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
