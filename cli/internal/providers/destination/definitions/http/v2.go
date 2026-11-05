package http

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

func init() {
	funcs.NewPattern(
		"http_v2_max_batch_size",
		`^([1-9][0-9]{0,2}|[1-4][0-9]{3}|5000)$`,
		"must be a string integer from 1 to 5000",
	)
}

type httpV2Config struct {
	APIURL              string                   `mapstructure:"api_url" validate:"required,pattern=http_api_url"`
	Auth                string                   `mapstructure:"auth" validate:"required,oneof=noAuth basicAuth bearerTokenAuth apiKeyAuth"`
	Username            string                   `mapstructure:"username" validate:"required_if=Auth basicAuth,omitempty,pattern=http_single_line_1_100"`
	Password            string                   `mapstructure:"password" validate:"required_if=Auth basicAuth,omitempty,pattern=single_line_100"`
	BearerToken         string                   `mapstructure:"bearer_token" validate:"required_if=Auth bearerTokenAuth,omitempty,pattern=http_single_line_1_2048"`
	APIKeyName          string                   `mapstructure:"api_key_name" validate:"required_if=Auth apiKeyAuth,omitempty,pattern=http_api_key_name"`
	APIKeyValue         string                   `mapstructure:"api_key_value" validate:"required_if=Auth apiKeyAuth,omitempty,pattern=http_single_line_1_100"`
	Method              string                   `mapstructure:"method" validate:"required,oneof=POST PUT PATCH GET DELETE"`
	Format              string                   `mapstructure:"format" validate:"oneof=JSON XML FORM" default:"JSON"`
	XMLRootKey          string                   `mapstructure:"xml_root_key" validate:"omitempty,pattern=single_line_100"`
	RequestBodyTemplate string                   `mapstructure:"request_body_template" validate:"required"`
	QueryParams         []queryParam             `mapstructure:"query_params" validate:"omitempty,dive"`
	Headers             []header                 `mapstructure:"headers" validate:"omitempty,dive"`
	PathParams          []pathParam              `mapstructure:"path_params" validate:"omitempty,dive"`
	MaxBatchSize        string                   `mapstructure:"max_batch_size" validate:"required_unless=Format FORM,omitempty,pattern=http_v2_max_batch_size"`
	ConnectionMode      common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement   common.ConsentManagement `mapstructure:"consent_management"`
}

// NewV2Definition keeps HTTP v1's identity, source support and secret policy,
// but gives v2 its own config model and template-based request body mappings.
func NewV2Definition() *definitions.DestinationDefinition {
	def := NewDefinition()
	def.Version = 2
	def.NewConfig = func() any { return &httpV2Config{} }
	def.Properties = []converter.ConfigProperty{
		converter.Simple("apiUrl", "api_url"),
		converter.Simple("auth", "auth"),
		converter.Simple("username", "username"),
		converter.Simple("password", "password"),
		converter.Simple("bearerToken", "bearer_token"),
		converter.Simple("apiKeyName", "api_key_name"),
		converter.Simple("apiKeyValue", "api_key_value"),
		converter.Simple("method", "method"),
		converter.Simple("format", "format"),
		converter.Simple("xmlRootKey", "xml_root_key"),
		converter.Simple("requestBodyTemplate", "request_body_template"),
		converter.ArrayWithObjects("queryParams", "query_params", map[string]any{"to": "to", "from": "from"}),
		converter.ArrayWithObjects("headers", "headers", map[string]any{"to": "to", "from": "from"}),
		converter.ArrayWithObjects("pathParams", "path_params", map[string]any{"path": "path"}),
		converter.Simple("maxBatchSize", "max_batch_size"),
	}
	def.Properties = append(def.Properties, common.ConnectionModeProperties(sourceTypes)...)
	def.Properties = append(def.Properties, common.Properties(sourceTypes)...)
	return def
}
