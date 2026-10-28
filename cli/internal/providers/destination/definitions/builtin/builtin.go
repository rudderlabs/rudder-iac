// Package builtin lists the destination definitions compiled into rudder-cli
// and whether each is verified. The registry the providers use and the docs
// command both read this one table, so they cannot disagree on what is
// registered or verified.
package builtin

import (
	"fmt"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	activecampaign "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/active_campaign"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/adj"
	adobeanalytics "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/adobe_analytics"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	attentivetag "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/attentive_tag"
	bingadsofflineconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bingads_offline_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bq"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/bqstream"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	confluentcloud "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/confluent_cloud"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio"
	customerioaudience "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/customerio_audience"
	facebookconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_conversions"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/firebase"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/ga"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/ga4"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/gcs"
	googleadwordsofflineconversions "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/google_adwords_offline_conversions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googleads"
	googlepubsub "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlepubsub"
	googlesheets "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlesheets"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/gtm"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/hs"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/intercom"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/iterable"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/kafka"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/kinesis"
	linkedinads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/linkedin_ads"
	linkedininsighttag "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/linkedin_insight_tag"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/marketo"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/mp"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/postgres"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/posthog"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/qualtrics"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/redis"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/rs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3"
	s3datalake "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3_datalake"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/salesforce"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/sentry"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/slack"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowflake"
	snowpipestreaming "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/snowpipe_streaming"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/statsig"
	tiktokads "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/tiktok_ads"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/vwo"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/webhook"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/zendesk"
)

// entry is one compiled destination definition.
type entry struct {
	NewDefinition func() *definitions.DestinationDefinition
	// Verified definitions always register. The others register only behind
	// the unverifiedDestinations experimental flag.
	Verified bool
}

// entries are in registration order: verified first, then unverified. Every
// newly onboarded definition starts unverified.
var entries = []entry{
	{NewDefinition: bqstream.NewDefinition, Verified: true},
	{NewDefinition: s3.NewDefinition, Verified: true},
	{NewDefinition: httpdest.NewDefinition, Verified: true},
	{NewDefinition: activecampaign.NewDefinition, Verified: true},
	{NewDefinition: attentivetag.NewDefinition, Verified: true},
	{NewDefinition: bq.NewDefinition, Verified: true},
	{NewDefinition: am.NewDefinition, Verified: true},
	{NewDefinition: braze.NewDefinition, Verified: true},
	{NewDefinition: customerio.NewDefinition, Verified: true},
	{NewDefinition: facebookconversions.NewDefinition, Verified: true},
	{NewDefinition: facebookpixel.NewDefinition, Verified: true},
	{NewDefinition: ga4.NewDefinition, Verified: true},
	{NewDefinition: gcs.NewDefinition, Verified: true},
	{NewDefinition: googleads.NewDefinition, Verified: true},
	{NewDefinition: hs.NewDefinition, Verified: true},
	{NewDefinition: iterable.NewDefinition, Verified: true},
	{NewDefinition: mp.NewDefinition, Verified: true},
	{NewDefinition: postgres.NewDefinition, Verified: true},
	{NewDefinition: posthog.NewDefinition, Verified: true},
	{NewDefinition: rs.NewDefinition, Verified: true},
	{NewDefinition: s3datalake.NewDefinition, Verified: true},
	{NewDefinition: snowflake.NewDefinition, Verified: true},
	{NewDefinition: tiktokads.NewDefinition, Verified: true},
	{NewDefinition: webhook.NewDefinition, Verified: true},
	{NewDefinition: adj.NewDefinition, Verified: true},
	{NewDefinition: adobeanalytics.NewDefinition, Verified: true},
	{NewDefinition: confluentcloud.NewDefinition, Verified: true},
	{NewDefinition: firebase.NewDefinition, Verified: true},
	{NewDefinition: ga.NewDefinition, Verified: true},
	{NewDefinition: googlepubsub.NewDefinition, Verified: true},
	{NewDefinition: googlesheets.NewDefinition, Verified: true},
	{NewDefinition: gtm.NewDefinition, Verified: true},
	{NewDefinition: intercom.NewDefinition, Verified: true},
	{NewDefinition: kinesis.NewDefinition, Verified: true},
	{NewDefinition: kafka.NewDefinition, Verified: true},
	{NewDefinition: linkedininsighttag.NewDefinition, Verified: true},
	{NewDefinition: marketo.NewDefinition, Verified: true},
	{NewDefinition: qualtrics.NewDefinition, Verified: true},
	{NewDefinition: redis.NewDefinition, Verified: true},
	{NewDefinition: salesforce.NewDefinition, Verified: true},
	{NewDefinition: sentry.NewDefinition, Verified: true},
	{NewDefinition: slack.NewDefinition, Verified: true},
	{NewDefinition: snowpipestreaming.NewDefinition, Verified: true},
	{NewDefinition: statsig.NewDefinition, Verified: true},
	{NewDefinition: vwo.NewDefinition, Verified: true},
	{NewDefinition: zendesk.NewDefinition, Verified: true},
	{NewDefinition: bingadsofflineconversions.NewDefinition},
	{NewDefinition: customerioaudience.NewDefinition},
	{NewDefinition: googleadwordsofflineconversions.NewDefinition},
	{NewDefinition: linkedinads.NewDefinition},
}

// NewRegistry registers the verified definitions and, with includeUnverified,
// the unverified ones too.
func NewRegistry(includeUnverified bool) (*definitions.Registry, error) {
	registry := definitions.NewRegistry()
	for _, entry := range entries {
		if !entry.Verified && !includeUnverified {
			continue
		}
		def := entry.NewDefinition()
		if err := registry.Register(def); err != nil {
			return nil, fmt.Errorf("registering %s destination definition: %w", def.Type, err)
		}
	}
	return registry, nil
}

// IsVerified reports whether the table marks that definition verified.
func IsVerified(destType string, version int64) bool {
	for _, entry := range entries {
		if def := entry.NewDefinition(); def.Type == destType && def.Version == version {
			return entry.Verified
		}
	}
	return false
}
