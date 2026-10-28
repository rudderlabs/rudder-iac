package destination

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	prules "github.com/rudderlabs/rudder-iac/cli/internal/provider/rules"
	vrules "github.com/rudderlabs/rudder-iac/cli/internal/validation/rules"
)

// destinationSource builds a destination spec file as written.
func destinationSource(id, destinationType, config string) []byte {
	return fmt.Appendf(nil, `version: rudder/v1
kind: destination
metadata:
  name: %[1]s
spec:
  id: %[1]s
  display_name: %[1]s
  type: %[2]s
  definition_version: 1
  enabled: true
  config:
%[3]s`, id, destinationType, config)
}

func sourceContexts(sources map[string][]byte) map[string]*vrules.ValidationContext {
	contexts := make(map[string]*vrules.ValidationContext, len(sources))
	for path, source := range sources {
		contexts[path] = vrules.NewSourceContext(path, &specs.RawSpec{Data: source})
	}
	return contexts
}

func TestSecretReferenceRuleMetadata(t *testing.T) {
	t.Parallel()

	rule := NewSecretReferenceRule(secretRuleRegistry(t))
	assert.Equal(t, SecretReferenceRuleID, rule.ID())
	assert.Equal(t, vrules.Warning, rule.Severity())
	assert.NotEmpty(t, rule.Description())
	assert.Equal(t, prules.V1VersionPatterns(DestinationSpecKind), rule.AppliesTo())
	assert.Nil(t, rule.Validate(&vrules.ValidationContext{}))
	assert.Implements(t, (*vrules.SourceRule)(nil), rule)
}

func TestSecretReferenceRule(t *testing.T) {
	t.Parallel()

	const fix = "; set the value in a var file or a RUDDER_ environment variable"

	cases := []struct {
		name            string
		id              string
		destinationType string
		config          string
		want            []vrules.ValidationResult
	}{
		{
			name:            "single-quoted reference",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: '{{ .SHOP_AMPLITUDE_API_KEY }}'
    api_secret: '{{ .ANY_NAME_WORKS }}'
    residency_server: standard
`,
		},
		{
			name:            "double-quoted and unquoted references",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: "{{ .SHOP_AMPLITUDE_API_KEY }}"
    api_secret: {{ .SHOP_AMPLITUDE_API_SECRET }}
    residency_server: standard
`,
		},
		{
			name:            "literal",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: fake-amplitude-api-key
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'` + fix,
			}},
		},
		{
			name:            "non-string literal",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: 1234567890
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'` + fix,
			}},
		},
		{
			name:            "reference with a default",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: '{{ .MY_AMPLITUDE_KEY | fake-amplitude-api-key }}'
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default; remove the default from the reference to MY_AMPLITUDE_KEY and` + strings.TrimPrefix(fix, ";"),
			}},
		},
		{
			name:            "reference with text around it",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: 'prefix-{{ .SHOP_AMPLITUDE_API_KEY }}'
    api_secret: '{{ .FIRST }}{{ .SECOND }}'
    residency_server: standard
`,
			want: []vrules.ValidationResult{
				{
					Reference: "/spec/config/api_secret",
					Message:   `secret key "api_secret" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_SECRET }}'` + fix,
				},
				{
					Reference: "/spec/config/api_key",
					Message:   `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'` + fix,
				},
			},
		},
		{
			name:            "literal inside a list",
			id:              "shop-webhook",
			destinationType: "webhook",
			config: `    webhook_url: https://example.com/hook
    headers:
      - from: Authorization
        to: '{{ .SHOP_WEBHOOK_HEADERS_0_TO }}'
      - from: X-Api-Key
        to: fake-header-value
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/headers/1/to",
				Message:   `secret key "headers[1].to" must hold only a variable reference with no default, such as '{{ .SHOP_WEBHOOK_HEADERS_1_TO }}'` + fix,
			}},
		},
		{
			// The unquoted reference followed by "/hook" is not YAML until
			// substitution runs.
			name:            "spec that parses only after substitution",
			id:              "shop-webhook",
			destinationType: "webhook",
			config: `    webhook_url: {{ .WEBHOOK_HOST }}/hook
    headers:
      - from: Authorization
        to: fake-header-value
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/headers/0/to",
				Message:   `secret key "headers[0].to" must hold only a variable reference with no default, such as '{{ .SHOP_WEBHOOK_HEADERS_0_TO }}'` + fix,
			}},
		},
		{
			name:            "empty values hold no secret",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key:
    api_secret: ""
    residency_server: standard
`,
		},
		{
			// http reads a header value from the event when it is a JSONPath.
			name:            "JSONPath read from each event",
			id:              "shop-http",
			destinationType: "http",
			config: `    api_url: https://example.com/events
    headers:
      - to: X-Request-Id
        from: $.messageId
      - to: X-Anonymous-Id
        from: $['anonymousId']
`,
		},
		{
			// Only http's headers.from reads a JSONPath. Elsewhere a value that
			// looks like one is a literal.
			name:            "JSONPath in a key that does not read one",
			id:              "shop-http",
			destinationType: "http",
			config: `    api_url: https://example.com/events
    auth: basicAuth
    username: '{{ .SHOP_HTTP_USERNAME }}'
    password: $.fake-password
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/password",
				Message:   `secret key "password" must hold only a variable reference with no default, such as '{{ .SHOP_HTTP_PASSWORD }}'` + fix,
			}},
		},
		{
			name:            "JSONPath in a webhook header value",
			id:              "shop-webhook",
			destinationType: "webhook",
			config: `    webhook_url: https://example.com/hook
    headers:
      - from: Authorization
        to: $['fake-token']
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/headers/0/to",
				Message:   `secret key "headers[0].to" must hold only a variable reference with no default, such as '{{ .SHOP_WEBHOOK_HEADERS_0_TO }}'` + fix,
			}},
		},
		{
			// A template resolves per event. Its fallback is a literal, so a
			// template with one is still reported.
			name:            "templates read from each event",
			id:              "shop-webhook",
			destinationType: "webhook",
			config: `    webhook_url: https://example.com/hook
    headers:
      - from: Authorization
        to: '{{ message.context.authToken || "" }}'
      - from: X-Api-Key
        to: '{{ message.context.apiKey || "fake-fallback-key" }}'
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/headers/1/to",
				Message:   `secret key "headers[1].to" must hold only a variable reference with no default, such as '{{ .SHOP_WEBHOOK_HEADERS_1_TO }}'` + fix,
			}},
		},
		{
			// Each template has its own fallback, so a value of two holds the
			// first one's literal even when the last one's is empty.
			name:            "two templates, the first with a literal fallback",
			id:              "shop-webhook",
			destinationType: "webhook",
			config: `    webhook_url: https://example.com/hook
    headers:
      - from: Authorization
        to: '{{ message.token || "fake-literal-token" }}{{ message.other || "" }}'
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/headers/0/to",
				Message:   `secret key "headers[0].to" must hold only a variable reference with no default, such as '{{ .SHOP_WEBHOOK_HEADERS_0_TO }}'` + fix,
			}},
		},
		{
			name:            "id that makes no variable name",
			id:              "shop.amplitude",
			destinationType: "am",
			config: `    api_key: fake-amplitude-api-key
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default` + fix,
			}},
		},
		{
			// The id reads as a placeholder before substitution, which must
			// not reach the suggested name.
			name:            "id that holds a reference",
			id:              "'{{ .DEST_ID }}'",
			destinationType: "am",
			config: `    api_key: fake-amplitude-api-key
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default` + fix,
			}},
		},
		{
			name:            "empty id",
			id:              `""`,
			destinationType: "am",
			config: `    api_key: fake-amplitude-api-key
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default` + fix,
			}},
		},
		{
			// A spec cannot pass for a reference by holding the placeholder
			// text that stands in for one.
			name:            "literal that reads like a placeholder",
			id:              "shop-amplitude",
			destinationType: "am",
			config: `    api_key: __rudder_variable_reference_0__
    api_secret: '{{ .SHOP_AMPLITUDE_API_SECRET }}'
    residency_server: standard
`,
			want: []vrules.ValidationResult{{
				Reference: "/spec/config/api_key",
				Message:   `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'` + fix,
			}},
		},
		{
			name:            "unregistered type is left to the syntax rule",
			id:              "shop-unknown",
			destinationType: "not_a_type",
			config: `    api_key: fake-api-key
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := destinationSource(tc.id, tc.destinationType, tc.config)
			rule := NewSecretReferenceRule(secretRuleRegistry(t)).(vrules.SourceRule)

			results := rule.ValidateSources(sourceContexts(map[string][]byte{"dest.yaml": source}))
			if tc.want == nil {
				assert.Empty(t, results)
				return
			}
			assert.Equal(t, map[string][]vrules.ValidationResult{"dest.yaml": tc.want}, results)
		})
	}
}

// definition_version decides which keys are secret. When it names no
// registered version, the keys of every version of the type are checked, so a
// literal is still reported. A type that is a reference cannot be read, and the
// spec is skipped.
func TestSecretReferenceRuleUnresolvedDefinition(t *testing.T) {
	t.Parallel()

	const header = `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
`
	const config = `  enabled: true
  config:
    api_key: fake-amplitude-api-key
    residency_server: standard
`
	want := []vrules.ValidationResult{{
		Reference: "/spec/config/api_key",
		Message:   `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'; set the value in a var file or a RUDDER_ environment variable`,
	}}

	cases := []struct {
		name       string
		definition string
		want       []vrules.ValidationResult
	}{
		{name: "missing version", definition: "  type: am\n", want: want},
		{name: "version that is a reference", definition: "  type: am\n  definition_version: {{ .VERSION }}\n", want: want},
		{name: "unregistered version", definition: "  type: am\n  definition_version: 99\n", want: want},
		{name: "type that is a reference", definition: "  type: '{{ .TYPE }}'\n  definition_version: 1\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			source := []byte(header + tc.definition + config)
			rule := NewSecretReferenceRule(secretRuleRegistry(t)).(vrules.SourceRule)

			results := rule.ValidateSources(sourceContexts(map[string][]byte{"dest.yaml": source}))
			if tc.want == nil {
				assert.Empty(t, results)
				return
			}
			assert.Equal(t, map[string][]vrules.ValidationResult{"dest.yaml": tc.want}, results)
		})
	}
}

// Fields outside config may hold references too. Reading the spec before
// substitution must not stop at a placeholder where a bool or a number belongs.
func TestSecretReferenceRuleReferencesOutsideConfig(t *testing.T) {
	t.Parallel()

	source := []byte(`version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: '{{ .DISPLAY_NAME }}'
  type: am
  definition_version: 1
  enabled: {{ .ENABLED }}
  config:
    api_key: fake-amplitude-api-key
    residency_server: standard
`)
	rule := NewSecretReferenceRule(secretRuleRegistry(t)).(vrules.SourceRule)

	results := rule.ValidateSources(sourceContexts(map[string][]byte{"dest.yaml": source}))
	assert.Equal(t, map[string][]vrules.ValidationResult{"dest.yaml": {{
		Reference: "/spec/config/api_key",
		Message:   `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'; set the value in a var file or a RUDDER_ environment variable`,
	}}}, results)
}
