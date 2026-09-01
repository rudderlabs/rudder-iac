package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rudderlabs/rudder-iac/cli/internal/varsubst/resolver"
)

func writeVarFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.vars.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestNewProjectOptions(t *testing.T) {
	t.Run("no var files wires substitutor", func(t *testing.T) {
		opts, err := NewProjectOptions(nil)
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("var file wires substitutor", func(t *testing.T) {
		path := writeVarFile(t, "FOO: bar")

		opts, err := NewProjectOptions([]string{path})
		require.NoError(t, err)
		assert.Len(t, opts, 1)
	})

	t.Run("missing var file surfaces ErrVarFileNotFound", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.vars.yaml")

		opts, err := NewProjectOptions([]string{path})
		require.Error(t, err)
		assert.Nil(t, opts)
		assert.ErrorIs(t, err, resolver.ErrVarFileNotFound)
	})

	t.Run("var file without .vars.yaml suffix surfaces ErrVarFileInvalidName", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "plain.yaml")
		require.NoError(t, os.WriteFile(path, []byte("FOO: bar"), 0644))

		opts, err := NewProjectOptions([]string{path})
		require.Error(t, err)
		assert.Nil(t, opts)
		assert.ErrorIs(t, err, resolver.ErrVarFileInvalidName)
	})

	t.Run("invalid var file surfaces ErrVarFileParseFailed", func(t *testing.T) {
		path := writeVarFile(t, "{{ not yaml")

		opts, err := NewProjectOptions([]string{path})
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
