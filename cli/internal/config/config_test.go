package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfigPath(t *testing.T) {
	t.Parallel()

	path := defaultConfigPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, ".rudder")

	// Should contain user home directory
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Contains(t, path, homeDir)
}

func TestDefaultConfigFile(t *testing.T) {
	t.Parallel()

	configFile := DefaultConfigFile()
	assert.NotEmpty(t, configFile)
	assert.Contains(t, configFile, ".rudder")
	assert.Contains(t, configFile, "config.json")
	assert.True(t, strings.HasSuffix(configFile, "config.json"))
}

func TestCreateConfigFileIfNotExists(t *testing.T) {
	cases := []struct {
		name      string
		setupFunc func(tempDir string) string
		expectErr bool
	}{
		{
			name: "CreateNewConfigFile",
			setupFunc: func(tempDir string) string {
				return filepath.Join(tempDir, "new_config.json")
			},
			expectErr: false,
		},
		{
			name: "ConfigFileAlreadyExists",
			setupFunc: func(tempDir string) string {
				configFile := filepath.Join(tempDir, "existing_config.json")
				// Create the file first
				err := os.WriteFile(configFile, []byte("{}"), 0644)
				if err != nil {
					panic(err)
				}
				return configFile
			},
			expectErr: false,
		},
		{
			name: "CreateNestedDirectory",
			setupFunc: func(tempDir string) string {
				return filepath.Join(tempDir, "nested", "dir", "config.json")
			},
			expectErr: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configFile := c.setupFunc(tempDir)

			err := createConfigFileIfNotExists(configFile)

			if c.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.FileExists(t, configFile)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	t.Run("WithCustomConfigFile", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "custom_config.json")

		assert.NotPanics(t, func() {
			InitConfig(configFile)
		})
	})

	t.Run("WithEmptyConfigFile", func(t *testing.T) {
		assert.NotPanics(t, func() {
			InitConfig("")
		})
	})
}

func TestSetAccessToken(t *testing.T) {
	t.Run("UpdateAccessToken", func(t *testing.T) {
		testToken := "test-access-token-123"
		assert.NotPanics(t, func() {
			SetAccessToken(testToken)
		})

		// Just verify the function doesn't panic
		// The actual implementation uses the global viper instance
	})
}

func TestSetTelemetryDisabled(t *testing.T) {
	t.Run("DisableTelemetry", func(t *testing.T) {
		assert.NotPanics(t, func() {
			SetTelemetryDisabled(true)
		})
	})

	t.Run("EnableTelemetry", func(t *testing.T) {
		assert.NotPanics(t, func() {
			SetTelemetryDisabled(false)
		})
	})
}

func TestSetTelemetryAnonymousID(t *testing.T) {
	t.Run("UpdateAnonymousID", func(t *testing.T) {
		testID := "test-anonymous-id-456"
		assert.NotPanics(t, func() {
			SetTelemetryAnonymousID(testID)
		})

		// Just verify the function doesn't panic
		// The actual implementation uses the global viper instance
	})
}

func TestGetConfig(t *testing.T) {
	t.Run("UnmarshalConfig", func(t *testing.T) {
		config := GetConfig()

		// Just verify the function returns a config object without panicking
		assert.NotNil(t, config)
		assert.NotNil(t, config.Auth)
		assert.NotNil(t, config.Telemetry)
	})
}

func TestGetConfigDir(t *testing.T) {
	t.Run("ReturnsConfigDirectory", func(t *testing.T) {
		configDir := GetConfigDir()
		assert.NotEmpty(t, configDir)
	})
}

func TestConfigDefaults(t *testing.T) {
	t.Run("VerifyDefaultValues", func(t *testing.T) {
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "defaults_config.json")

		// Save and clear environment variables that might affect defaults
		envVars := []string{
			"RUDDERSTACK_ACCESS_TOKEN",
			"RUDDERSTACK_API_URL",
			"RUDDERSTACK_CLI_EXPERIMENTAL",
			"RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY",
			"RUDDERSTACK_CLI_TELEMETRY_DATAPLANE_URL",
			"RUDDERSTACK_CLI_TELEMETRY_DISABLED",
		}

		originalEnvs := make(map[string]string)
		for _, envVar := range envVars {
			originalEnvs[envVar] = os.Getenv(envVar)
			os.Unsetenv(envVar)
		}

		defer func() {
			// Restore environment variables
			for envVar, value := range originalEnvs {
				if value != "" {
					os.Setenv(envVar, value)
				}
			}
		}()

		// Initialize config with the temporary file
		// Completely reset viper state to avoid interference from global config
		viper.Reset()

		// Write an empty JSON file to ensure clean state
		err := os.WriteFile(configFile, []byte("{}"), 0644)
		require.NoError(t, err)

		InitConfig(configFile)

		// Verify default values
		assert.False(t, viper.GetBool("debug"))
		assert.False(t, viper.GetBool("experimental"))
		assert.False(t, viper.GetBool("verbose"))
		assert.NotEmpty(t, viper.GetString("apiURL"))
		assert.False(t, viper.GetBool("telemetry.disabled"))
		assert.Equal(t, TelemetryWriteKey, viper.GetString("telemetry.writeKey"))
		assert.Equal(t, TelemetryDataplaneURL, viper.GetString("telemetry.dataplaneURL"))
	})
}

func TestCreateConfigFileIfNotExistsCI(t *testing.T) {
	t.Parallel()

	t.Run("CreatesConfigFileSuccessfully", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test-config.json")

		err := createConfigFileIfNotExists(configFile)
		assert.NoError(t, err)

		// File should exist
		assert.FileExists(t, configFile)
	})

	t.Run("HandlesMissingDirectory", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "subdir", "config.json")

		err := createConfigFileIfNotExists(configFile)
		assert.NoError(t, err)

		// File should exist
		assert.FileExists(t, configFile)
	})

	t.Run("HandlesInvalidPath", func(t *testing.T) {
		t.Parallel()

		// Try to create config in an invalid path
		invalidPath := "/invalid/path/config.json"

		// This should return an error but not panic
		err := createConfigFileIfNotExists(invalidPath)
		assert.Error(t, err)
	})

	t.Run("SkipsExistingFile", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "existing-config.json")

		// Create the file first
		err := os.WriteFile(configFile, []byte(`{"test": true}`), 0644)
		require.NoError(t, err)

		// Should not error when file already exists
		err = createConfigFileIfNotExists(configFile)
		assert.NoError(t, err)

		// File content should be unchanged
		content, err := os.ReadFile(configFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), "test")
	})
}
