package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/api/client"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetViper(t *testing.T) {
	t.Helper()
	viper.Reset()
}

func initTestConfig(t *testing.T) string {
	t.Helper()
	resetViper(t)
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	InitConfig(cfgFile)
	return cfgFile
}

func TestDefaultConfigFile(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(homeDir, ".rudder", "config.json"), DefaultConfigFile())
}

func TestInitConfig_CreatesConfigFile(t *testing.T) {
	resetViper(t)

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	_, err := os.Stat(cfgFile)
	require.ErrorIs(t, err, os.ErrNotExist)

	InitConfig(cfgFile)

	_, err = os.Stat(cfgFile)
	require.NoError(t, err)
}

func TestGetConfig_Defaults(t *testing.T) {
	initTestConfig(t)

	cfg := GetConfig()

	assert.Equal(t, Config{
		Debug:   false,
		Verbose: false,
		APIURL:  client.BASE_URL,
		Auth: struct {
			AccessToken string `mapstructure:"accessToken"`
		}{},
		Telemetry: struct {
			Disabled     bool   `mapstructure:"disabled"`
			AnonymousID  string `mapstructure:"anonymousId"`
			WriteKey     string `mapstructure:"writeKey"`
			DataplaneURL string `mapstructure:"dataplaneURL"`
		}{
			Disabled:     false,
			WriteKey:     TelemetryWriteKey,
			DataplaneURL: TelemetryDataplaneURL,
		},
		ExperimentalFlags: ExperimentalConfig{},
		Concurrency: struct {
			Syncer            int `mapstructure:"syncer"`
			CatalogClient     int `mapstructure:"catalogClient"`
			CompositeProvider int `mapstructure:"compositeProvider"`
			CatalogProvider   int `mapstructure:"catalogProvider"`
			DataGraph         int `mapstructure:"dataGraph"`
		}{
			Syncer:            30,
			CatalogClient:     10,
			CompositeProvider: 2,
			CatalogProvider:   4,
			DataGraph:         4,
		},
	}, cfg)
}

func TestGetConfigDir(t *testing.T) {
	cfgFile := initTestConfig(t)

	assert.Equal(t, filepath.Dir(cfgFile), GetConfigDir())
}

func TestSetAccessToken(t *testing.T) {
	cfgFile := initTestConfig(t)

	SetAccessToken("test-token")

	cfg := GetConfig()
	assert.Equal(t, "test-token", cfg.Auth.AccessToken)

	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)

	var persisted map[string]any
	require.NoError(t, json.Unmarshal(data, &persisted))

	auth, ok := persisted["auth"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-token", auth["accessToken"])
}

func TestSetTelemetryDisabled(t *testing.T) {
	cfgFile := initTestConfig(t)

	SetTelemetryDisabled(true)

	cfg := GetConfig()
	assert.True(t, cfg.Telemetry.Disabled)

	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)

	var persisted map[string]any
	require.NoError(t, json.Unmarshal(data, &persisted))

	telemetry, ok := persisted["telemetry"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, telemetry["disabled"])
}

func TestSetTelemetryAnonymousID(t *testing.T) {
	cfgFile := initTestConfig(t)

	SetTelemetryAnonymousID("anon-123")

	cfg := GetConfig()
	assert.Equal(t, "anon-123", cfg.Telemetry.AnonymousID)

	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)

	var persisted map[string]any
	require.NoError(t, json.Unmarshal(data, &persisted))

	telemetry, ok := persisted["telemetry"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "anon-123", telemetry["anonymousID"])
}

func TestSetExperimentalFlag(t *testing.T) {
	cfgFile := initTestConfig(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")

	SetExperimentalFlag("transformations", true)

	cfg := GetConfig()
	assert.True(t, cfg.ExperimentalFlags.Transformations)

	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)

	var persisted map[string]any
	require.NoError(t, json.Unmarshal(data, &persisted))

	flags, ok := persisted["flags"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, flags["transformations"])
}

func TestResetExperimentalFlags(t *testing.T) {
	cfgFile := initTestConfig(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")

	SetExperimentalFlag("transformations", true)
	ResetExperimentalFlags()

	cfg := GetConfig()
	assert.Equal(t, ExperimentalConfig{}, cfg.ExperimentalFlags)

	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)

	var persisted map[string]any
	require.NoError(t, json.Unmarshal(data, &persisted))
	_, hasFlags := persisted["flags"]
	assert.False(t, hasFlags)
}

func TestGetConfig_ClearsExperimentalFlagsWhenDisabled(t *testing.T) {
	initTestConfig(t)
	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "true")

	SetExperimentalFlag("transformations", true)
	require.True(t, GetConfig().ExperimentalFlags.Transformations)

	t.Setenv("RUDDERSTACK_CLI_EXPERIMENTAL", "false")
	viper.Set("experimental", false)

	cfg := GetConfig()
	assert.Equal(t, ExperimentalConfig{}, cfg.ExperimentalFlags)
}
