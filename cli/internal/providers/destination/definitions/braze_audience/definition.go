package brazeaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/braze_audience
// db-config.json: a warehouse-only destination, reached through rETL alone.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// brazeAudienceConfig is the local YAML config model. The field set mirrors
// integrations-config destinations/braze_audience defaultConfig; validation
// mirrors schema.json, which constrains rudderAccountId alone.
//
// consent_management is absent: db-config scopes only connectionMode to the
// warehouse source type, so the destination declares no consent surface.
type brazeAudienceConfig struct {
	// The linked RudderStack account holding the Braze REST API key. Upstream
	// constrains it to a bare `^.{1,100}$`, so templates are judged as literals
	// and `required` supplies the non-empty half.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required,pattern=single_line_100"`
	// rest_api_key and data_center predate account linking: they sit in
	// db-config defaultConfig, carry no schema.json entry and hence no
	// constraint. They stay modelled because destination update replaces the
	// whole config object.
	RestAPIKey     string                `mapstructure:"rest_api_key"`
	DataCenter     string                `mapstructure:"data_center"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// NewDefinition returns the Braze Audiences destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("restApiKey", "rest_api_key"),
		converter.Simple("dataCenter", "data_center"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "braze_audience",
		DisplayName: "Braze Audiences",
		APIType:     "BRAZE_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{"rest_api_key"},
		NewConfig: func() any {
			return &brazeAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
