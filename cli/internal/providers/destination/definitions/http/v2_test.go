package http_test

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/testutil"
)

func registeredHTTPV2(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewV2Definition()))
	def, err := registry.Get("http", 2)
	require.NoError(t, err)
	return def
}

func v2MinimalConfig() map[string]any {
	return map[string]any{
		"api_url":               "https://example.com/webhook",
		"auth":                  "noAuth",
		"method":                "POST",
		"request_body_template": "[$$.events][0]",
		"max_batch_size":        "1",
	}
}

func TestHTTPV2RequiresBatchSizeForJSON(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	delete(config, "max_batch_size")
	errors := def.ValidateConfig(config)
	require.Len(t, errors, 1)
	assert.Equal(t, "/max_batch_size", errors[0].Path)
}

func TestHTTPV2AllowsFormWithoutBatchSize(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["format"] = "FORM"
	delete(config, "max_batch_size")
	assert.Empty(t, def.ValidateConfig(config))
}

func TestHTTPV2RequiresBatchSizeForXML(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["format"] = "XML"
	delete(config, "max_batch_size")
	errors := def.ValidateConfig(config)
	require.Len(t, errors, 1)
	assert.Equal(t, "/max_batch_size", errors[0].Path)
}

func TestHTTPV2SupportsFiveThousandXMLBatch(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["format"] = "XML"
	config["max_batch_size"] = "5000"
	config["xml_root_key"] = "Envelope"
	assert.Empty(t, def.ValidateConfig(config))
}

func TestHTTPV2AllowsStoredFormBatchSize(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["format"] = "FORM"
	config["max_batch_size"] = "5000"
	assert.Empty(t, def.ValidateConfig(config))
}

func TestHTTPV2RejectsBatchSizeAboveFiveThousand(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["max_batch_size"] = "5001"
	errors := def.ValidateConfig(config)
	require.Len(t, errors, 1)
	assert.Equal(t, "/max_batch_size", errors[0].Path)
}

func TestHTTPV2RejectsInvalidFormat(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["format"] = "SOAP"
	errors := def.ValidateConfig(config)
	require.NotEmpty(t, errors)
	assertValidationPaths(t, errors, "/format")
}

func TestHTTPV2RejectsOverlongXMLRootKey(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["format"] = "XML"
	config["xml_root_key"] = stringOfLength(101)
	errors := def.ValidateConfig(config)
	require.Len(t, errors, 1)
	assert.Equal(t, "/xml_root_key", errors[0].Path)
}

func TestHTTPV2RequiresBodyTemplate(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	delete(config, "request_body_template")
	errors := def.ValidateConfig(config)
	require.Len(t, errors, 1)
	assert.Equal(t, "/request_body_template", errors[0].Path)
}

func TestHTTPV2RejectsLegacyMappings(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := v2MinimalConfig()
	config["properties_mapping"] = []any{}
	errors := def.ValidateConfig(config)
	require.NotEmpty(t, errors)
	assert.Equal(t, "/properties_mapping", errors[0].Path)
}

func TestHTTPV2XMLConversion(t *testing.T) {
	t.Parallel()
	testutil.AssertConversion(t, httpdest.NewV2Definition().Properties, []testutil.ConversionCase{{
		Name:      "xml request body template",
		LocalJSON: `{"api_url":"https://example.com/webhook","auth":"noAuth","method":"POST","format":"XML","xml_root_key":"Envelope","request_body_template":"{\"event\": $$.events}","max_batch_size":"5000"}`,
		APIJSON:   `{"apiUrl":"https://example.com/webhook","auth":"noAuth","method":"POST","format":"XML","xmlRootKey":"Envelope","requestBodyTemplate":"{\"event\": $$.events}","maxBatchSize":"5000"}`,
	}})
}

func TestHTTPV2FormConversionWithoutBatchSize(t *testing.T) {
	t.Parallel()
	testutil.AssertConversion(t, httpdest.NewV2Definition().Properties, []testutil.ConversionCase{{
		Name:      "form without max batch size",
		LocalJSON: `{"api_url":"https://example.com/webhook","auth":"noAuth","method":"POST","format":"FORM","request_body_template":"[$$.events][0]"}`,
		APIJSON:   `{"apiUrl":"https://example.com/webhook","auth":"noAuth","method":"POST","format":"FORM","requestBodyTemplate":"[$$.events][0]"}`,
	}})
}

func TestHTTPV2RemoteMatchesSpec(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		config map[string]any
		remote string
	}{
		{
			name: "XML",
			config: map[string]any{
				"format": "XML", "xml_root_key": "Envelope", "max_batch_size": "1",
			},
			remote: `{"apiUrl":"https://example.com/webhook","auth":"noAuth","method":"POST","format":"XML","xmlRootKey":"Envelope","requestBodyTemplate":"[$$.events][0]","maxBatchSize":"1"}`,
		},
		{
			name:   "FORM",
			config: map[string]any{"format": "FORM"},
			remote: `{"apiUrl":"https://example.com/webhook","auth":"noAuth","method":"POST","format":"FORM","requestBodyTemplate":"[$$.events][0]"}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			registry := definitions.NewRegistry()
			require.NoError(t, registry.Register(httpdest.NewV2Definition()))
			h := destination.NewHandler(nil, registry)
			config := v2MinimalConfig()
			delete(config, "max_batch_size")
			for key, value := range tc.config {
				config[key] = value
			}
			id := "http-" + tc.name
			resource, err := h.Impl.ExtractResourcesFromSpec("http-v2.yaml", &destination.DestinationSpec{
				ID: id, DisplayName: "HTTP " + tc.name, Type: "http", Enabled: true,
				DefinitionVersion: lo.ToPtr[int64](2), Config: config,
			})
			require.NoError(t, err)
			remote, _, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{Destination: &client.Destination{
				ID: "remote-http", ExternalID: id, Name: "HTTP " + tc.name, Type: "HTTP", Version: 2, IsEnabled: true,
				Config: []byte(tc.remote),
			}}, urnResolver{})
			require.NoError(t, err)
			assert.Equal(t, resource[id], remote)
		})
	}
}

func TestHTTPV2Defaults(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	assert.Equal(t, map[string]any{"format": "JSON"}, def.ConfigDefaults())
	assert.Empty(t, def.ValidateConfig(v2MinimalConfig()))
	assert.Equal(t, "JSON", def.ApplyDefaults(v2MinimalConfig())["format"])
}
