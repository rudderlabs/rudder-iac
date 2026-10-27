package http_test

import (
	"encoding/json"
	"testing"

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
	require.NoError(t, registry.Register(httpdest.NewDefinitionV2()))
	def, err := registry.Get("http", 2)
	require.NoError(t, err)
	return def
}

func validV2Config() map[string]any {
	return map[string]any{
		"api_url":               "https://example.com/webhook",
		"auth":                  "noAuth",
		"method":                "POST",
		"format":                "JSON",
		"request_body_template": `$$.events.{"customer": {"id": userId}, "items": properties.items}`,
	}
}

func TestHTTPV2Conversion(t *testing.T) {
	t.Parallel()
	testutil.AssertConversion(t, httpdest.NewDefinitionV2().Properties, []testutil.ConversionCase{{
		Name: "JSONata request and filtering",
		LocalJSON: `{
			"api_url": "$$.event.context.endpoint",
			"auth": "noAuth", "method": "POST", "format": "JSON",
			"request_body_template": "$$.events.{\"id\": userId}",
			"supported_message_types": ["track", "identify"],
			"headers": [{"to": "X-Event", "from": "$.type"}],
			"query_params": [{"to": "source", "from": "rudderstack"}],
			"path_params": [{"path": "$.properties[\"account-id\"]"}],
			"is_batching_enabled": true, "max_batch_size": "5000",
			"connection_mode": {"web": "cloud"}
		}`,
		APIJSON: `{
			"apiUrl": "$$.event.context.endpoint",
			"auth": "noAuth", "method": "POST", "format": "JSON",
			"requestBodyTemplate": "$$.events.{\"id\": userId}",
			"supportedMessageTypes": ["track", "identify"],
			"headers": [{"to": "X-Event", "from": "$.type"}],
			"queryParams": [{"to": "source", "from": "rudderstack"}],
			"pathParams": [{"path": "$.properties[\"account-id\"]"}],
			"isBatchingEnabled": true, "maxBatchSize": "5000",
			"connectionMode": {"web": "cloud"}
		}`,
	}})
}

func TestHTTPV2HandlerSpecAndRemoteRoundTrip(t *testing.T) {
	t.Parallel()
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))
	require.NoError(t, registry.Register(httpdest.NewDefinitionV2()))
	h := destination.NewHandler(nil, registry)
	local := validV2Config()
	local["supported_message_types"] = []any{"track", "identify"}
	local["connection_mode"] = map[string]any{"web": "cloud"}
	extracted, err := h.Impl.ExtractResourcesFromSpec("http-v2.yaml", &destination.DestinationSpec{
		ID: "http-v2", DisplayName: "HTTP v2", Type: "http", Enabled: true,
		DefinitionVersion: 2, Config: local,
	})
	require.NoError(t, err)
	want := registeredHTTPV2(t).ApplyDefaults(local)
	assert.Equal(t, want, extracted["http-v2"].Config)
	apiConfig, err := registry.Get("http", 2)
	require.NoError(t, err)
	wire, err := apiConfig.LocalToAPI(want)
	require.NoError(t, err)
	payload, err := json.Marshal(wire)
	require.NoError(t, err)
	remote, _, err := h.Impl.MapRemoteToState(&destination.RemoteDestination{Destination: &client.Destination{
		ID: "remote-http-v2", ExternalID: "http-v2", Name: "HTTP v2", Type: "HTTP", Version: 2,
		IsEnabled: true, Config: payload,
	}}, urnResolver{})
	require.NoError(t, err)
	assert.Equal(t, want, remote.Config)
	assert.Equal(t, int64(2), remote.DefinitionVersion)
}

func TestHTTPV2RejectsV1Mapping(t *testing.T) {
	t.Parallel()
	config := validV2Config()
	config["properties_mapping"] = []any{map[string]any{"to": "$.customer", "from": "$.userId"}}
	errors := registeredHTTPV2(t).ValidateConfig(config)
	require.Len(t, errors, 1)
	assert.Equal(t, "/properties_mapping", errors[0].Path)
}

func TestHTTPV2RequestFieldExpressionsAndDefaults(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	config := validV2Config()
	config["api_url"] = `$$.event.context.apiUrl`
	config["headers"] = []any{map[string]any{"to": "X-Order", "from": `$$.event.properties.{"id": orderId}`}}
	config["query_params"] = []any{map[string]any{"to": "filter", "from": `$.properties["order-id"]`}}
	config["path_params"] = []any{map[string]any{"path": `$.properties["order-id"]`}}
	assert.Empty(t, def.ValidateConfig(config))
	assert.Equal(t, map[string]any{"is_batching_enabled": false}, def.ConfigDefaults())
	assert.Equal(t, "HTTP", def.APIType)
	assert.Equal(t, int64(2), def.Version)
	assert.Equal(t, httpdest.NewDefinition().SecretKeys, def.SecretKeys())
}

func TestHTTPV2RequiredFieldsAndTypes(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	for _, tc := range []struct {
		key   string
		value any
		path  string
	}{
		{"request_body_template", "", "/request_body_template"},
		{"supported_message_types", []any{"purchase"}, "/supported_message_types/0"},
		{"format", "XML", "/format"},
	} {
		config := validV2Config()
		config[tc.key] = tc.value
		errors := def.ValidateConfig(config)
		require.NotEmpty(t, errors, tc.key)
		assert.Equal(t, tc.path, errors[0].Path)
	}
}

func TestHTTPV2RejectsInvalidHeaderName(t *testing.T) {
	t.Parallel()
	config := validV2Config()
	config["headers"] = []any{map[string]any{"to": "Bad Header", "from": "value"}}
	errors := registeredHTTPV2(t).ValidateConfig(config)
	require.NotEmpty(t, errors)
	assert.Equal(t, "/headers/0/to", errors[0].Path)
}

func TestHTTPV2RejectsInvalidQueryParamName(t *testing.T) {
	t.Parallel()
	config := validV2Config()
	config["query_params"] = []any{map[string]any{"to": "$name", "from": "value"}}
	errors := registeredHTTPV2(t).ValidateConfig(config)
	require.NotEmpty(t, errors)
	assert.Equal(t, "/query_params/0/to", errors[0].Path)
}

func TestHTTPV2RejectsDuplicateMessageTypes(t *testing.T) {
	t.Parallel()
	config := validV2Config()
	config["supported_message_types"] = []any{"track", "track"}
	errors := registeredHTTPV2(t).ValidateConfig(config)
	require.NotEmpty(t, errors)
	assert.Equal(t, "/supported_message_types", errors[0].Path)
}

func TestHTTPV2AllowsMigratedEmptyRequestFields(t *testing.T) {
	t.Parallel()
	config := validV2Config()
	config["headers"] = []any{map[string]any{"to": "", "from": ""}}
	config["query_params"] = []any{map[string]any{"to": "", "from": ""}}
	config["path_params"] = []any{map[string]any{"path": ""}}
	assert.Empty(t, registeredHTTPV2(t).ValidateConfig(config))
}

func TestHTTPV2BatchSizeBounds(t *testing.T) {
	t.Parallel()
	def := registeredHTTPV2(t)
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{"1", true}, {"100", true}, {"999", true}, {"1000", true}, {"4999", true}, {"5000", true},
		{"0", false}, {"5001", false}, {"9999", false}, {"01", false}, {"1.5", false},
	} {
		config := validV2Config()
		config["is_batching_enabled"] = true
		config["max_batch_size"] = tc.value
		errors := def.ValidateConfig(config)
		if tc.valid {
			assert.Empty(t, errors, tc.value)
		} else {
			require.NotEmpty(t, errors, tc.value)
			assert.Equal(t, "/max_batch_size", errors[0].Path)
		}
	}
}
