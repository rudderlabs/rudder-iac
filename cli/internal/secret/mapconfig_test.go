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

// Secret paths carry no container information — the walkers dispatch on the
// runtime type at each step: maps descend into the one nested object, slices
// fan the remaining path out to every member. These cases pin that dispatch
// across object nesting, typed object slices, and heterogeneous deep shapes.
func TestMapConfigMapAndSliceContainerShapes(t *testing.T) {
	t.Run("object nesting descends without fan-out", func(t *testing.T) {
		config := map[string]any{
			"auth":        map[string]any{"token": "map-secret", "kind": "bearer"},
			"webhook_url": "https://webhooks.example.com/rudder",
		}

		wrapped := WrapKnownSecrets(config, []string{"auth.token"})
		auth, ok := wrapped["auth"].(map[string]any)
		require.True(t, ok)
		require.IsType(t, &String{}, auth["token"])
		assert.Equal(t, "map-secret", auth["token"].(*String).Reveal())
		assert.Equal(t, "bearer", auth["kind"], "sibling keys stay untouched")

		revealed := RevealSecrets(wrapped, []string{"auth.token"})
		assert.Equal(t, "map-secret", revealed["auth"].(map[string]any)["token"])
		assert.IsType(t, &String{}, wrapped["auth"].(map[string]any)["token"], "RevealSecrets must not mutate caller config")

		unknown := WrapUnknownSecrets(revealed, []string{"auth.token"})
		require.IsType(t, &String{}, unknown["auth"].(map[string]any)["token"])
		assert.True(t, unknown["auth"].(map[string]any)["token"].(*String).IsUnknown())

		enableVarSubstitution(t)
		require.NoError(t, MaskSecrets(unknown, "webhook-prod", []string{"auth.token"}))
		assert.Equal(t, "{{ .WEBHOOK_PROD_AUTH_TOKEN }}", unknown["auth"].(map[string]any)["token"], "map descent adds no index to the variable name")
	})

	t.Run("typed object slices walk like generic slices", func(t *testing.T) {
		config := map[string]any{
			"headers": []map[string]any{
				{"from": "X-Api-Key", "to": "typed-secret-one"},
				{"from": "X-Trace", "to": "typed-secret-two"},
			},
		}

		WrapKnownSecrets(config, []string{"headers.to"})
		headers, ok := config["headers"].([]map[string]any)
		require.True(t, ok, "wrapping must not reshape the typed slice")
		assert.Equal(t, "typed-secret-one", headers[0]["to"].(*String).Reveal())
		assert.Equal(t, "typed-secret-two", headers[1]["to"].(*String).Reveal())

		revealed := RevealSecrets(config, []string{"headers.to"})
		assert.Equal(t, "typed-secret-one", revealed["headers"].([]map[string]any)[0]["to"])
		assert.IsType(t, &String{}, headers[0]["to"], "RevealSecrets must not mutate caller config")

		enableVarSubstitution(t)
		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
		assert.Equal(t, "{{ .WEBHOOK_PROD_HEADERS_0_TO }}", headers[0]["to"])
		assert.Equal(t, "{{ .WEBHOOK_PROD_HEADERS_1_TO }}", headers[1]["to"])
		assert.Equal(t, "X-Api-Key", headers[0]["from"])
	})

	t.Run("heterogeneous deep shapes resolve per member", func(t *testing.T) {
		// One path, three leaves: a[0].b is itself a slice, a[1].b is an object.
		config := map[string]any{
			"a": []any{
				map[string]any{"b": []any{
					map[string]any{"c": "deep-1"},
					map[string]any{"c": "deep-2"},
				}},
				map[string]any{"b": map[string]any{"c": "deep-3"}},
			},
		}

		WrapKnownSecrets(config, []string{"a.b.c"})
		first := config["a"].([]any)[0].(map[string]any)["b"].([]any)
		second := config["a"].([]any)[1].(map[string]any)["b"].(map[string]any)
		assert.Equal(t, "deep-1", first[0].(map[string]any)["c"].(*String).Reveal())
		assert.Equal(t, "deep-2", first[1].(map[string]any)["c"].(*String).Reveal())
		assert.Equal(t, "deep-3", second["c"].(*String).Reveal())

		enableVarSubstitution(t)
		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"a.b.c"}))
		assert.Equal(t, "{{ .WEBHOOK_PROD_A_0_B_0_C }}", first[0].(map[string]any)["c"])
		assert.Equal(t, "{{ .WEBHOOK_PROD_A_0_B_1_C }}", first[1].(map[string]any)["c"])
		assert.Equal(t, "{{ .WEBHOOK_PROD_A_1_B_C }}", second["c"], "map hop contributes no index between the slice indices")
	})
}

// "." in a secret key is purely a path separator — there is no escape syntax,
// since upstream secret keys are camelCase identifiers that never contain dots.
// These cases pin how malformed paths and non-object array members are handled:
// everything is left untouched, nothing panics.
func TestMapConfigSecretPathShapes(t *testing.T) {
	t.Run("empty path segments match nothing", func(t *testing.T) {
		config := map[string]any{
			"headers": []any{map[string]any{"to": "header-secret"}},
			"":        "empty-key-value",
		}

		for _, key := range []string{"", ".", "headers.", ".to", "headers..to"} {
			WrapKnownSecrets(config, []string{key})
			WrapUnknownSecrets(config, []string{key})
			require.NoError(t, MaskSecrets(config, "webhook-prod", []string{key}))
			assert.Equal(t, config, RevealSecrets(config, []string{key}), key)
		}
		assert.Equal(t, "header-secret", config["headers"].([]any)[0].(map[string]any)["to"])
		assert.Equal(t, "empty-key-value", config[""], "even a literal empty key is never resolved")
	})

	t.Run("keys containing literal dots are not addressable", func(t *testing.T) {
		config := map[string]any{"headers.to": "flat-secret"}

		WrapKnownSecrets(config, []string{"headers.to"})
		assert.Equal(t, "flat-secret", config["headers.to"], "a dotted key is a path, never a literal lookup")

		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
		assert.Equal(t, "flat-secret", config["headers.to"])
	})

	t.Run("non-object array members are skipped", func(t *testing.T) {
		config := map[string]any{
			"headers": []any{
				"not-an-object",
				map[string]any{"to": "header-secret"},
				42,
			},
		}

		WrapKnownSecrets(config, []string{"headers.to"})
		headers := config["headers"].([]any)
		assert.Equal(t, "not-an-object", headers[0])
		assert.IsType(t, &String{}, headers[1].(map[string]any)["to"])
		assert.Equal(t, 42, headers[2])

		enableVarSubstitution(t)
		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
		headers = config["headers"].([]any)
		assert.Equal(t, "not-an-object", headers[0])
		assert.Equal(t, "{{ .WEBHOOK_PROD_HEADERS_1_TO }}", headers[1].(map[string]any)["to"])
		assert.Equal(t, 42, headers[2])
	})
}
