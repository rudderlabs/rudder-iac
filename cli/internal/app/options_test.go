package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
)

func setExperimental(t *testing.T, enabled bool) {
	t.Helper()
	prevExp := viper.Get("experimental")
	prevFlag := viper.Get("flags.enableVarSubstitution")
	viper.Set("experimental", enabled)
	viper.Set("flags.enableVarSubstitution", enabled)
	t.Cleanup(func() {
		viper.Set("experimental", prevExp)
		viper.Set("flags.enableVarSubstitution", prevFlag)
	})
}

func writeVarFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.vars.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestNewProjectOptions(t *testing.T) {
	t.Run("flag disabled returns no options", func(t *testing.T) {
		setExperimental(t, false)

		opts, err := NewProjectOptions(config.GetConfig(), []string{"/does/not/matter"})
		require.NoError(t, err)
		assert.Nil(t, opts)
	})

	t.Run("flag enabled with no var files wires substitutor", func(t *testing.T) {
		setExperimental(t, true)

		opts, err := NewProjectOptions(config.GetConfig(), nil)
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("flag enabled with var file wires substitutor", func(t *testing.T) {
		setExperimental(t, true)
		path := writeVarFile(t, "FOO: bar")

		opts, err := NewProjectOptions(config.GetConfig(), []string{path})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("missing var file surfaces ErrVarFileNotFound", func(t *testing.T) {
		setExperimental(t, true)
		path := filepath.Join(t.TempDir(), "missing.vars.yaml")

		opts, err := NewProjectOptions(config.GetConfig(), []string{path})
		require.Error(t, err)
		assert.Nil(t, opts)
		assert.ErrorIs(t, err, resolver.ErrVarFileNotFound)
	})

	t.Run("var file without .vars.yaml suffix surfaces ErrVarFileInvalidName", func(t *testing.T) {
		setExperimental(t, true)
		path := filepath.Join(t.TempDir(), "plain.yaml")
		require.NoError(t, os.WriteFile(path, []byte("FOO: bar"), 0644))

		opts, err := NewProjectOptions(config.GetConfig(), []string{path})
		require.Error(t, err)
		assert.Nil(t, opts)
		assert.ErrorIs(t, err, resolver.ErrVarFileInvalidName)
	})

	t.Run("invalid var file surfaces ErrVarFileParseFailed", func(t *testing.T) {
		setExperimental(t, true)
		path := writeVarFile(t, "{{ not yaml")

		opts, err := NewProjectOptions(config.GetConfig(), []string{path})
		require.Error(t, err)
		assert.Nil(t, opts)
		assert.ErrorIs(t, err, resolver.ErrVarFileParseFailed)
	})
}

func TestBuildSubstitutor_ResolverChain(t *testing.T) {
	t.Run("env resolver is always wired (no var files)", func(t *testing.T) {
		t.Setenv("RUDDER_GREETING", "hello")

		sub, err := buildSubstitutor(nil)
		require.NoError(t, err)

		got, errs := sub.SubstituteBytes([]byte(`{{ .GREETING }}`))
		require.Empty(t, errs)
		assert.Equal(t, "hello", string(got))
	})

	t.Run("env resolver takes priority over file resolver", func(t *testing.T) {
		t.Setenv("RUDDER_NAME", "env-value")
		path := writeVarFile(t, "NAME: file-value")

		sub, err := buildSubstitutor([]string{path})
		require.NoError(t, err)

		got, errs := sub.SubstituteBytes([]byte(`{{ .NAME }}`))
		require.Empty(t, errs)
		assert.Equal(t, "env-value", string(got))
	})

	t.Run("later var file wins over earlier var file", func(t *testing.T) {
		dir := t.TempDir()
		path1 := filepath.Join(dir, "first.vars.yaml")
		path2 := filepath.Join(dir, "second.vars.yaml")
		require.NoError(t, os.WriteFile(path1, []byte("X: first"), 0644))
		require.NoError(t, os.WriteFile(path2, []byte("X: second"), 0644))

		sub, err := buildSubstitutor([]string{path1, path2})
		require.NoError(t, err)

		got, errs := sub.SubstituteBytes([]byte(`{{ .X }}`))
		require.Empty(t, errs)
		assert.Equal(t, "second", string(got))
	})

	t.Run("file resolver supplies values not in env", func(t *testing.T) {
		path := writeVarFile(t, "DB_HOST: db.example.com")

		sub, err := buildSubstitutor([]string{path})
		require.NoError(t, err)

		got, errs := sub.SubstituteBytes([]byte(`{{ .DB_HOST }}`))
		require.Empty(t, errs)
		assert.Equal(t, "db.example.com", string(got))
	})
}
