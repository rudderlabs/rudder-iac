package tiktokaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/tiktok_audience
// db-config.json: a warehouse-only destination, reached through rETL alone.
var sourceTypes = []string{
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeWarehouse: {"cloud"},
}

// tiktokAudienceConfig is the local YAML config model. The field set is the
// union of integrations-config destinations/tiktok_audience defaultConfig, its
// warehouse destConfig list and schema.json; validation mirrors schema.json,
// which constrains rudderAccountId alone.
//
// consent_management is absent: db-config scopes only adAccountId and
// connectionMode to the warehouse source type, so the destination declares no
// consent surface.
type tiktokAudienceConfig struct {
	// The linked RudderStack account holding the TikTok Business OAuth grant.
	// schema.json declares it a required string with no pattern, so `required`
	// is the whole constraint.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required"`
	// advertiser_id sits in defaultConfig with no schema.json entry, so it
	// carries no constraint. It stays modelled because destination update
	// replaces the whole config object.
	AdvertiserID string `mapstructure:"advertiser_id"`
	// ad_account_id is listed under destConfig.warehouse rather than
	// defaultConfig, so it is gated to that source type. The gate covers every
	// source type this destination has, which is the point: it documents that
	// the key is warehouse-scoped upstream rather than a default key.
	AdAccountID string `mapstructure:"ad_account_id"`
	// audience_id appears only in schema.json's anyOf branch, in a `then` that
	// lists it under properties without requiring it, so it is available
	// rather than required. It is in no destConfig list at all — flagged
	// upstream discrepancy, modelled so an apply cannot erase it.
	AudienceID     string                `mapstructure:"audience_id"`
	ConnectionMode common.ConnectionMode `mapstructure:"connection_mode"`
}

// NewDefinition returns the TikTok Audience destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("advertiserId", "advertiser_id"),
		converter.Gated(
			converter.Simple("adAccountId", "ad_account_id"),
			common.SourceTypeWarehouse,
		),
		converter.Simple("audienceId", "audience_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "tiktok_audience",
		DisplayName: "Tiktok Audience",
		APIType:     "TIKTOK_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{},
		NewConfig: func() any {
			return &tiktokAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
