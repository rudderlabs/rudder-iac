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

func initTestConfig(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	InitConfig(cfgFile)
	return cfgFile
}

func readConfigJSON(t *testing.T, cfgFile string) map[string]any {
	t.Helper()

	data, err := os.ReadFile(cfgFile)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	return parsed
}

func TestInitConfig_createsConfigFileWithDefaults(t *testing.T) {
	cfgFile := initTestConfig(t)

	_, err := os.Stat(cfgFile)
	require.NoError(t, err)

	cfg := GetConfig()
	assert.False(t, cfg.Debug)
	assert.False(t, cfg.Verbose)
	assert.Equal(t, client.BASE_URL, cfg.APIURL)
	assert.Equal(t, 30, cfg.Concurrency.Syncer)
}

func TestGetConfig_clearsExperimentalFlagsWhenGateDisabled(t *testing.T) {
	initTestConfig(t)

	viper.Set("experimental", true)
	viper.Set("flags.transformations", true)

	cfg := GetConfig()
	assert.True(t, cfg.ExperimentalFlags.Transformations)

	viper.Set("experimental", false)
	cfg = GetConfig()
	assert.False(t, cfg.ExperimentalFlags.Transformations)
}

func TestSetAccessToken_persistsToConfigFile(t *testing.T) {
	cfgFile := initTestConfig(t)

	SetAccessToken("test-token")

	parsed := readConfigJSON(t, cfgFile)
	auth, ok := parsed["auth"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test-token", auth["accessToken"])

	cfg := GetConfig()
	assert.Equal(t, "test-token", cfg.Auth.AccessToken)
}

func TestSetTelemetryDisabled_persistsToConfigFile(t *testing.T) {
	cfgFile := initTestConfig(t)

	SetTelemetryDisabled(true)

	parsed := readConfigJSON(t, cfgFile)
	telemetry, ok := parsed["telemetry"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, telemetry["disabled"])

	cfg := GetConfig()
	assert.True(t, cfg.Telemetry.Disabled)
}

func TestSetTelemetryAnonymousID_persistsToConfigFile(t *testing.T) {
	cfgFile := initTestConfig(t)

	SetTelemetryAnonymousID("anon-123")

	parsed := readConfigJSON(t, cfgFile)
	telemetry, ok := parsed["telemetry"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "anon-123", telemetry["anonymousID"])
}

func TestSetExperimentalFlag_persistsValidFlag(t *testing.T) {
	cfgFile := initTestConfig(t)
	viper.Set("experimental", true)

	SetExperimentalFlag("transformations", true)

	parsed := readConfigJSON(t, cfgFile)
	flags, ok := parsed["flags"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, flags["transformations"])
}

func TestResetExperimentalFlags_removesFlagsFromConfigFile(t *testing.T) {
	cfgFile := initTestConfig(t)
	viper.Set("experimental", true)

	SetExperimentalFlag("dataGraph", true)
	ResetExperimentalFlags()

	parsed := readConfigJSON(t, cfgFile)
	_, hasFlags := parsed["flags"]
	assert.False(t, hasFlags)
}

func TestGetConfigDir_returnsConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "nested", "config.json")
	InitConfig(cfgFile)

	assert.Equal(t, filepath.Join(dir, "nested"), GetConfigDir())
}

func TestDefaultConfigFile_endsWithRudderConfigJSON(t *testing.T) {
	path := DefaultConfigFile()
	assert.Contains(t, path, ".rudder")
	assert.Equal(t, "config.json", filepath.Base(path))
}

func TestBindExperimentalFlags_readsFlagFromEnv(t *testing.T) {
	envVar := GetEnvironmentVariableName("transformations")
	t.Setenv(envVar, "true")

	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")
	InitConfig(cfgFile)

	assert.True(t, viper.GetBool("flags.transformations"))
}
