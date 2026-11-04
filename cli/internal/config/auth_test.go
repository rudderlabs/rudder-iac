package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Serial: InitConfig mutates the viper singleton.
func TestInitConfig_CreatesPrivateFileAndDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rudder", "config.json")

	InitConfig(path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	info, err = os.Stat(filepath.Dir(path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestInitConfig_TightensExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"auth":{"accessToken":"old"}}`), 0o644))
	require.NoError(t, os.Chmod(path, 0o644))

	InitConfig(path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSetKeychainLogin_RemovesFileToken(t *testing.T) {
	t.Setenv(AccessTokenEnvVar, "")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"auth":{"accessToken":"old"},"debug":true}`), 0o600))
	InitConfig(path)

	SetKeychainLogin("ws-1", "Prod")

	cfg := GetConfig()
	assert.Equal(t, "", cfg.Auth.AccessToken)
	assert.Equal(t, TokenStoreKeychain, cfg.Auth.TokenStore)
	assert.Equal(t, "ws-1", cfg.Auth.WorkspaceID)
	assert.Equal(t, "Prod", cfg.Auth.WorkspaceName)
	assert.True(t, cfg.Debug)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "old")
}

func TestSetFileToken_DropsKeychainMarker(t *testing.T) {
	t.Setenv(AccessTokenEnvVar, "")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"auth":{"tokenStore":"keychain","workspaceID":"ws-1"}}`), 0o600))
	InitConfig(path)

	SetFileToken("new")

	cfg := GetConfig()
	assert.Equal(t, "new", cfg.Auth.AccessToken)
	assert.Equal(t, "", cfg.Auth.TokenStore)
	assert.Equal(t, "", cfg.Auth.WorkspaceID)
}

func TestClearAuth(t *testing.T) {
	t.Setenv(AccessTokenEnvVar, "")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"auth":{"accessToken":"old","tokenStore":"keychain"},"debug":true}`), 0o600))
	InitConfig(path)

	ClearAuth()

	cfg := GetConfig()
	assert.Equal(t, "", cfg.Auth.AccessToken)
	assert.Equal(t, "", cfg.Auth.TokenStore)
	assert.True(t, cfg.Debug)
}
