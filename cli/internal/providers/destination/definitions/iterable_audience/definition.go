package iterableaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/iterable_audience
// db-config.json: a warehouse-only destination, reached through rETL alone.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// iterableAudienceConfig is the local YAML config model. The field set mirrors
// integrations-config destinations/iterable_audience defaultConfig; validation
// mirrors schema.json, which constrains rudderAccountId alone.
//
// consent_management is absent: db-config scopes only connectionMode to the
// warehouse source type, so the destination declares no consent surface.
type iterableAudienceConfig struct {
	// The linked RudderStack account holding the Iterable API key
	// (db-config supportedAccountDefinitions, accounts/iterable_audience_api_key).
	// Upstream constrains it to a bare `^.{1,100}$`, so templates are judged as
	// literals and `required` supplies the non-empty half.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required,pattern=single_line_100"`
	// api_key, data_center and project_type predate account linking: they sit in
	// db-config defaultConfig, carry no schema.json entry and hence no
	// constraint, and the dashboard no longer shows them. They stay modelled
	// because destination update replaces the whole config object.
	APIKey         string                `mapstructure:"api_key"`
	DataCenter     string                `mapstructure:"data_center"`
	ProjectType    string                `mapstructure:"project_type"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// NewDefinition returns the Iterable Audience destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("apiKey", "api_key"),
		converter.Simple("dataCenter", "data_center"),
		converter.Simple("projectType", "project_type"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "iterable_audience",
		DisplayName: "Iterable Audience",
		APIType:     "ITERABLE_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{"api_key"},
		NewConfig: func() any {
			return &iterableAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
