package slack

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

const (
	slackWebhookPattern       = `^(.{0,100})$`
	slackWebhookRejectPattern = `\.ngrok\.io`
)

func init() {
	funcs.NewPatternWithReject(
		"slack_webhook_url",
		slackWebhookPattern,
		slackWebhookRejectPattern,
		"must be at most 100 characters, must not contain line breaks, and must not use ngrok",
	)
}

// Source types from integrations-config destinations/slack/db-config.json.
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

// slackConfig is the local YAML config model. Field set mirrors the Terraform
// Slack destination surface; validation constraints mirror schema.json for those
// mapped fields.
type slackConfig struct {
	WebhookURL               string                   `mapstructure:"webhook_url" validate:"required,dynamic_or_pattern=slack_webhook_url"`
	IdentifyTemplate         string                   `mapstructure:"identify_template" validate:"omitempty,dynamic_or_pattern=single_line_1000"`
	EventChannelSettings     []eventChannelSetting    `mapstructure:"event_channel_settings" validate:"omitempty,dive"`
	EventTemplateSettings    []eventTemplateSetting   `mapstructure:"event_template_settings" validate:"omitempty,dive"`
	WhitelistedTraitSettings []string                 `mapstructure:"whitelisted_trait_settings" validate:"omitempty,dive,dynamic_or_pattern=single_line_100"`
	ConsentManagement        common.ConsentManagement `mapstructure:"consent_management"`
}

type eventChannelSetting struct {
	Name    string `mapstructure:"name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Channel string `mapstructure:"channel" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Regex   *bool  `mapstructure:"regex"`
}

type eventTemplateSetting struct {
	Name     string `mapstructure:"name" validate:"omitempty,dynamic_or_pattern=single_line_100"`
	Template string `mapstructure:"template" validate:"omitempty,dynamic_or_pattern=single_line_1000"`
	Regex    *bool  `mapstructure:"regex"`
}

// NewDefinition returns the Slack destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("webhookUrl", "webhook_url"),
		converter.Simple("identifyTemplate", "identify_template"),
		converter.ArrayWithObjects("eventChannelSettings", "event_channel_settings", map[string]any{
			"eventName":    "name",
			"eventChannel": "channel",
			"eventRegex":   "regex",
		}),
		converter.ArrayWithObjects("eventTemplateSettings", "event_template_settings", map[string]any{
			"eventName":     "name",
			"eventTemplate": "template",
			"eventRegex":    "regex",
		}),
		converter.ArrayWithStrings("whitelistedTraitsSettings", "trait", "whitelisted_trait_settings"),
	}
	properties = append(properties, common.Properties(sourceTypes)...)

	return &definitions.DestinationDefinition{
		Type:       "slack",
		APIType:    "SLACK",
		Version:    1,
		Properties: properties,
		SecretKeys: []string{},
		NewConfig: func() any {
			return &slackConfig{}
		},
		SourceTypes:     append([]string(nil), sourceTypes...),
		ConnectionModes: connectionModes,
	}
}
