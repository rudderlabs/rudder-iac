package destination_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/docs/docstest"
	"github.com/rudderlabs/rudder-iac/cli/internal/validation/renderer"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst"
)

type mapResolver map[string]string

func (m mapResolver) Resolve(name string) (string, bool) {
	v, ok := m[name]
	return v, ok
}

// renderLoad runs files through project.Load with the given variables and
// returns validate's text output.
func renderLoad(t *testing.T, files map[string]string, variables map[string]string) (string, error) {
	t.Helper()
	return renderLoadWith(t, renderer.NewTextRenderer, files, variables)
}

// renderLoadWith is renderLoad with the renderer validate picks for its
// output flag, such as renderer.NewJSONRenderer for --json.
func renderLoadWith(t *testing.T, newRenderer func(io.Writer) renderer.Renderer, files map[string]string, variables map[string]string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	err := docstest.Load(
		destinationProvider(t, destination.SecretRuleRegistry(t)),
		files,
		varsubst.NewSubstitutor(mapResolver(variables)),
		newRenderer(&out),
	)
	return out.String(), err
}

// destinationProvider is the provider validate builds, with the destination
// provider alone.
func destinationProvider(t *testing.T, registry *definitions.Registry) provider.Provider {
	t.Helper()

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"destination": destination.NewProvider(nil, registry),
	})
	require.NoError(t, err)
	return cp
}

const amplitudeWithLiteral = `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: fake-literal-amplitude-key
    api_secret: '{{ .SHOP_AMPLITUDE_API_SECRET }}'
    residency_server: standard
`

const webhookWithUndefinedURL = `version: rudder/v1
kind: destination
metadata:
  name: shop-webhook
spec:
  id: shop-webhook
  display_name: Shop Webhook
  type: webhook
  definition_version: 1
  enabled: true
  config:
    webhook_url: '{{ .SHOP_WEBHOOK_URL }}'
`

// Substitution stops the load at the first undefined variable, before any
// rule runs on the parsed specs. The secret check reads the specs as written,
// so its warning is still reported, whether the undefined variable is in the
// same file or another one.
func TestSecretReferenceReportedWhenSubstitutionFails(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{
		"amplitude.yaml": amplitudeWithLiteral,
		"webhook.yaml":   webhookWithUndefinedURL,
	}, nil)
	require.ErrorContains(t, err, "variable substitution failed")

	assert.Contains(t, out, `warning[destination/secret-reference]: secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'`)
	assert.Contains(t, out, "   12 | api_key: ***\n")
	assert.Contains(t, out, `error[project/var-substitution]: undefined variable "SHOP_AMPLITUDE_API_SECRET"`)
	assert.Contains(t, out, `error[project/var-substitution]: undefined variable "SHOP_WEBHOOK_URL"`)
	assert.Contains(t, out, "Found 2 error(s), 1 warning(s)")
	assert.NotContains(t, out, "fake-literal-amplitude-key")
}

// A literal beside an undefined reference gets a secret-reference warning and
// a substitution error on the same line. The error prints the line as written,
// so it takes the warning's masked line.
func TestSubstitutionErrorMasksSecretLine(t *testing.T) {
	t.Parallel()

	spec := `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: 'fake-literal-prefix{{ .UNDEF }}'
    residency_server: standard
`
	// The substitution error must land on the line the secret warning masked
	// with every line ending the YAML parser reads.
	for name, lineBreak := range map[string]string{"LF": "\n", "CRLF": "\r\n", "CR": "\r"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, err := renderLoad(t, map[string]string{"amplitude.yaml": strings.ReplaceAll(spec, "\n", lineBreak)}, nil)
			require.ErrorContains(t, err, "variable substitution failed")

			assert.Contains(t, out, `error[project/var-substitution]: undefined variable "UNDEF"`)
			assert.Equal(t, 2, strings.Count(out, "   12 | api_key: ***\n"), out)
			assert.NotContains(t, out, "fake-literal-prefix")
		})
	}
}

// A shared variable's line holds only the reference, so neither the warning
// nor a substitution error on that line is masked.
func TestSharedVariableLineIsNotMasked(t *testing.T) {
	t.Parallel()

	const amplitude = `version: rudder/v1
kind: destination
metadata:
  name: %[1]s
spec:
  id: %[1]s
  display_name: %[1]s
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: '{{ .SHARED }}'
    residency_server: standard
`
	out, err := renderLoad(t, map[string]string{
		"a.yaml": fmt.Sprintf(amplitude, "shop-a"),
		"b.yaml": fmt.Sprintf(amplitude, "shop-b"),
	}, nil)
	require.ErrorContains(t, err, "variable substitution failed")

	assert.Contains(t, out, "warning[destination/secret-reference-unique]")
	assert.Contains(t, out, "   12 | api_key: {{ .SHARED }}\n")
	assert.Contains(t, out, "   12 |     api_key: '{{ .SHARED }}'\n")
	assert.NotContains(t, out, "***")
}

// An unquoted reference followed by more text is not YAML until substitution,
// so the shared variable's line is read from the copy with placeholders. It
// still shows the reference as written.
func TestSharedVariableLineShowsReferenceInFileThatNeedsSubstitution(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{
		"amplitude.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: '{{ .SHARED }}'
    residency_server: standard
`,
		"webhook.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-webhook
spec:
  id: shop-webhook
  display_name: Shop Webhook
  type: webhook
  definition_version: 1
  enabled: true
  config:
    webhook_url: {{ .HOOK_BASE }}/hook
    headers:
      - to: '{{ .SHARED }}'
        from: X-Api-Key
`,
	}, map[string]string{
		"SHARED":    "fake-shared-secret-value",
		"HOOK_BASE": "https://example.com",
	})
	require.NoError(t, err)

	assert.Contains(t, out, `warning[destination/secret-reference-unique]: secret key "headers[0].to" of destination "shop-webhook" shares variable SHARED`)
	assert.Contains(t, out, "   14 | to: {{ .SHARED }}\n")
	assert.NotContains(t, out, "__rudder_variable_reference_")
	assert.NotContains(t, out, "fake-shared-secret-value")
}

func TestSecretReferenceReportedWhenSubstitutionSucceeds(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{
		"amplitude.yaml": amplitudeWithLiteral,
		"webhook.yaml":   webhookWithUndefinedURL,
	}, map[string]string{
		"SHOP_AMPLITUDE_API_SECRET": "fake-amplitude-secret-value",
		"SHOP_WEBHOOK_URL":          "https://example.com/hook",
	})
	require.NoError(t, err, "a warning alone must not fail validate")

	assert.Contains(t, out, `warning[destination/secret-reference]: secret key "api_key"`)
	assert.Contains(t, out, "Found 0 error(s), 1 warning(s)")
	assert.NotContains(t, out, "fake-literal-amplitude-key")
	assert.NotContains(t, out, "fake-amplitude-secret-value")
}

// Config errors are reported after substitution, so a line at a secret key, or
// at a list item holding one, would print the value from the var file.
func TestValidateMasksSubstitutedSecrets(t *testing.T) {
	t.Parallel()

	longSecret := "fake-" + strings.Repeat("s", 100)

	cases := []struct {
		name     string
		file     string
		wantLine string
	}{
		{
			name: "secret key",
			file: `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: '{{ .SECRET }}'
    residency_server: standard
`,
			wantLine: "   12 | api_key: ***\n",
		},
		{
			// In http the header value, from, is the secret.
			name: "http header value",
			file: `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - from: '{{ .SECRET }}'
        to: X-Api-Key
`,
			wantLine: "   17 | from: ***\n",
		},
		{
			name: "http header list item",
			file: `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - '{{ .SECRET }}'
`,
			wantLine: "   17 | ***\n",
		},
		{
			// A scalar list item has no key, so the ": " in its value must not
			// be read as one.
			name: "http header list item holding a colon",
			file: `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - 'Bearer: {{ .SECRET }}'
`,
			wantLine: "   17 | ***\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := renderLoad(t, map[string]string{"dest.yaml": tc.file}, map[string]string{"SECRET": longSecret})
			require.ErrorContains(t, err, "syntax validation failed")

			assert.Contains(t, out, "error[destination/spec-syntax-valid]")
			assert.Contains(t, out, tc.wantLine)
			assert.NotContains(t, out, longSecret)
		})
	}
}

// A multi-line value in a quoted reference keeps its line breaks after
// substitution, so it fails a single-line secret key's pattern. The error
// must print the key's line masked, with no line of the value.
func TestValidateMasksMultiLineSecret(t *testing.T) {
	t.Parallel()

	for _, quoted := range []string{`'{{ .SECRET }}'`, `"{{ .SECRET }}"`} {
		t.Run(quoted, func(t *testing.T) {
			t.Parallel()

			out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: ` + quoted + `
    residency_server: standard
`}, map[string]string{"SECRET": "fake-secret-line-one\nfake-secret-line-two\n"})
			require.ErrorContains(t, err, "syntax validation failed")

			assert.Contains(t, out, "must not contain line breaks")
			assert.Contains(t, out, "   12 | api_key: ***\n")
			assert.NotContains(t, out, "fake-secret-line")
		})
	}
}

// A null secret key holds no value to hide, so the error that it must not be
// null prints its line as written.
func TestValidatePrintsNullSecretKeyAsWritten(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: '{{ .SECRET }}'
    api_secret: ~
    residency_server: standard
`}, map[string]string{"SECRET": "fake-amplitude-key"})
	require.ErrorContains(t, err, "syntax validation failed")

	assert.Contains(t, out, "'api_secret' must not be null")
	assert.Contains(t, out, "   13 | api_secret: ~\n")
	assert.NotContains(t, out, "fake-amplitude-key")
}

// A config error on a key beside a secret prints that key's own line, which
// holds no secret, so the value the user has to fix stays visible.
func TestValidatePrintsLinesBesideSecrets(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - from: '{{ .SECRET }}'
        to: 'X-Bad:Name'
`}, map[string]string{"SECRET": "fake-header-secret"})
	require.ErrorContains(t, err, "syntax validation failed")

	assert.Contains(t, out, "   18 | to: X-Bad:Name\n")
	assert.NotContains(t, out, "fake-header-secret")
}

// A key beside a secret can hold an alias. Its error prints the alias's own
// line, not the list item's, whose first property is the secret.
func TestValidatePrintsAnAliasedLineBesideASecret(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - from: '{{ .FIRST }}'
        to: &bad 'X-Bad:Name'
      - from: '{{ .SECOND }}'
        to: *bad
`}, map[string]string{"FIRST": "fake-first-secret", "SECOND": "fake-second-secret"})
	require.ErrorContains(t, err, "syntax validation failed")

	assert.Contains(t, out, "   18 | to: X-Bad:Name\n")
	assert.Contains(t, out, "   20 | to: *bad\n")
	assert.NotContains(t, out, "fake-first-secret")
	assert.NotContains(t, out, "fake-second-secret")
}

// A header row aliased or merged into a path that holds no secret keeps its
// secret out of the error on that path: the line under the alias or merge
// shows the value masked.
func TestValidateMasksAliasedAndMergedRows(t *testing.T) {
	t.Parallel()

	for name, mapping := range map[string]string{
		"alias": `    properties_mapping:
      - *row`,
		"merge": `    properties_mapping:
      - to: $.context.secretish
        <<: *row`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - &row
        from: '{{ .SECRET }}'
        to: X-Api-Key
` + mapping + "\n"}, map[string]string{"SECRET": "fake/secret"})
			require.ErrorContains(t, err, "syntax validation failed")

			assert.Contains(t, out, "from: ***")
			assert.NotContains(t, out, "fake/secret")
		})
	}
}

// An alias of a secret under a key that is not one prints the alias, not the
// secret it resolves to.
func TestValidatePrintsAnAliasOfASecretAsWritten(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-amplitude
spec:
  id: shop-amplitude
  display_name: Shop Amplitude
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: &key '{{ .SECRET }}'
    residency_server: *key
`}, map[string]string{"SECRET": "fake-amplitude-secret"})
	require.ErrorContains(t, err, "syntax validation failed")

	assert.Contains(t, out, "   13 | residency_server: *key\n")
	assert.NotContains(t, out, "fake-amplitude-secret")
}

// A key beside a secret can come from a merge key. Its error prints the merged
// key's own line, not the list item's, whose first property is the secret.
func TestValidatePrintsAMergedLineBesideASecret(t *testing.T) {
	t.Parallel()

	out, err := renderLoad(t, map[string]string{"dest.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - &first
        from: '{{ .FIRST }}'
        to: 'X-Bad:Name'
      - from: '{{ .SECOND }}'
        <<: *first
`}, map[string]string{"FIRST": "fake-first-secret", "SECOND": "fake-second-secret"})
	require.ErrorContains(t, err, "syntax validation failed")

	assert.Contains(t, out, "   19 | to: X-Bad:Name\n")
	assert.NotContains(t, out, "fake-first-secret")
	assert.NotContains(t, out, "fake-second-secret")
}

// validate --json prints no line text, so masking has nothing to do there,
// but the secret warnings must still reach its diagnostics array with severity
// warning, and no value from a spec or the var file may reach any field.
func TestValidateJSONReportsSecretWarningsWithoutValues(t *testing.T) {
	t.Parallel()

	const (
		literalKey = "fake-literal-amplitude-key"
		varValue   = "fake-amplitude-secret-value"
	)
	longSecret := "fake-" + strings.Repeat("j", 100)

	const sharedAmplitude = `version: rudder/v1
kind: destination
metadata:
  name: %[1]s
spec:
  id: %[1]s
  display_name: %[1]s
  type: am
  definition_version: 1
  enabled: true
  config:
    api_key: '{{ .SHARED }}'
    residency_server: standard
`

	cases := []struct {
		name        string
		files       map[string]string
		variables   map[string]string
		wantErr     string
		want        []string // severity and ruleId of each diagnostic, in order
		wantMessage string   // part of the first diagnostic's message
	}{
		{
			name:        "literal secret",
			files:       map[string]string{"amplitude.yaml": amplitudeWithLiteral},
			variables:   map[string]string{"SHOP_AMPLITUDE_API_SECRET": varValue},
			want:        []string{"warning destination/secret-reference"},
			wantMessage: `secret key "api_key" must hold only a variable reference with no default, such as '{{ .SHOP_AMPLITUDE_API_KEY }}'`,
		},
		{
			name: "one variable in two secret slots",
			files: map[string]string{
				"a.yaml": fmt.Sprintf(sharedAmplitude, "shop-a"),
				"b.yaml": fmt.Sprintf(sharedAmplitude, "shop-b"),
			},
			variables: map[string]string{"SHARED": varValue},
			want: []string{
				"warning destination/secret-reference-unique",
				"warning destination/secret-reference-unique",
			},
			wantMessage: "SHARED",
		},
		{
			name: "undefined variables",
			files: map[string]string{
				"amplitude.yaml": amplitudeWithLiteral,
				"webhook.yaml":   webhookWithUndefinedURL,
			},
			wantErr: "variable substitution failed",
			want: []string{
				"warning destination/secret-reference",
				"error project/var-substitution",
				"error project/var-substitution",
			},
			wantMessage: `secret key "api_key"`,
		},
		{
			name: "config error on a substituted secret",
			files: map[string]string{"http.yaml": `version: rudder/v1
kind: destination
metadata:
  name: shop-http
spec:
  id: shop-http
  display_name: Shop HTTP
  type: http
  definition_version: 1
  enabled: true
  config:
    api_url: https://example.com/events
    auth: noAuth
    method: POST
    format: JSON
    headers:
      - from: '{{ .SECRET }}'
        to: X-Api-Key
`},
			variables:   map[string]string{"SECRET": longSecret},
			wantErr:     "syntax validation failed",
			want:        []string{"error destination/spec-syntax-valid"},
			wantMessage: "from",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out, err := renderLoadWith(t, renderer.NewJSONRenderer, tc.files, tc.variables)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err, "a warning alone must not fail validate")
			}

			var doc struct {
				Diagnostics []struct {
					RuleID   string `json:"ruleId"`
					Severity string `json:"severity"`
					Message  string `json:"message"`
				} `json:"diagnostics"`
				Summary struct {
					Errors   int `json:"errors"`
					Warnings int `json:"warnings"`
				} `json:"summary"`
			}
			require.NoError(t, json.Unmarshal([]byte(out), &doc), out)

			var got []string
			var errorCount, warningCount int
			for _, d := range doc.Diagnostics {
				got = append(got, d.Severity+" "+d.RuleID)
				switch d.Severity {
				case "error":
					errorCount++
				case "warning":
					warningCount++
				}
			}
			require.Equal(t, tc.want, got, out)
			assert.Contains(t, doc.Diagnostics[0].Message, tc.wantMessage)
			assert.Equal(t, errorCount, doc.Summary.Errors)
			assert.Equal(t, warningCount, doc.Summary.Warnings)

			for _, secret := range []string{literalKey, varValue, longSecret} {
				assert.NotContains(t, out, secret)
			}
		})
	}
}
