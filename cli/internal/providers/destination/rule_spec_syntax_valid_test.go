package destination

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/common"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// ruleTestConfig types source-type-scoped blocks as open maps so the config
// model accepts arbitrary platform keys — the source-type key check is then
// the only guard, which is exactly what these tests exercise.
type ruleTestConfig struct {
	WebhookURL        string                   `mapstructure:"webhook_url" validate:"required"`
	ConnectionMode    map[string]string        `mapstructure:"connection_mode"`
	UseNativeSDK      map[string]bool          `mapstructure:"use_native_sdk"`
	ConsentManagement common.ConsentManagement `mapstructure:"consent_management"`
}

func ruleTestRegistry(t *testing.T) *definitions.Registry {
	t.Helper()

	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(&definitions.DestinationDefinition{
		Type:    "WEBHOOK",
		Version: 1,
		NewConfig: func() any {
			return &ruleTestConfig{}
		},
		SourceTypes: []string{"web", "react_native"},
		ConnectionModes: map[string][]string{
			"web":          {"cloud"},
			"react_native": {"cloud"},
		},
	}))
	return registry
}

func validSpecMap() map[string]any {
	return map[string]any{
		"id":                 "webhook-prod",
		"display_name":       "Production Webhook",
		"type":               "WEBHOOK",
		"enabled":            true,
		"definition_version": 1,
		"config": map[string]any{
			"webhook_url": "https://example.com/hook",
		},
	}
}

func runSyntaxRule(t *testing.T, registry *definitions.Registry, spec map[string]any) []vrules.ValidationResult {
	t.Helper()

	rule := NewSpecSyntaxValidRule(registry)
	return rule.Validate(&vrules.ValidationContext{
		Spec:    spec,
		Kind:    DestinationSpecKind,
		Version: "rudder/v1",
	})
}

func TestSpecSyntaxValidRuleMetadata(t *testing.T) {
	t.Parallel()

	rule := NewSpecSyntaxValidRule(ruleTestRegistry(t))
	assert.Equal(t, SpecSyntaxValidRuleID, rule.ID())
	assert.Equal(t, vrules.Error, rule.Severity())
	assert.NotEmpty(t, rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(DestinationSpecKind), rule.AppliesTo())
}

func TestSpecSyntaxValidRuleDecodeFailure(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	spec["definition_version"] = "abc"

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	require.Len(t, results, 1)
	assert.Equal(t, "/spec", results[0].Reference)
	assert.Contains(t, results[0].Message, "definition_version")
}

func TestSpecSyntaxValidRuleUnknownEnvelopeKeys(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	spec["foo"] = "x"
	spec["bar"] = "y"

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	assert.Equal(t, []vrules.ValidationResult{
		{Reference: "/spec/bar", Message: `unknown field "bar"`},
		{Reference: "/spec/foo", Message: `unknown field "foo"`},
	}, results)
}

func TestSpecSyntaxValidRuleEnvelopeErrorsShortCircuitRegistryChecks(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	delete(spec, "id")
	spec["type"] = "UNREGISTERED"
	spec["extra"] = true

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	assert.Equal(t, []vrules.ValidationResult{
		{Reference: "/spec/extra", Message: `unknown field "extra"`},
		{Reference: "/spec/id", Message: "'id' is required"},
	}, results)
}

func TestSpecSyntaxValidRuleRequiredFields(t *testing.T) {
	t.Parallel()

	results := runSyntaxRule(t, ruleTestRegistry(t), map[string]any{})
	assert.Equal(t, []vrules.ValidationResult{
		{Reference: "/spec/id", Message: "'id' is required"},
		{Reference: "/spec/display_name", Message: "'display_name' is required"},
		{Reference: "/spec/type", Message: "'type' is required"},
		{Reference: "/spec/definition_version", Message: "'definition_version' is required"},
	}, results)
}

func TestSpecSyntaxValidRuleDisplayNamePattern(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		value    string
		expected []vrules.ValidationResult
	}{
		{
			name:  "valid display name",
			value: "Production Webhook",
		},
		{
			name:  "too short",
			value: "a",
			expected: []vrules.ValidationResult{{
				Reference: "/spec/display_name",
				Message:   "'display_name' is not valid: must be 2-100 characters and contain only letters, digits, underscores, spaces, periods, and hyphens",
			}},
		},
		{
			name:  "invalid character",
			value: "Webhook@Prod",
			expected: []vrules.ValidationResult{{
				Reference: "/spec/display_name",
				Message:   "'display_name' is not valid: must be 2-100 characters and contain only letters, digits, underscores, spaces, periods, and hyphens",
			}},
		},
		{
			name:  "too long",
			value: strings.Repeat("a", 101),
			expected: []vrules.ValidationResult{{
				Reference: "/spec/display_name",
				Message:   "'display_name' is not valid: must be 2-100 characters and contain only letters, digits, underscores, spaces, periods, and hyphens",
			}},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpecMap()
			spec["display_name"] = c.value

			results := runSyntaxRule(t, ruleTestRegistry(t), spec)
			if c.expected == nil {
				assert.Empty(t, results)
				return
			}
			assert.Equal(t, c.expected, results)
		})
	}
}

func TestSpecSyntaxValidRuleConfigDynamicValues(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config map[string]any
	}{
		{
			name: "allows env reference",
			config: map[string]any{
				"webhook_url": "env.WEBHOOK_URL",
			},
		},
		{
			name: "allows ui template fallback",
			config: map[string]any{
				"webhook_url": "{{ config.url || https://example.com/hook }}",
			},
		},
		{
			name: "allows iac variable substitution",
			config: map[string]any{
				"webhook_url": "{{ .WEBHOOK_URL }}",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpecMap()
			spec["config"] = c.config

			results := runSyntaxRule(t, ruleTestRegistry(t), spec)
			assert.Empty(t, results)
		})
	}
}

func TestSpecSyntaxValidRuleUnregisteredType(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	spec["type"] = "NOPE"

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	assert.Equal(t, []vrules.ValidationResult{
		{
			Reference: "/spec/type",
			Message:   "destination type 'NOPE' is not supported; supported types: WEBHOOK",
		},
	}, results)
}

func TestSpecSyntaxValidRuleEmptyRegistry(t *testing.T) {
	t.Parallel()

	results := runSyntaxRule(t, definitions.NewRegistry(), validSpecMap())
	assert.Equal(t, []vrules.ValidationResult{
		{
			Reference: "/spec/type",
			Message:   "destination type 'WEBHOOK' is not supported; no destination types are currently supported",
		},
	}, results)
}

func TestSpecSyntaxValidRuleInvalidVersion(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	spec["definition_version"] = 2

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	assert.Equal(t, []vrules.ValidationResult{
		{
			Reference: "/spec/definition_version",
			Message:   "definition_version 2 is not valid for destination type 'WEBHOOK'; valid versions: 1",
		},
	}, results)
}

func TestSpecSyntaxValidRuleTransformationRefFormat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ref     string
		invalid bool
	}{
		{name: "valid ref", ref: "#transformation:my-id"},
		{name: "missing hash", ref: "transformation:my-id", invalid: true},
		{name: "wrong kind", ref: "#tp:my-id", invalid: true},
		{name: "empty id", ref: "#transformation:", invalid: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpecMap()
			spec["transformation"] = c.ref

			results := runSyntaxRule(t, ruleTestRegistry(t), spec)
			if !c.invalid {
				assert.Empty(t, results)
				return
			}
			assert.Equal(t, []vrules.ValidationResult{
				{
					Reference: "/spec/transformation",
					Message:   "'transformation' is invalid: must be of pattern #transformation:<id>",
				},
			}, results)
		})
	}
}

func TestSpecSyntaxValidRuleConfigErrors(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	spec["config"] = map[string]any{"extra": 1}

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	assert.Equal(t, []vrules.ValidationResult{
		{Reference: "/spec/config/extra", Message: `unknown config field "extra"`},
		{Reference: "/spec/config/webhook_url", Message: "'webhook_url' is required"},
	}, results)
}

func TestSpecSyntaxValidRuleSourceTypeKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		config   map[string]any
		expected []vrules.ValidationResult
	}{
		{
			name: "unsupported source type under connection_mode",
			config: map[string]any{
				"webhook_url":     "https://example.com/hook",
				"connection_mode": map[string]any{"ios": "cloud"},
			},
			expected: []vrules.ValidationResult{
				{
					Reference: "/spec/config/connection_mode/ios",
					Message:   "source type 'ios' is not supported by destination type 'WEBHOOK'; supported source types: web, react_native",
				},
			},
		},
		{
			name: "unsupported source type under use_native_sdk",
			config: map[string]any{
				"webhook_url":    "https://example.com/hook",
				"use_native_sdk": map[string]any{"ios": true},
			},
			expected: []vrules.ValidationResult{
				{
					Reference: "/spec/config/use_native_sdk/ios",
					Message:   "source type 'ios' is not supported by destination type 'WEBHOOK'; supported source types: web, react_native",
				},
			},
		},
		{
			name: "unsupported source type under consent_management",
			config: map[string]any{
				"webhook_url": "https://example.com/hook",
				"consent_management": map[string]any{
					"ios": []any{},
				},
			},
			expected: []vrules.ValidationResult{
				{
					Reference: "/spec/config/consent_management/ios",
					Message:   "source type 'ios' is not supported by destination type 'WEBHOOK'; supported source types: web, react_native",
				},
			},
		},
		{
			name: "supported snake_case source types pass",
			config: map[string]any{
				"webhook_url":     "https://example.com/hook",
				"connection_mode": map[string]any{"web": "cloud", "react_native": "cloud"},
				"use_native_sdk":  map[string]any{"react_native": true},
			},
			expected: []vrules.ValidationResult{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpecMap()
			spec["config"] = c.config

			results := runSyntaxRule(t, ruleTestRegistry(t), spec)
			assert.Equal(t, c.expected, results)
		})
	}
}

func TestSpecSyntaxValidRuleScalarSourceTypeBlock(t *testing.T) {
	t.Parallel()

	spec := validSpecMap()
	spec["config"] = map[string]any{
		"webhook_url":     "https://example.com/hook",
		"connection_mode": "cloud",
	}

	results := runSyntaxRule(t, ruleTestRegistry(t), spec)
	// The config model owns the shape error; the source-type key check must
	// skip the scalar without panicking or piling on.
	require.Len(t, results, 1)
	assert.Equal(t, "/spec/config/connection_mode", results[0].Reference)
}

func TestSpecSyntaxValidRuleValidSpec(t *testing.T) {
	t.Parallel()

	results := runSyntaxRule(t, ruleTestRegistry(t), validSpecMap())
	assert.Empty(t, results)
}
