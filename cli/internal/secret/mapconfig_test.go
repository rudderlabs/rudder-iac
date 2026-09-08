package secret

import (
	"encoding/json"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func knownSecret(v string) *String {
	s := New(v)
	return &s
}

func unknownSecret() *String {
	s := NewUnknown()
	return &s
}

func TestMapConfigNestedSecretPaths(t *testing.T) {
	config := map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": "header-secret-one"},
			map[string]any{"from": "X-Trace", "to": "header-secret-two"},
		},
		"webhook_url": "https://webhooks.example.com/rudder",
	}

	wrapped := WrapKnownSecrets(config, []string{"headers.to"})
	wantWrapped := map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": knownSecret("header-secret-one")},
			map[string]any{"from": "X-Trace", "to": knownSecret("header-secret-two")},
		},
		"webhook_url": "https://webhooks.example.com/rudder",
	}
	assert.Equal(t, wantWrapped, wrapped)

	revealed := RevealSecrets(wrapped, []string{"headers.to"})
	assert.Equal(t, map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": "header-secret-one"},
			map[string]any{"from": "X-Trace", "to": "header-secret-two"},
		},
		"webhook_url": "https://webhooks.example.com/rudder",
	}, revealed)
	assert.Equal(t, wantWrapped, wrapped, "RevealSecrets must not mutate caller config")

	unknown := WrapUnknownSecrets(revealed, []string{"headers.to"})
	assert.Equal(t, map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": unknownSecret()},
			map[string]any{"from": "X-Trace", "to": unknownSecret()},
		},
		"webhook_url": "https://webhooks.example.com/rudder",
	}, unknown)

	require.NoError(t, MaskSecrets(unknown, "webhook-prod", []string{"headers.to"}))
	assert.Equal(t, map[string]any{
		"headers": []any{
			map[string]any{"from": "X-Api-Key", "to": "{{ .WEBHOOK_PROD_HEADERS_0_TO }}"},
			map[string]any{"from": "X-Trace", "to": "{{ .WEBHOOK_PROD_HEADERS_1_TO }}"},
		},
		"webhook_url": "https://webhooks.example.com/rudder",
	}, unknown, "each slice member exports its own indexed variable")

	payload, err := json.Marshal(unknown)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "header-secret")
}

func TestMapConfigNestedSecretPathsDoNotInventAbsentSecrets(t *testing.T) {
	config := map[string]any{
		"headers": []any{map[string]any{"from": "X-Trace"}},
	}
	want := map[string]any{
		"headers": []any{map[string]any{"from": "X-Trace"}},
	}

	assert.Equal(t, want, WrapKnownSecrets(config, []string{"headers.to"}))
	assert.Equal(t, want, WrapUnknownSecrets(config, []string{"headers.to"}))
	assert.Equal(t, want, RevealSecrets(config, []string{"headers.to"}))

	require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
	assert.Equal(t, want, config)
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
		wantWrapped := map[string]any{
			"auth":        map[string]any{"token": knownSecret("map-secret"), "kind": "bearer"},
			"webhook_url": "https://webhooks.example.com/rudder",
		}
		assert.Equal(t, wantWrapped, wrapped, "sibling keys stay untouched")

		revealed := RevealSecrets(wrapped, []string{"auth.token"})
		assert.Equal(t, map[string]any{
			"auth":        map[string]any{"token": "map-secret", "kind": "bearer"},
			"webhook_url": "https://webhooks.example.com/rudder",
		}, revealed)
		assert.Equal(t, wantWrapped, wrapped, "RevealSecrets must not mutate caller config")

		unknown := WrapUnknownSecrets(revealed, []string{"auth.token"})
		assert.Equal(t, map[string]any{
			"auth":        map[string]any{"token": unknownSecret(), "kind": "bearer"},
			"webhook_url": "https://webhooks.example.com/rudder",
		}, unknown)

		require.NoError(t, MaskSecrets(unknown, "webhook-prod", []string{"auth.token"}))
		assert.Equal(t, map[string]any{
			"auth":        map[string]any{"token": "{{ .WEBHOOK_PROD_AUTH_TOKEN }}", "kind": "bearer"},
			"webhook_url": "https://webhooks.example.com/rudder",
		}, unknown, "map descent adds no index to the variable name")
	})

	t.Run("typed object slices walk like generic slices", func(t *testing.T) {
		config := map[string]any{
			"headers": []map[string]any{
				{"from": "X-Api-Key", "to": "typed-secret-one"},
				{"from": "X-Trace", "to": "typed-secret-two"},
			},
		}

		config = WrapKnownSecrets(config, []string{"headers.to"})
		wantWrapped := map[string]any{
			"headers": []map[string]any{
				{"from": "X-Api-Key", "to": knownSecret("typed-secret-one")},
				{"from": "X-Trace", "to": knownSecret("typed-secret-two")},
			},
		}
		assert.Equal(t, wantWrapped, config, "wrapping must not reshape the typed slice")

		revealed := RevealSecrets(config, []string{"headers.to"})
		assert.Equal(t, map[string]any{
			"headers": []map[string]any{
				{"from": "X-Api-Key", "to": "typed-secret-one"},
				{"from": "X-Trace", "to": "typed-secret-two"},
			},
		}, revealed)
		assert.Equal(t, wantWrapped, config, "RevealSecrets must not mutate caller config")

		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
		assert.Equal(t, map[string]any{
			"headers": []map[string]any{
				{"from": "X-Api-Key", "to": "{{ .WEBHOOK_PROD_HEADERS_0_TO }}"},
				{"from": "X-Trace", "to": "{{ .WEBHOOK_PROD_HEADERS_1_TO }}"},
			},
		}, config)
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

		config = WrapKnownSecrets(config, []string{"a.b.c"})
		assert.Equal(t, map[string]any{
			"a": []any{
				map[string]any{"b": []any{
					map[string]any{"c": knownSecret("deep-1")},
					map[string]any{"c": knownSecret("deep-2")},
				}},
				map[string]any{"b": map[string]any{"c": knownSecret("deep-3")}},
			},
		}, config)

		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"a.b.c"}))
		assert.Equal(t, map[string]any{
			"a": []any{
				map[string]any{"b": []any{
					map[string]any{"c": "{{ .WEBHOOK_PROD_A_0_B_0_C }}"},
					map[string]any{"c": "{{ .WEBHOOK_PROD_A_0_B_1_C }}"},
				}},
				map[string]any{"b": map[string]any{"c": "{{ .WEBHOOK_PROD_A_1_B_C }}"}},
			},
		}, config, "a map hop contributes no index between the slice indices")
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
		want := map[string]any{
			"headers": []any{map[string]any{"to": "header-secret"}},
			"":        "empty-key-value",
		}

		for _, key := range []string{"", ".", "headers.", ".to", "headers..to"} {
			assert.Equal(t, want, WrapKnownSecrets(config, []string{key}), key)
			assert.Equal(t, want, WrapUnknownSecrets(config, []string{key}), key)
			assert.Equal(t, want, RevealSecrets(config, []string{key}), key)
			require.NoError(t, MaskSecrets(config, "webhook-prod", []string{key}))
			assert.Equal(t, want, config, "even a literal empty key is never resolved: %q", key)
		}
	})

	t.Run("keys containing literal dots are not addressable", func(t *testing.T) {
		config := map[string]any{"headers.to": "flat-secret"}
		want := map[string]any{"headers.to": "flat-secret"}

		assert.Equal(t, want, WrapKnownSecrets(config, []string{"headers.to"}), "a dotted key is a path, never a literal lookup")

		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
		assert.Equal(t, want, config)
	})

	t.Run("non-object array members are skipped", func(t *testing.T) {
		config := map[string]any{
			"headers": []any{
				"not-an-object",
				map[string]any{"to": "header-secret"},
				42,
			},
		}

		config = WrapKnownSecrets(config, []string{"headers.to"})
		assert.Equal(t, map[string]any{
			"headers": []any{
				"not-an-object",
				map[string]any{"to": knownSecret("header-secret")},
				42,
			},
		}, config)

		require.NoError(t, MaskSecrets(config, "webhook-prod", []string{"headers.to"}))
		assert.Equal(t, map[string]any{
			"headers": []any{
				"not-an-object",
				map[string]any{"to": "{{ .WEBHOOK_PROD_HEADERS_1_TO }}"},
				42,
			},
		}, config)
	})
}

// Callers reach these helpers through a shallow copy of the parsed spec
// (destination's ApplyDefaults uses maps.Copy). A flat key only ever wrote the
// copied top-level map, so the caller was safe by accident; a nested path walks
// into inner containers the shallow copy still shares. The producing helpers
// therefore copy rather than mutate — the caller's spec must survive intact.
func TestMapConfigHelpersDoNotMutateCallerConfig(t *testing.T) {
	// specConfig stands in for the caller's parsed spec; shallowCopy is what
	// ApplyDefaults hands the helpers.
	newSpecConfig := func() map[string]any {
		return map[string]any{
			"api_key": "flat-secret",
			"headers": []any{map[string]any{"from": "X-Api-Key", "to": "nested-secret"}},
			"auth":    map[string]any{"token": "map-secret"},
		}
	}
	shallowCopy := func(in map[string]any) map[string]any {
		out := make(map[string]any, len(in))
		maps.Copy(out, in)
		return out
	}
	keys := []string{"api_key", "headers.to", "auth.token"}

	assertSpecIntact := func(t *testing.T, specConfig map[string]any) {
		t.Helper()
		assert.Equal(t, newSpecConfig(), specConfig, "caller's spec config must be untouched")
	}

	t.Run("WrapKnownSecrets", func(t *testing.T) {
		specConfig := newSpecConfig()
		wrapped := WrapKnownSecrets(shallowCopy(specConfig), keys)

		assertSpecIntact(t, specConfig)
		assert.Equal(t, map[string]any{
			"api_key": knownSecret("flat-secret"),
			"headers": []any{map[string]any{"from": "X-Api-Key", "to": knownSecret("nested-secret")}},
			"auth":    map[string]any{"token": knownSecret("map-secret")},
		}, wrapped)
	})

	t.Run("WrapUnknownSecrets", func(t *testing.T) {
		specConfig := newSpecConfig()
		wrapped := WrapUnknownSecrets(shallowCopy(specConfig), keys)

		assertSpecIntact(t, specConfig)
		assert.Equal(t, map[string]any{
			"api_key": unknownSecret(),
			"headers": []any{map[string]any{"from": "X-Api-Key", "to": unknownSecret()}},
			"auth":    map[string]any{"token": unknownSecret()},
		}, wrapped)
	})

	t.Run("RevealSecrets", func(t *testing.T) {
		var (
			specConfig = WrapKnownSecrets(newSpecConfig(), keys)
			before     = WrapKnownSecrets(newSpecConfig(), keys)
		)

		revealed := RevealSecrets(shallowCopy(specConfig), keys)

		assert.Equal(t, before, specConfig, "caller's config must be untouched")
		assert.Equal(t, newSpecConfig(), revealed)
	})

	// MaskSecrets is the one in-place helper — it returns an error, not a config.
	// Both call sites build the map themselves (apiConfigToLocal / unmarshalOptions)
	// and already mutate it via pruneEmptyValues, so in-place is the contract.
	t.Run("MaskSecrets mutates in place by contract", func(t *testing.T) {
		config := newSpecConfig()
		require.NoError(t, MaskSecrets(config, "webhook-prod", keys))

		assert.Equal(t, map[string]any{
			"api_key": "{{ .WEBHOOK_PROD_API_KEY }}",
			"headers": []any{map[string]any{"from": "X-Api-Key", "to": "{{ .WEBHOOK_PROD_HEADERS_0_TO }}"}},
			"auth":    map[string]any{"token": "{{ .WEBHOOK_PROD_AUTH_TOKEN }}"},
		}, config)
	})
}

// The variable name for a flat key is exactly what it was before dotted paths
// existed: prefix + "_" + upper(key), with no character folding beyond the
// externalID's own kebab-to-underscore. Folding "-" in the key as well would
// turn a loud marshal-time grammar error into a silent rename, and would make
// "api-key" and "api_key" collide on one variable.
func TestMapConfigMaskVariableNameGrammar(t *testing.T) {
	t.Run("flat key keeps the pre-existing name", func(t *testing.T) {
		config := map[string]any{"api_secret": "s"}
		require.NoError(t, MaskSecrets(config, "my-dest", []string{"api_secret"}))
		assert.Equal(t, map[string]any{"api_secret": "{{ .MY_DEST_API_SECRET }}"}, config)
	})

	t.Run("a dash in the key fails at marshal, not silently renames", func(t *testing.T) {
		config := map[string]any{"api-secret": "s"}
		err := MaskSecrets(config, "my-dest", []string{"api-secret"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not satisfy the variable grammar")
	})
}
