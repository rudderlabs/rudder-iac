package fbcustomaudience

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// Source types from integrations-config destinations/fb_custom_audience
// db-config.json. Upstream also lists shopify, which the CLI never declares:
// it cannot produce that token, so a connection could never match it.
var sourceTypes = []string{
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
}

var connectionModes = map[string][]string{
	common.SourceTypeCloud:     {"cloud"},
	common.SourceTypeWarehouse: {"cloud"},
}

// oneTrustCookieCategories and ketchConsentPurposes are deliberately absent:
// the backend migrates them into consentManagement on write and never returns
// them, so modelling them makes every plan diff. See DEX-696 Discrepancy 3.
//
// fbCustomAudienceConfig is the local YAML config model. The field set mirrors
// integrations-config destinations/fb_custom_audience defaultConfig; validation
// mirrors schema.json.
type fbCustomAudienceConfig struct {
	// The linked RudderStack account holding the Facebook access token
	// (db-config supportedAccountDefinitions). Upstream constrains it to
	// `^.{1,100}$` with no template branch, so templates are rejected and
	// `required` supplies the non-empty half.
	RudderAccountID string `mapstructure:"rudder_account_id" validate:"required,pattern=single_line_100"`
	// accessToken and appSecret predate account linking and carry no
	// schema.json entry, hence no constraint beyond their secrecy. They stay
	// modelled because destination update replaces the whole config object.
	AccessToken string `mapstructure:"access_token"`
	AppSecret   string `mapstructure:"app_secret"`
	// Identifiers Facebook matches on. Upstream defaults this to ["EMAIL"],
	// which ApplyDefaults cannot carry: it handles top-level scalars only.
	UserSchema     []string `mapstructure:"user_schema" validate:"omitempty,dive,oneof=EMAIL PHONE GEN MADID EXTERN_ID DOBY DOBM DOBD LN FN FI CT ST ZIP COUNTRY"`
	IsHashRequired *bool    `mapstructure:"is_hash_required" default:"true"`
	IsRaw          *bool    `mapstructure:"is_raw" default:"false"`
	DisableFormat  *bool    `mapstructure:"disable_format" default:"false"`
	Type           string   `mapstructure:"type" validate:"omitempty,oneof=UNKNOWN FILE_IMPORTED EVENT_BASED SEED_BASED THIRD_PARTY_IMPORTED COPY_PASTE CONTACT_IMPORTER HOUSEHOLD_AUDIENCE NA" default:"NA"`
	SubType        string   `mapstructure:"sub_type" validate:"omitempty,oneof=ANYTHING NOTHING HASHES USER_IDS HASHES_OR_USER_IDS MOBILE_ADVERTISER_IDS FB_EVENT_SIGNALS EXTERNAL_IDS MULTI_HASHES TOKENS EXTERNAL_IDS_MIX WEB_PIXEL_HITS MOBILE_APP_EVENTS MOBILE_APP_COMBINATION_EVENTS VIDEO_EVENTS WEB_PIXEL_COMBINATION_EVENTS IG_BUSINESS_EVENTS MULTI_DATA_EVENTS STORE_VISIT_EVENTS INSTANT_ARTICLE_EVENTS ENGAGEMENT_EVENT_USERS FACEBOOK_WIFI_EVENTS CUSTOM_AUDIENCE_USERS S_EXPR DYNAMIC_RULE CONVERSION_PIXEL_HITS APP_USERS CAMPAIGN_CONVERSIONS WEB_PIXEL_HITS_CUSTOM_AUDIENCE_USERS MOBILE_APP_CUSTOM_AUDIENCE_USERS VIDEO_EVENT_USERS FB_PIXEL_HITS IG_PROMOTED_POST PLACE_VISITS OFFLINE_EVENT_USERS EXPANDED_AUDIENCE SEED_LIST PARTNER_CATEGORY_USERS PAGE_SMART_AUDIENCE MULTICOUNTRY_COMBINATION PLATFORM_USERS MULTI_EVENT_SOURCE SMART_AUDIENCE LOOKALIKE_PLATFORM SIGNAL_SOURCE MAIL_CHIMP_EMAIL_HASHES CONSTANT_CONTACTS_EMAIL_HASHES COPY_PASTE_EMAIL_HASHES CONTACT_IMPORTER DATA_FILE NA" default:"NA"`
	// Required at connect time rather than unconditionally: ad_account_id for a
	// warehouse (rETL) source, audience_id for a cloud one. See
	// ConnectionRequiredKeys below.
	AdAccountID       string                   `mapstructure:"ad_account_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	AudienceID        string                   `mapstructure:"audience_id" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the Facebook Custom Audience destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("rudderAccountId", "rudder_account_id"),
		converter.Simple("accessToken", "access_token"),
		converter.Simple("appSecret", "app_secret"),
		converter.Simple("userSchema", "user_schema"),
		converter.Simple("isHashRequired", "is_hash_required"),
		converter.Simple("isRaw", "is_raw"),
		converter.Simple("disableFormat", "disable_format"),
		converter.Simple("type", "type"),
		converter.Simple("subType", "sub_type"),
		converter.Simple("adAccountId", "ad_account_id"),
		converter.Simple("audienceId", "audience_id"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:        "fb_custom_audience",
		DisplayName: "Facebook Custom Audience",
		APIType:     "FB_CUSTOM_AUDIENCE",
		Version:     1,
		Properties:  properties,
		SecretKeys:  []string{"access_token", "app_secret"},
		NewConfig: func() any {
			return &fbCustomAudienceConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
		// From the connectionMode-conditioned allOf branches in schema.json.
		ConnectionRequiredKeys: map[string]map[string][]string{
			common.SourceTypeCloud:     {"cloud": {"audience_id"}},
			common.SourceTypeWarehouse: {"cloud": {"ad_account_id"}},
		},
		// db-config.json config.syncBehaviours and config.supportsVisualMapperV2.
		// Mirror is the only behaviour upstream accepts, and it is valid only on
		// the visual-mapper-v2 flow this flag selects.
		SyncBehaviours:         []string{"mirror"},
		SupportsVisualMapperV2: true,
	}
}
