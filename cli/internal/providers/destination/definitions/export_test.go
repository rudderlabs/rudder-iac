package definitions

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

type testConnectionMode struct {
	Web     *string `json:"web" mapstructure:"web" validate:"omitempty,oneof=cloud device hybrid"`
	Android *string `json:"android" mapstructure:"android" validate:"omitempty,oneof=cloud device"`
}

type testGA4Config struct {
	APISecret      string             `json:"api_secret" mapstructure:"api_secret" validate:"required"`
	TypesOfClient  string             `json:"types_of_client" mapstructure:"types_of_client" validate:"required,oneof=gtag firebase"`
	MeasurementID  string             `json:"measurement_id" mapstructure:"measurement_id" validate:"required_if=TypesOfClient gtag"`
	DebugMode      *bool              `json:"debug_mode" mapstructure:"debug_mode"`
	ConnectionMode testConnectionMode `json:"connection_mode" mapstructure:"connection_mode"`
}

type testWebhookConnectionMode struct {
	Web *string `json:"web" mapstructure:"web" validate:"omitempty,oneof=cloud"`
}

type testWebhookConfig struct {
	WebhookURL     string                    `json:"webhook_url" mapstructure:"webhook_url" validate:"required"`
	ConnectionMode testWebhookConnectionMode `json:"connection_mode" mapstructure:"connection_mode"`
}

// GA4TestDefinition returns a destination definition used by external tests.
func GA4TestDefinition() *DestinationDefinition {
	return &DestinationDefinition{
		Type:    "GA4",
		Version: 1,
		Properties: []converter.ConfigProperty{
			converter.Simple("apiSecret", "api_secret"),
			converter.Simple("measurementId", "measurement_id"),
		},
		SecretKeys: []string{"api_secret"},
		NewConfig: func() any {
			return &testGA4Config{}
		},
		SourceTypes: []string{"web", "android"},
		ConnectionModes: map[string][]string{
			"web":     {"cloud", "device", "hybrid"},
			"android": {"cloud", "device"},
		},
	}
}

// WebhookTestDefinition returns a webhook destination definition used by external tests.
func WebhookTestDefinition(destType string, version int64) *DestinationDefinition {
	return &DestinationDefinition{
		Type:    destType,
		Version: version,
		Properties: []converter.ConfigProperty{
			converter.Simple("webhookUrl", "webhook_url"),
		},
		NewConfig: func() any {
			return &testWebhookConfig{}
		},
		SourceTypes: []string{"web"},
		ConnectionModes: map[string][]string{
			"web": {"cloud"},
		},
	}
}

// WebhookTestDefinitionWithoutConnectionMode returns a minimal webhook definition.
func WebhookTestDefinitionWithoutConnectionMode() *DestinationDefinition {
	return &DestinationDefinition{
		Type:    "WEBHOOK",
		Version: 1,
		NewConfig: func() any {
			return &struct {
				WebhookURL string `mapstructure:"webhook_url" validate:"required"`
			}{}
		},
	}
}
