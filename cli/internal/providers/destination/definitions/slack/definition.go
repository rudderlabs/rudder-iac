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

	// Event names double as user-supplied regexes (see the regex flag alongside
	// them), so schema.json guards them against catastrophic backtracking with
	// ^(?!.*\([^()]*(?:[*+{]|\|)[^()]*\)\s*[*+{?])(.{0,100})$. RE2 has no
	// lookahead, so the guard becomes a reject pattern: a parenthesised group
	// containing * + { or |, followed by a quantifier.
	slackEventNamePattern       = `^(.{0,100})$`
	slackEventNameRejectPattern = `\([^()]*(?:[*+{]|\|)[^()]*\)\s*[*+{?]`
)

func init() {
	funcs.NewPatternWithReject(
		"slack_webhook_url",
		slackWebhookPattern,
		slackWebhookRejectPattern,
		"must be at most 100 characters, must not contain line breaks, and must not use ngrok",
	)

	funcs.NewPatternWithReject(
		"slack_event_name",
		slackEventNamePattern,
		slackEventNameRejectPattern,
		"must be at most 100 characters and must not use a quantified group such as (a|b)+",
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
	common.SourceTypeCloud,
	common.SourceTypeReactNative,
	common.SourceTypeFlutter,
	common.SourceTypeCordova,
}

var connectionModes = map[string][]string{
	common.SourceTypeAndroid:       {"cloud"},
	common.SourceTypeAndroidKotlin: {"cloud"},
	common.SourceTypeIOS:           {"cloud"},
	common.SourceTypeIOSSwift:      {"cloud"},
	common.SourceTypeWeb:           {"cloud"},
	common.SourceTypeUnity:         {"cloud"},
	common.SourceTypeCloud:         {"cloud"},
	common.SourceTypeReactNative:   {"cloud"},
	common.SourceTypeFlutter:       {"cloud"},
	common.SourceTypeCordova:       {"cloud"},
}

// slackConfig is the local YAML config model. Field set mirrors the Terraform
// Slack destination surface; validation constraints mirror schema.json for those
// mapped fields.
type slackConfig struct {
	WebhookURL               string                   `mapstructure:"webhook_url" validate:"required,pattern=slack_webhook_url"`
	IncomingWebhooksType     string                   `mapstructure:"incoming_webhooks_type" validate:"omitempty,oneof=legacy modern" default:"legacy"`
	IdentifyTemplate         string                   `mapstructure:"identify_template" validate:"omitempty,pattern=single_line_1000"`
	EventChannelSettings     []eventChannelSetting    `mapstructure:"event_channel_settings" validate:"omitempty,dive"`
	EventTemplateSettings    []eventTemplateSetting   `mapstructure:"event_template_settings" validate:"omitempty,dive"`
	WhitelistedTraitSettings []string                 `mapstructure:"whitelisted_trait_settings" validate:"omitempty,dive,pattern=single_line_100"`
	DenyListOfEvents         []string                 `mapstructure:"deny_list_of_events" validate:"omitempty,dive,pattern=single_line_100"`
	ConnectionMode           common.ConnectionMode    `mapstructure:"connection_mode"`
	ConsentManagement        common.ConsentManagement `mapstructure:"consent_management"`
}

type eventChannelSetting struct {
	Name    string `mapstructure:"name" validate:"omitempty,pattern=slack_event_name"`
	Channel string `mapstructure:"channel" validate:"omitempty,pattern=single_line_100"`
	Webhook string `mapstructure:"webhook" validate:"omitempty,pattern=slack_webhook_url"`
	Regex   *bool  `mapstructure:"regex"`
}

type eventTemplateSetting struct {
	Name     string `mapstructure:"name" validate:"omitempty,pattern=slack_event_name"`
	Template string `mapstructure:"template" validate:"omitempty,pattern=single_line_1000"`
	Regex    *bool  `mapstructure:"regex"`
}

// NewDefinition returns the Slack destination definition.
func NewDefinition() *definitions.DestinationDefinition {
	properties := []converter.ConfigProperty{
		converter.Simple("webhookUrl", "webhook_url"),
		converter.Simple("incomingWebhooksType", "incoming_webhooks_type"),
		converter.Simple("identifyTemplate", "identify_template"),
		converter.ArrayWithObjects("eventChannelSettings", "event_channel_settings", map[string]any{
			"eventName":           "name",
			"eventChannel":        "channel",
			"eventChannelWebhook": "webhook",
			"eventRegex":          "regex",
		}),
		converter.ArrayWithObjects("eventTemplateSettings", "event_template_settings", map[string]any{
			"eventName":     "name",
			"eventTemplate": "template",
			"eventRegex":    "regex",
		}),
		converter.ArrayWithStrings("whitelistedTraitsSettings", "trait", "whitelisted_trait_settings"),
		converter.ArrayWithStrings("denyListOfEvents", "eventName", "deny_list_of_events"),
	}
	properties = append(properties, common.ConnectionModeProperties(sourceTypes)...)
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
