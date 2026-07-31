package http

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	// apiURLPatternTag validates the base endpoint URL. Only domain-name URLs
	// over http/https are accepted; ngrok and localhost hosts are rejected to
	// reduce misconfiguration risk (ported from terraform-provider-rudderstack,
	// which splits the source schema's single lookahead pattern into a positive
	// match plus reject patterns because Go's RE2 has no negative lookahead).
	apiURLPatternTag     = "http_api_url"
	apiURLPattern        = `^(https?://)([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,}(:(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5]\d{4}|[1-9]\d{1,3}))?(/.*)?$`
	apiURLRejectPattern  = `^https?://(?:[^/]*\.ngrok\.io|localhost[^/:]*|[^/]*\.localhost[^/:]*)(?:[:/].*)?$`
	apiURLPatternMessage = "must be a valid http(s) URL with a domain name (ngrok and localhost hosts are not allowed)"

	// maxBatchSizePatternTag validates the batch size cap (1-100).
	maxBatchSizePatternTag     = "http_max_batch_size"
	maxBatchSizePattern        = `^([1-9][0-9]?|100)$`
	maxBatchSizePatternMessage = "must be a number between 1 and 100"
)

func init() {
	funcs.NewPatternWithReject(apiURLPatternTag, apiURLPattern, apiURLRejectPattern, apiURLPatternMessage)
	funcs.NewPattern(maxBatchSizePatternTag, maxBatchSizePattern, maxBatchSizePatternMessage)
}

// Source types from integrations-config destinations/http/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns
// (amp, shopify, warehouse dropped, matching the S3 precedent).
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeCloud,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
}

// keyValueMapping is the local shape for query params, headers, and properties
// mapping: a list of {to, from} pairs.
type keyValueMapping struct {
	To   string `mapstructure:"to" validate:"omitempty,max=100"`
	From string `mapstructure:"from" validate:"omitempty,max=100"`
}

// pathParam is the local shape for a single path parameter entry.
type pathParam struct {
	Path string `mapstructure:"path" validate:"omitempty,max=100"`
}

// eventFiltering holds the allowlist/denylist of event names. Exactly one side
// is populated; the eventFilteringOption discriminator is derived at conversion.
type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"omitempty,dive,max=100"`
	Blacklist []string `mapstructure:"blacklist" validate:"omitempty,dive,max=100"`
}

// httpConfig is the local YAML config model. Field set mirrors integrations-config
// destinations/http defaultConfig (schema.json / db-config.json), excluding
// connection_mode and the consent boilerplate handled by common. Validation
// constraints mirror the overlapping schema.json / terraform rules.
type httpConfig struct {
	APIURL      string `mapstructure:"api_url" validate:"required,pattern=http_api_url"`
	Auth        string `mapstructure:"auth" validate:"required,dynamic_or_oneof=noAuth basicAuth bearerTokenAuth apiKeyAuth"`
	Username    string `mapstructure:"username" validate:"omitempty,max=100"`
	Password    string `mapstructure:"password" validate:"omitempty,max=100"`
	BearerToken string `mapstructure:"bearer_token" validate:"omitempty,max=255"`
	APIKeyName  string `mapstructure:"api_key_name" validate:"omitempty,max=100"`
	APIKeyValue string `mapstructure:"api_key_value" validate:"omitempty,max=100"`
	XMLRootKey  string `mapstructure:"xml_root_key" validate:"omitempty,max=100"`
	Method      string `mapstructure:"method" validate:"required,dynamic_or_oneof=POST PUT PATCH GET DELETE"`
	Format      string `mapstructure:"format" validate:"required,dynamic_or_oneof=JSON XML FORM"`

	PropertiesMapping []keyValueMapping `mapstructure:"properties_mapping" validate:"omitempty,dive"`
	QueryParams       []keyValueMapping `mapstructure:"query_params" validate:"omitempty,dive"`
	Headers           []keyValueMapping `mapstructure:"headers" validate:"omitempty,dive"`
	PathParams        []pathParam       `mapstructure:"path_params" validate:"omitempty,dive"`

	IsBatchingEnabled *bool  `mapstructure:"is_batching_enabled"`
	MaxBatchSize      string `mapstructure:"max_batch_size" validate:"omitempty,pattern=http_max_batch_size"`

	EventFiltering   *eventFiltering `mapstructure:"event_filtering" validate:"omitempty"`
	IsDefaultMapping *bool           `mapstructure:"is_default_mapping"`

	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

// NewDefinition returns the HTTP Webhook destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	keyValueFields := map[string]any{
		"to":   "to",
		"from": "from",
	}

	properties := []converter.ConfigProperty{
		converter.Simple("apiUrl", "api_url"),
		converter.Simple("auth", "auth"),
		converter.Simple("username", "username"),
		converter.Simple("password", "password"),
		converter.Simple("bearerToken", "bearer_token"),
		converter.Simple("apiKeyName", "api_key_name"),
		converter.Simple("apiKeyValue", "api_key_value"),
		converter.Simple("xmlRootKey", "xml_root_key"),
		converter.Simple("method", "method"),
		converter.Simple("format", "format"),
		converter.ArrayWithObjects("propertiesMapping", "properties_mapping", keyValueFields),
		converter.ArrayWithObjects("queryParams", "query_params", keyValueFields),
		converter.ArrayWithObjects("headers", "headers", keyValueFields),
		converter.ArrayWithObjects("pathParams", "path_params", map[string]any{
			"path": "path",
		}),
		converter.Simple("isBatchingEnabled", "is_batching_enabled"),
		converter.Simple("maxBatchSize", "max_batch_size"),
		converter.ArrayWithStrings("whitelistedEvents", "eventName", "event_filtering.whitelist"),
		converter.ArrayWithStrings("blacklistedEvents", "eventName", "event_filtering.blacklist"),
		converter.Discriminator("eventFilteringOption", converter.DiscriminatorValues{
			"event_filtering.whitelist": "whitelistedEvents",
			"event_filtering.blacklist": "blacklistedEvents",
		}),
		converter.Simple("isDefaultMapping", "is_default_mapping"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "http",
		APIType:    "HTTP",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"password", "bearer_token", "api_key_value"},
		NewConfig: func() any {
			return &httpConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
