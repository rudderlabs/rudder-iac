package configpath

import (
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("reads top-level and nested values", func(t *testing.T) {
		t.Parallel()

		config := map[string]any{
			"top": "value",
			"s3":  map[string]any{"access_key_id": "nested-value"},
		}

		value, ok, err := Get(config, "top")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "value", value)

		value, ok, err = Get(config, "s3.access_key_id")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "nested-value", value)
	})

	t.Run("reports missing values without mutating", func(t *testing.T) {
		t.Parallel()

		config := map[string]any{"s3": map[string]any{"bucket": "b"}}

		value, ok, err := Get(config, "s3.access_key")
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Nil(t, value)
		assert.Equal(t, map[string]any{"s3": map[string]any{"bucket": "b"}}, config)
	})

	t.Run("rejects empty and numeric segments", func(t *testing.T) {
		t.Parallel()

		for _, path := range []string{"", "s3..access_key", "s3.0.access_key"} {
			_, _, err := Get(map[string]any{}, path)
			require.Error(t, err)
		}
	})
}

func TestSet(t *testing.T) {
	t.Parallel()

	t.Run("creates missing parent maps and preserves secret pointers", func(t *testing.T) {
		t.Parallel()

		secretValue := secret.New("secret")
		config := map[string]any{"top": "value"}

		err := Set(config, "s3.access_key", &secretValue)
		require.NoError(t, err)

		parent, ok := config["s3"].(map[string]any)
		require.True(t, ok)
		assert.Same(t, &secretValue, parent["access_key"])
		assert.Equal(t, "value", config["top"])
	})

	t.Run("rejects non-map parents", func(t *testing.T) {
		t.Parallel()

		config := map[string]any{"s3": "not-a-map"}

		err := Set(config, "s3.access_key", "secret")
		require.Error(t, err)
		assert.Equal(t, map[string]any{"s3": "not-a-map"}, config)
	})
}

func TestSetCopyOnWrite(t *testing.T) {
	t.Parallel()

	t.Run("clones only maps on the edited path", func(t *testing.T) {
		t.Parallel()

		shared := map[string]any{"other": "unchanged"}
		parent := map[string]any{
			"access_key": "old",
			"shared":     shared,
		}
		config := map[string]any{
			"s3":  parent,
			"top": "value",
		}

		out, err := SetCopyOnWrite(config, "s3.access_key", "new")
		require.NoError(t, err)

		assert.Equal(t, "old", parent["access_key"])
		outParent, ok := out["s3"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "new", outParent["access_key"])
		parent["access_key"] = "changed-after-write"
		assert.Equal(t, "new", outParent["access_key"], "edited ancestors must be cloned")
		shared["other"] = "changed-after-write"
		assert.Equal(t, "changed-after-write", outParent["shared"].(map[string]any)["other"], "off-path maps stay shared")
		assert.Equal(t, "value", out["top"])
	})

	t.Run("returns original map unchanged when path cannot be traversed", func(t *testing.T) {
		t.Parallel()

		config := map[string]any{"s3": "not-a-map"}

		out, err := SetCopyOnWrite(config, "s3.access_key", "new")
		require.Error(t, err)
		config["s3"] = "changed-after-error"
		assert.Equal(t, "changed-after-error", out["s3"], "error path must return the original map")
		assert.Equal(t, map[string]any{"s3": "changed-after-error"}, config)
	})
}
