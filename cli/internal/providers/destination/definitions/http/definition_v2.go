package http

import (
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

const httpV2MaxBatchSizePattern = `^([1-9][0-9]{0,2}|[1-4][0-9]{3}|5000)$`

func init() {
	funcs.NewPatternWithReject("http_v2_api_url", `^(?:\$[\s\S]+|`+httpAPIURLPattern[1:len(httpAPIURLPattern)-1]+`)$`, httpAPIURLRejectPattern,
		"must be a public http(s) domain URL or a JSONata expression")
	funcs.NewPattern("http_v2_max_batch_size", httpV2MaxBatchSizePattern, "must be a string integer from 1 to 5000")
	funcs.NewPattern("http_v2_header_name", `^(?:|`+httpHeaderNameSegmentPattern+`)$`, "must be a literal HTTP header name up to 100 characters")
	funcs.NewPattern("http_v2_query_param_name", `^(?:|`+httpValueSegmentWithSpacePattern+`)$`, "must be a literal query parameter name up to 100 characters")
}

type httpV2Config struct {
	APIURL                string                   `mapstructure:"api_url" validate:"required,pattern=http_v2_api_url"`
	Auth                  string                   `mapstructure:"auth" validate:"required,oneof=noAuth basicAuth bearerTokenAuth apiKeyAuth"`
	Username              string                   `mapstructure:"username" validate:"required_if=Auth basicAuth,omitempty,pattern=http_single_line_1_100"`
	Password              string                   `mapstructure:"password" validate:"required_if=Auth basicAuth,omitempty,pattern=single_line_100"`
	BearerToken           string                   `mapstructure:"bearer_token" validate:"required_if=Auth bearerTokenAuth,omitempty,pattern=http_single_line_1_2048"`
	APIKeyName            string                   `mapstructure:"api_key_name" validate:"required_if=Auth apiKeyAuth,omitempty,pattern=http_api_key_name"`
	APIKeyValue           string                   `mapstructure:"api_key_value" validate:"required_if=Auth apiKeyAuth,omitempty,pattern=http_single_line_1_100"`
	Method                string                   `mapstructure:"method" validate:"required,oneof=POST PUT PATCH GET DELETE"`
	Format                string                   `mapstructure:"format" validate:"required,oneof=JSON"`
	RequestBodyTemplate   string                   `mapstructure:"request_body_template" validate:"required"`
	SupportedMessageTypes []string                 `mapstructure:"supported_message_types" validate:"http_v2_unique_message_types,dive,oneof=track identify page screen group alias"`
	QueryParams           []httpV2RequestField     `mapstructure:"query_params" validate:"omitempty,dive"`
	Headers               []httpV2Header           `mapstructure:"headers" validate:"omitempty,dive"`
	PathParams            []httpV2PathParam        `mapstructure:"path_params" validate:"omitempty,dive"`
	IsBatchingEnabled     *bool                    `mapstructure:"is_batching_enabled" default:"false"`
	MaxBatchSize          string                   `mapstructure:"max_batch_size" validate:"required_if=IsBatchingEnabled true,omitempty,pattern=http_v2_max_batch_size"`
	EventFiltering        *eventFiltering          `mapstructure:"event_filtering"`
	ConnectionMode        common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement     common.ConsentManagement `mapstructure:"consent_management"`
}

type httpV2RequestField struct {
	To   string `mapstructure:"to" validate:"pattern=http_v2_query_param_name"`
	From string `mapstructure:"from"`
}

type httpV2Header struct {
	To   string `mapstructure:"to" validate:"pattern=http_v2_header_name"`
	From string `mapstructure:"from"`
}

type httpV2PathParam struct {
	Path string `mapstructure:"path"`
}

func uniqueMessageTypes(fl validator.FieldLevel) bool {
	values := fl.Field().Interface().([]string)
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

// NewDefinitionV2 keeps the v1 conversion and metadata for shared settings,
// but has its own config model and version-specific request fields.
func NewDefinitionV2() *definitions.DestinationDefinition {
	def := NewDefinition()
	def.Version = 2
	def.Properties = slices.DeleteFunc(def.Properties, func(p converter.ConfigProperty) bool {
		return slices.Contains([]string{"xml_root_key", "properties_mapping", "is_default_mapping"}, p.LocalKey)
	})
	def.Properties = append(def.Properties,
		converter.Simple("requestBodyTemplate", "request_body_template"),
		converter.Simple("supportedMessageTypes", "supported_message_types"),
	)
	def.NewConfig = func() any { return &httpV2Config{} }
	def.ConfigValidateFuncs = []rules.CustomValidateFunc{{Tag: "http_v2_unique_message_types", Func: uniqueMessageTypes}}
	return def
}
