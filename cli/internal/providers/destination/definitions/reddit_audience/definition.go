package redditaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/reddit_audience
// db-config.json: a warehouse-only destination, reached through rETL alone.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// redditAudienceConfig is the local YAML config model. The field set mirrors
// integrations-config destinations/reddit_audience defaultConfig; validation
// mirrors schema.json.
//
// consent_management is absent: db-config scopes only connectionMode to the
// warehouse source type, so the destination declares no consent surface.
type redditAudienceConfig struct {
	// The linked RudderStack account holding the Reddit Ads OAuth grant.
	// Upstream constrains it to a bare `^.{1,100}$`, so templates are judged as
	// literals and `required` supplies the non-empty half.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required,pattern=single_line_100"`
	// Reddit ad account the audience belongs to. Upstream wraps the constraint
	// in a {{ … || … }} alternative, so a template is accepted here.
	AdAccountID    string                `mapstructure:"ad_account_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// NewDefinition returns the Reddit Audience destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("adAccountId", "ad_account_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "reddit_audience",
		DisplayName: "Reddit Audience",
		APIType:     "REDDIT_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{},
		NewConfig: func() any {
			return &redditAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
