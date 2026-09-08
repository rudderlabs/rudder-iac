package definitions

import (
	"github.com/rudderlabs/rudder-iac/cli/internal/provider/rules/funcs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/converter"
)

// testNestedEnumBlock is a generic nested block with enum-validated fields; it
// exercises nested-struct machinery and is deliberately not named after a real
// config key, so it does not squat on one the registry reserves a type for.
type testNestedEnumBlock struct {
	Web     *string `json:"web" mapstructure:"web" validate:"omitempty,dynamic_or_oneof=cloud device hybrid"`
	Android *string `json:"android" mapstructure:"android" validate:"omitempty,dynamic_or_oneof=cloud device"`
}

type testGA4Config struct {
	APISecret     string              `json:"api_secret" mapstructure:"api_secret" validate:"required"`
	TypesOfClient string              `json:"types_of_client" mapstructure:"types_of_client" validate:"required,dynamic_or_oneof=gtag firebase"`
	MeasurementID string              `json:"measurement_id" mapstructure:"measurement_id" validate:"required_if=TypesOfClient gtag"`
	DebugMode     *bool               `json:"debug_mode" mapstructure:"debug_mode"`
	NestedBlock   testNestedEnumBlock `json:"nested_block" mapstructure:"nested_block"`
}

type testWebhookNestedEnumBlock struct {
	Web *string `json:"web" mapstructure:"web" validate:"omitempty,dynamic_or_oneof=cloud"`
}

type testWebhookConfig struct {
	WebhookURL  string                     `json:"webhook_url" mapstructure:"webhook_url" validate:"required"`
	NestedBlock testWebhookNestedEnumBlock `json:"nested_block" mapstructure:"nested_block"`
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
		ConnectionRequiredKeys: map[string]map[string][]string{
			"web": {"cloud": {"api_secret"}},
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

// testDynamicPatternConfig exercises the dynamic_or_pattern tag.
type testDynamicPatternConfig struct {
	AccountID      string  `json:"account_id" mapstructure:"account_id" validate:"required,dynamic_or_pattern=test_digits"`
	SignUpSourceID string  `json:"sign_up_source_id" mapstructure:"sign_up_source_id" validate:"omitempty,dynamic_or_pattern=test_digits"`
	OptionalID     *string `json:"optional_id" mapstructure:"optional_id" validate:"omitempty,dynamic_or_pattern=test_digits"`
}

// DynamicPatternTestDefinition returns a definition whose config uses
// dynamic_or_pattern. Pattern registration is idempotent, so it happens here to
// keep the fixture self-contained.
func DynamicPatternTestDefinition() *DestinationDefinition {
	funcs.NewPattern("test_digits", `^[0-9]+$`, "must contain only digits")

	return &DestinationDefinition{
		Type:    "DYNAMICPATTERN",
		Version: 1,
		NewConfig: func() any {
			return &testDynamicPatternConfig{}
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
