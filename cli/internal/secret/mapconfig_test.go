package secret

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapConfigNestedSecretPaths(t *testing.T) {
	config := map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": "header-secret-one"},
			map[string]any{"from": "X-Trace", "to": "header-secret-two"},
		},
		"webhook_url": "https://webhooks.example.com/rudder",
	}

	wrapped := WrapKnownSecrets(config, []string{"headers.to"})
	headers, ok := wrapped["headers"].([]any)
	require.True(t, ok)
	for i, header := range headers {
		h, ok := header.(map[string]any)
		require.True(t, ok)
		assert.IsType(t, &String{}, h["to"])
		assert.Equal(t, fmt.Sprintf("header-secret-%s", []string{"one", "two"}[i]), h["to"].(*String).Reveal())
		assert.IsType(t, "", h["from"])
	}

	revealed := RevealSecrets(wrapped, []string{"headers.to"})
	revealedHeaders, ok := revealed["headers"].([]any)
	require.True(t, ok)
	assert.Equal(t, "header-secret-one", revealedHeaders[0].(map[string]any)["to"])
	assert.Equal(t, "header-secret-two", revealedHeaders[1].(map[string]any)["to"])
	assert.IsType(t, &String{}, wrapped["headers"].([]any)[0].(map[string]any)["to"], "RevealSecrets must not mutate caller config")

	unknown := WrapUnknownSecrets(revealed, []string{"headers.to"})
	unknownHeaders, ok := unknown["headers"].([]any)
	require.True(t, ok)
	assert.True(t, unknownHeaders[0].(map[string]any)["to"].(*String).IsUnknown())
	assert.True(t, unknownHeaders[1].(map[string]any)["to"].(*String).IsUnknown())

	enableVarSubstitution(t)
	require.NoError(t, MaskSecrets(unknown, "webhook-prod", []string{"headers.to"}))
	maskedHeaders, ok := unknown["headers"].([]any)
	require.True(t, ok)
	assert.Equal(t, "{{ .WEBHOOK_PROD_HEADERS_0_TO }}", maskedHeaders[0].(map[string]any)["to"])
	assert.Equal(t, "{{ .WEBHOOK_PROD_HEADERS_1_TO }}", maskedHeaders[1].(map[string]any)["to"])
	assert.NotEqual(t, maskedHeaders[0].(map[string]any)["to"], maskedHeaders[1].(map[string]any)["to"])
	assert.Equal(t, "X-Api-Key", maskedHeaders[0].(map[string]any)["from"])
	assert.Equal(t, "https://webhooks.example.com/rudder", unknown["webhook_url"])

	payload, err := json.Marshal(unknown)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "header-secret")
}

func TestMapConfigNestedSecretPathsDoNotInventAbsentSecrets(t *testing.T) {
	config := map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Trace"},
		},
	}

	WrapKnownSecrets(config, []string{"headers.to"})
	assert.Equal(t, []any{map[string]any{"from": "X-Trace"}}, config["headers"])

	WrapUnknownSecrets(config, []string{"headers.to"})
	assert.Equal(t, []any{map[string]any{"from": "X-Trace"}}, config["headers"])

	revealed := RevealSecrets(config, []string{"headers.to"})
	assert.Equal(t, config, revealed)

	require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
	assert.Equal(t, []any{map[string]any{"from": "X-Trace"}}, config["headers"])
}
