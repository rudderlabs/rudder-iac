package http

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	httpAPIURLPattern       = `^(https?://)([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,}(:(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5]\d{4}|[1-9]\d{1,3}))?(/.*)?$`
	httpAPIURLRejectPattern = `^https?://([^/]*\.ngrok\.io|localhost[^/:]*|[^/]*\.localhost[^/:]*)(?:[:/].*)?$`

	httpJSONPathPattern              = `\$(?:\.|(?:\.\.(\w+|\*)|\.(\w+|\*)|\[\d+\]|\[('\w+'|"\w+")\]|\[\*\]|\.\w+\(\))*)`
	httpValueSegmentPattern          = "[A-Za-z0-9!#%&'*+.^_`|~-][A-Za-z0-9!#$%&'*+.^_`|~-]{0,99}"
	httpValueSegmentWithSpacePattern = "[A-Za-z0-9!#%&'*+.^_`|~\\- ][A-Za-z0-9!#$%&'*+.^_`|~\\- ]{0,99}"
	httpHeaderNameSegmentPattern     = "[A-Za-z0-9!#%&'*+.^_`|~-][A-Za-z0-9!#%&'*+.^_`|~-]{0,99}"
	httpHeaderValueSegmentPattern    = "[A-Za-z0-9!#%&'*+.^_`|~\\- /\\\\][A-Za-z0-9!#$%&'*+.^_`|~\\- /\\\\]{0,99}"
	httpPropertiesMappingToPattern   = `^(?:$|` + httpJSONPathPattern + `)$`
	httpPropertiesMappingFromPattern = `^(?:` + httpJSONPathPattern + `|` + httpValueSegmentPattern + `)$`
	httpQueryParamNamePattern        = `^(?:` + httpValueSegmentWithSpacePattern + `)$`
	httpQueryParamFromPattern        = `^(?:` + httpJSONPathPattern + `|` + httpValueSegmentWithSpacePattern + `)$`
	httpHeaderNamePattern            = `^(?:` + httpHeaderNameSegmentPattern + `)$`
	httpHeaderFromPattern            = `^(?:` + httpJSONPathPattern + `|` + httpHeaderValueSegmentPattern + `)$`
	httpPathParamPattern             = `^(?:` + httpJSONPathPattern + `/?|` + httpValueSegmentPattern + `/?)$`
	httpAuthHeaderNamePattern        = `^\S{1,100}$`
	httpMaxBatchSizePattern          = `^([1-9][0-9]{0,1}|100)$`
)

func init() {
	funcs.NewPatternWithReject(
		"http_api_url",
		httpAPIURLPattern,
		httpAPIURLRejectPattern,
		"must be a public http(s) domain URL and must not use localhost or ngrok",
	)
	funcs.NewPattern(
		"http_properties_mapping_to",
		httpPropertiesMappingToPattern,
		"must be empty or a JSONPath expression",
	)
	funcs.NewPattern(
		"http_properties_mapping_from",
		httpPropertiesMappingFromPattern,
		"must be a JSONPath expression or a constant token up to 100 characters",
	)
	funcs.NewPattern(
		"http_query_param_key",
		httpQueryParamNamePattern,
		"must be a non-JSONPath token up to 100 characters",
	)
	funcs.NewPattern(
		"http_query_param_from",
		httpQueryParamFromPattern,
		"must be a JSONPath expression or a constant token up to 100 characters",
	)
	funcs.NewPattern(
		"http_header_key",
		httpHeaderNamePattern,
		"must be a non-JSONPath token up to 100 characters",
	)
	funcs.NewPattern(
		"http_header_from",
		httpHeaderFromPattern,
		"must be a JSONPath expression or a constant token up to 100 characters",
	)
	funcs.NewPattern(
		"http_path_param",
		httpPathParamPattern,
		"must be a JSONPath expression or path token up to 100 characters",
	)
	funcs.NewPattern(
		"http_api_key_name",
		httpAuthHeaderNamePattern,
		"must be 1-100 non-whitespace characters",
	)
	funcs.NewPattern(
		"http_max_batch_size",
		httpMaxBatchSizePattern,
		"must be a string integer from 1 to 100",
	)
}

// Source types from integrations-config destinations/http/db-config.json
// supportedSourceTypes, restricted to types the CLI event-stream provider owns.
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

type httpConfig struct {
	APIURL            string                   `mapstructure:"api_url" validate:"required,pattern=http_api_url"`
	Auth              string                   `mapstructure:"auth" validate:"required,dynamic_or_oneof=noAuth basicAuth bearerTokenAuth apiKeyAuth"`
	Username          string                   `mapstructure:"username" validate:"required_if=Auth basicAuth,omitempty,min=1,max=100"`
	Password          string                   `mapstructure:"password" validate:"required_if=Auth basicAuth,omitempty,max=100"`
	BearerToken       string                   `mapstructure:"bearer_token" validate:"required_if=Auth bearerTokenAuth,omitempty,min=1,max=255"`
	APIKeyName        string                   `mapstructure:"api_key_name" validate:"required_if=Auth apiKeyAuth,omitempty,pattern=http_api_key_name"`
	APIKeyValue       string                   `mapstructure:"api_key_value" validate:"required_if=Auth apiKeyAuth,omitempty,min=1,max=100"`
	XMLRootKey        string                   `mapstructure:"xml_root_key" validate:"omitempty,max=100"`
	Method            string                   `mapstructure:"method" validate:"required,dynamic_or_oneof=POST PUT PATCH GET DELETE"`
	Format            string                   `mapstructure:"format" validate:"required,dynamic_or_oneof=JSON XML FORM"`
	PropertiesMapping []propertiesMapping      `mapstructure:"properties_mapping" validate:"omitempty,dive"`
	QueryParams       []queryParam             `mapstructure:"query_params" validate:"omitempty,dive"`
	Headers           []header                 `mapstructure:"headers" validate:"omitempty,dive"`
	PathParams        []pathParam              `mapstructure:"path_params" validate:"omitempty,dive"`
	IsBatchingEnabled *bool                    `mapstructure:"is_batching_enabled" default:"false"`
	MaxBatchSize      string                   `mapstructure:"max_batch_size" validate:"required_if=IsBatchingEnabled true,omitempty,pattern=http_max_batch_size"`
	EventFiltering    *eventFiltering          `mapstructure:"event_filtering"`
	IsDefaultMapping  *bool                    `mapstructure:"is_default_mapping" default:"true"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

type propertiesMapping struct {
	To   string `mapstructure:"to" validate:"omitempty,pattern=http_properties_mapping_to"`
	From string `mapstructure:"from" validate:"omitempty,pattern=http_properties_mapping_from"`
}

type queryParam struct {
	To   string `mapstructure:"to" validate:"omitempty,pattern=http_query_param_key"`
	From string `mapstructure:"from" validate:"omitempty,pattern=http_query_param_from"`
}

type header struct {
	To   string `mapstructure:"to" validate:"omitempty,pattern=http_header_key"`
	From string `mapstructure:"from" validate:"omitempty,pattern=http_header_from"`
}

type pathParam struct {
	Path string `mapstructure:"path" validate:"omitempty,pattern=http_path_param"`
}

type eventFiltering struct {
	Whitelist []string `mapstructure:"whitelist" validate:"excluded_with=Blacklist,dive,max=100"`
	Blacklist []string `mapstructure:"blacklist" validate:"excluded_with=Whitelist,dive,max=100"`
}

// NewDefinition returns the HTTP Webhook destination definition.
func NewDefinition() *definitions.DestinationDefinition {
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
		converter.ArrayWithObjects("propertiesMapping", "properties_mapping", map[string]any{
			"to":   "to",
			"from": "from",
		}),
		converter.ArrayWithObjects("queryParams", "query_params", map[string]any{
			"to":   "to",
			"from": "from",
		}),
		converter.ArrayWithObjects("headers", "headers", map[string]any{
			"to":   "to",
			"from": "from",
		}),
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
