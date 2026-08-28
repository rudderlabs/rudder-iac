package webhook

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	webhookURLPattern = `^(https?://)([a-zA-Z0-9-]{1,63}\.)+[a-zA-Z]{2,}(:(6553[0-5]|655[0-2][0-9]|65[0-4][0-9]{2}|6[0-4][0-9]{3}|[1-5]\d{4}|[1-9]\d{1,3}))?(/.*)?$`
	// RE2 has no lookahead, so schema.json's two negative lookaheads become a
	// reject pattern. Both are anchored to the position just after the scheme,
	// which is what makes this equivalent rather than merely similar: ngrok is
	// rejected only as the first label, while `.localhost` is rejected anywhere.
	webhookURLRejectPattern = `^https?://([a-zA-Z0-9-]*\.ngrok\.io|localhost|.*\.localhost)`
)

func init() {
	funcs.NewPatternWithReject(
		"webhook_url",
		webhookURLPattern,
		webhookURLRejectPattern,
		"must be a public http(s) domain URL and must not use localhost or ngrok",
	)
}

// Source types from integrations-config destinations/webhook/db-config.json.
var sourceTypes = []string{
	common.SourceTypeAndroid,
	common.SourceTypeAndroidKotlin,
	common.SourceTypeIOS,
	common.SourceTypeIOSSwift,
	common.SourceTypeWeb,
	common.SourceTypeUnity,
	common.SourceTypeAMP,
	common.SourceTypeCloud,
	common.SourceTypeWarehouse,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
	common.SourceTypeShopify,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeAMP:           {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeWarehouse:     {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
	common.SourceTypeShopify:       {"cloud"},
}

type webhookConfig struct {
	WebhookURL string `mapstructure:"webhook_url" validate:"required,dynamic_or_pattern=webhook_url"`
	// schema.json defaults webhookMethod to POST and the backend applies it on
	// persist, so the tag keeps a spec that omits the key from diffing.
	WebhookMethod     string                   `mapstructure:"webhook_method" validate:"omitempty,oneof=POST PUT PATCH GET DELETE" default:"POST"`
	Headers           []header                 `mapstructure:"headers" validate:"omitempty,dive"`
	ConnectionMode    common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

type header struct {
	From string `mapstructure:"from" validate:"omitempty,dynamic_or_pattern=single_line_1000"`
	To   string `mapstructure:"to" validate:"omitempty,dynamic_or_pattern=single_line_1000"`
}

// NewDefinition returns the classic Webhook destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("webhookUrl", "webhook_url"),
		converter.Simple("webhookMethod", "webhook_method"),
		converter.ArrayWithObjects("headers", "headers", map[string]any{
			"from": "from",
			"to":   "to",
		}),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "webhook",
		APIType:    "WEBHOOK",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{"headers.to"},
		NewConfig: func() any {
			return &webhookConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
