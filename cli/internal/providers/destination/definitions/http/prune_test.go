package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions"
	httpdest "github.com/rudderlabs/rudder-iac/cli/internal/providers/destination/definitions/http"
)

func registeredHTTP(t *testing.T) *definitions.RegisteredDefinition {
	t.Helper()
	registry := definitions.NewRegistry()
	require.NoError(t, registry.Register(httpdest.NewDefinition()))
	registered, err := registry.Get("http", 1)
	require.NoError(t, err)
	return registered
}

func TestPruneEmptyOptional_NoAuthDropsAuthFields(t *testing.T) {
	t.Parallel()

	registered := registeredHTTP(t)

	// Mirrors the backend's full-schema response for a noAuth HTTP destination:
	// every field present, most empty.
	full := map[string]any{
		"api_url":             "http://abcd.com",
		"auth":                "noAuth",
		"username":            "",
		"password":            "",
		"bearer_token":        "",
		"api_key_name":        "x-api-key",
		"api_key_value":       "",
		"xml_root_key":        "",
		"method":              "POST",
		"format":              "JSON",
		"properties_mapping":  []any{},
		"query_params":        []any{},
		"headers":             []any{},
		"path_params":         []any{},
		"is_batching_enabled": false,
		"max_batch_size":      "",
		"is_default_mapping":  true,
		"event_filtering": map[string]any{
			"whitelist": []any{""},
			"blacklist": []any{""},
		},
	}

	pruned := registered.PruneEmptyOptional(full)

	// Meaningful fields survive.
	assert.Equal(t, "http://abcd.com", pruned["api_url"])
	assert.Equal(t, "noAuth", pruned["auth"])
	assert.Equal(t, "POST", pruned["method"])
	assert.Equal(t, "JSON", pruned["format"])
	assert.Equal(t, "x-api-key", pruned["api_key_name"], "non-empty optional stays")
	assert.Equal(t, true, pruned["is_default_mapping"], "bool value stays")
	assert.Equal(t, false, pruned["is_batching_enabled"], "bool value stays")

	// Empty, not-required fields are dropped — no phantom secret placeholders.
	for _, key := range []string{
		"username", "password", "bearer_token", "api_key_value",
		"xml_root_key", "max_batch_size",
		"properties_mapping", "query_params", "headers", "path_params",
		"event_filtering",
	} {
		assert.NotContains(t, pruned, key, "empty non-required %q should be pruned", key)
	}

	// The pruned config still validates.
	assert.Empty(t, registered.ValidateConfig(pruned))
}

func TestPruneEmptyOptional_BasicAuthKeepsRequiredEmpties(t *testing.T) {
	t.Parallel()

	registered := registeredHTTP(t)

	// basicAuth makes username/password required; even if the backend returned
	// them empty, they must be kept (they are the fields the user must fill).
	config := map[string]any{
		"api_url":  "https://example.com/hook",
		"auth":     "basicAuth",
		"username": "",
		"password": "",
		"method":   "POST",
		"format":   "JSON",
	}

	pruned := registered.PruneEmptyOptional(config)

	assert.Contains(t, pruned, "username", "required-when-basicAuth field kept even if empty")
	assert.Contains(t, pruned, "password", "required-when-basicAuth field kept even if empty")
}

func TestPruneEmptyOptional_KeepsUserSetOptional(t *testing.T) {
	t.Parallel()

	registered := registeredHTTP(t)

	config := map[string]any{
		"api_url":      "https://example.com/hook",
		"auth":         "noAuth",
		"method":       "POST",
		"format":       "XML",
		"xml_root_key": "rudderEvent", // optional but user-set
		"headers": []any{
			map[string]any{"to": "X-Source", "from": "rudder"},
		},
	}

	pruned := registered.PruneEmptyOptional(config)

	assert.Equal(t, "rudderEvent", pruned["xml_root_key"], "non-empty optional survives")
	assert.Contains(t, pruned, "headers", "non-empty array survives")
}

func TestPruneEmptyOptional_BatchingRequiresMaxBatchSize(t *testing.T) {
	t.Parallel()

	registered := registeredHTTP(t)

	// is_batching_enabled=true makes max_batch_size required_if; keep it even
	// if empty so the user sees the field they must fill.
	config := map[string]any{
		"api_url":             "https://example.com/hook",
		"auth":                "noAuth",
		"method":              "POST",
		"format":              "JSON",
		"is_batching_enabled": true,
		"max_batch_size":      "",
	}

	pruned := registered.PruneEmptyOptional(config)

	assert.Contains(t, pruned, "max_batch_size", "required_if=IsBatchingEnabled true kept even if empty")
}
