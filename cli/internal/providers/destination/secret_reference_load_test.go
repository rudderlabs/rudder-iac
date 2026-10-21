package destination_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/project"
	"github.com/rudderlabs/rudder-iac/cli/internal/provider"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
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

	cp, err := provider.NewCompositeProvider(map[string]provider.Provider{
		"destination": destination.NewProvider(nil, secretTestRegistry(t)),
	})
	require.NoError(t, err)

	var out bytes.Buffer
	err = project.New(cp,
		project.WithLoader(memoryFiles(files)),
		project.WithRenderer(renderer.NewTextRenderer(&out)),
		project.WithSubstitutor(varsubst.NewSubstitutor(mapResolver(variables))),
	).Load("example")
	return out.String(), err
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
