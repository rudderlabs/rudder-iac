package linkedinaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/linkedin_audience
// db-config.json: a warehouse-only destination, reached through rETL alone.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// linkedinAudienceConfig is the local YAML config model. Upstream declares the
// smallest config surface of any vdm-next destination: the linked account and
// nothing else. Everything the sync needs comes from that account and from the
// connection's own field mappings.
//
// connection_mode is modelled although schema.json declares no connectionMode
// property, because db-config names it under the warehouse source type and the
// fleet models it everywhere on that basis. Its legal values come from this
// definition's own ConnectionModes map.
//
// consent_management is absent: db-config scopes only connectionMode to the
// warehouse source type, so the destination declares no consent surface.
type linkedinAudienceConfig struct {
	// The linked RudderStack account holding the LinkedIn OAuth grant.
	// schema.json declares it a required string with no pattern, so `required`
	// is the whole constraint.
	RudderAccountID string                `mapstructure:"rudder_account_id" validate:"required"`
	ConnectionMode  common.ConnectionMode `mapstructure:"connection_mode"`
}

// NewDefinition returns the LinkedIn Audience destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "linkedin_audience",
		DisplayName: "Linkedin Audience",
		APIType:     "LINKEDIN_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{},
		NewConfig: func() any {
			return &linkedinAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
