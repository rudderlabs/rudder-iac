package destination_test

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/project/specs"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/am"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/braze"
	facebookpixel "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/facebook_pixel"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/googlesheets"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandlerImpl_FormatForExport_RequiredSecretsTheAPIStrips pins that import
// writes a reference for every secret the destination's own validation
// requires, even when the API does not return it, so an imported spec
// validates once the var file is filled in. A secret the config does not
// require is still left out.
func TestHandlerImpl_FormatForExport_RequiredSecretsTheAPIStrips(t *testing.T) {
	t.Parallel()

	registry := definitions.NewRegistry()
	for _, def := range []*definitions.DestinationDefinition{httpdest.NewDefinition(), googlesheets.NewDefinition(), s3.NewDefinition(), am.NewDefinition(), braze.NewDefinition(), facebookpixel.NewDefinition()} {
		require.NoError(t, registry.Register(def))
	}

	cases := []struct {
		name     string
		apiType  string
		config   string
		expected map[string]any
		// contains, when set, replaces expected with a subset check on the
		// keys a connect-time requirement adds.
		contains map[string]any
	}{
		{
			name:    "http bearer token when auth is bearerTokenAuth",
			apiType: "HTTP",
			config:  `{"apiUrl":"https://example.com/hook","auth":"bearerTokenAuth","method":"POST","format":"JSON"}`,
			expected: map[string]any{
				"api_url": "https://example.com/hook", "auth": "bearerTokenAuth", "method": "POST", "format": "JSON",
				"bearer_token": "{{ .DEST_BEARER_TOKEN }}",
			},
		},
		{
			name:    "no http secret when auth needs none",
			apiType: "HTTP",
			config:  `{"apiUrl":"https://example.com/hook","auth":"noAuth","method":"POST","format":"JSON"}`,
			expected: map[string]any{
				"api_url": "https://example.com/hook", "auth": "noAuth", "method": "POST", "format": "JSON",
			},
		},
		{
			name:    "google sheets credentials",
			apiType: "GOOGLESHEETS",
			config:  `{"sheetId":"sheet","sheetName":"Sheet1","eventKeyMap":[{"from":"id","to":"id"}]}`,
			expected: map[string]any{
				"sheet_id": "sheet", "sheet_name": "Sheet1",
				"event_key_map": []any{map[string]any{"from": "id", "to": "id"}},
				"credentials":   "{{ .DEST_CREDENTIALS }}",
			},
		},
		{
			name:    "s3 access keys when role based auth is off",
			apiType: "S3",
			config:  `{"bucketName":"bucket","roleBasedAuth":false,"enableSSE":false}`,
			expected: map[string]any{
				"bucket_name": "bucket", "role_based_auth": false, "enable_sse": false,
				"access_key_id": "{{ .DEST_ACCESS_KEY_ID }}",
				"access_key":    "{{ .DEST_ACCESS_KEY }}",
			},
		},
		{
			name:    "no s3 access keys when role based auth is on",
			apiType: "S3",
			config:  `{"bucketName":"bucket","roleBasedAuth":true,"iamRoleARN":"arn:aws:iam::123456789012:role/rudder","enableSSE":false}`,
			expected: map[string]any{
				"bucket_name": "bucket", "role_based_auth": true, "enable_sse": false,
				"iam_role_arn": "arn:aws:iam::123456789012:role/rudder",
			},
		},
		{
			// DEX-661: a secret the API returns empty is unset, not stripped,
			// so it is dropped and gets no reference.
			name:    "http bearer token returned empty is dropped",
			apiType: "HTTP",
			config:  `{"apiUrl":"https://example.com/hook","auth":"bearerTokenAuth","method":"POST","format":"JSON","bearerToken":""}`,
			expected: map[string]any{
				"api_url": "https://example.com/hook", "auth": "bearerTokenAuth", "method": "POST", "format": "JSON",
			},
		},
		{
			name:    "http bearer token returned null is dropped",
			apiType: "HTTP",
			config:  `{"apiUrl":"https://example.com/hook","auth":"bearerTokenAuth","method":"POST","format":"JSON","bearerToken":null}`,
			expected: map[string]any{
				"api_url": "https://example.com/hook", "auth": "bearerTokenAuth", "method": "POST", "format": "JSON",
			},
		},
		{
			// Connecting a warehouse source needs rest_api_key in cloud mode,
			// the mode a warehouse source runs in when connection_mode leaves
			// it out (V-C5).
			name:     "braze rest api key a warehouse source needs, with no connection mode stored",
			apiType:  "BRAZE",
			config:   `{"dataCenter":"US-01"}`,
			contains: map[string]any{"rest_api_key": "{{ .DEST_REST_API_KEY }}"},
		},
		{
			name:     "facebook pixel access token a warehouse source needs",
			apiType:  "FACEBOOK_PIXEL",
			config:   `{"connectionMode":{"warehouse":"cloud"}}`,
			contains: map[string]any{"access_token": "{{ .DEST_ACCESS_TOKEN }}"},
		},
		{
			name:    "amplitude api secret is optional and stays out",
			apiType: "AM",
			config:  `{"apiKey":"key","residencyServer":"standard"}`,
			expected: map[string]any{
				"api_key": "{{ .DEST_API_KEY }}", "residency_server": "standard",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := destination.NewHandler(nil, registry)
			entities, _, err := h.Impl.FormatForExport(map[string]*destination.RemoteDestination{
				"dest": {Destination: &client.Destination{
					ID: "dst-1", Name: "Dest", Type: tc.apiType, Version: 1, IsEnabled: true, Config: []byte(tc.config),
				}},
			}, nil, stubResolver{})
			require.NoError(t, err)
			require.Len(t, entities, 1)

			spec, ok := entities[0].Content.(*specs.Spec)
			require.True(t, ok)
			if tc.contains == nil {
				assert.Equal(t, tc.expected, spec.Spec["config"])
				return
			}
			config, ok := spec.Spec["config"].(map[string]any)
			require.True(t, ok)
			for key, value := range tc.contains {
				assert.Equal(t, value, config[key], key)
			}
		})
	}
}
