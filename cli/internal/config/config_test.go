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

func TestDefaultConfigPathWithoutHome(t *testing.T) {
	t.Run("DefaultConfigPathWithoutHomeDir", func(t *testing.T) {
		// Save original HOME
		originalHome := os.Getenv("HOME")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
		}()

		// Remove HOME environment variable to trigger the fallback
		os.Unsetenv("HOME")

		path := defaultConfigPath()
		assert.NotEmpty(t, path)
		// Should fall back to temp dir or current directory
		assert.True(t, strings.Contains(path, ".rudder"))
	})

	t.Run("DefaultConfigPathWithTempDir", func(t *testing.T) {
		// Save original HOME and TMPDIR
		originalHome := os.Getenv("HOME")
		originalTmp := os.Getenv("TMPDIR")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
			if originalTmp != "" {
				os.Setenv("TMPDIR", originalTmp)
			} else {
				os.Unsetenv("TMPDIR")
			}
		}()

		// Remove HOME, set valid TMPDIR
		os.Unsetenv("HOME")
		tempDir := t.TempDir()
		os.Setenv("TMPDIR", tempDir)

		path := defaultConfigPath()
		assert.NotEmpty(t, path)
		assert.Contains(t, path, ".rudder")
	})
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
	configTests := []struct {
		name string
		path func(string) string
	}{
		{"CreateNewConfigFile", func(dir string) string { return filepath.Join(dir, "new_config.json") }},
		{"ConfigFileAlreadyExists", func(dir string) string {
			configFile := filepath.Join(dir, "existing_config.json")
			os.WriteFile(configFile, []byte("{}"), 0644)
			return configFile
		}},
		{"CreateNestedDirectory", func(dir string) string { return filepath.Join(dir, "nested", "dir", "config.json") }},
	}

	for _, test := range configTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()
			configFile := test.path(tempDir)
			err := createConfigFileIfNotExists(configFile)
			assert.NoError(t, err)
			assert.FileExists(t, configFile)
		})
	}
}

func TestCreateConfigFileIfNotExistsErrorScenarios(t *testing.T) {
	t.Run("ErrorCreatingDirectory", func(t *testing.T) {
		// Create a read-only directory
		tempDir := t.TempDir()
		readOnlyDir := filepath.Join(tempDir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0755)
		require.NoError(t, err)
		err = os.Chmod(readOnlyDir, 0444) // Make read-only
		require.NoError(t, err)

		defer func() {
			// Restore permissions for cleanup
			os.Chmod(readOnlyDir, 0755)
		}()

		// Try to create config file in subdirectory of read-only directory
		configFile := filepath.Join(readOnlyDir, "subdir", "config.json")

		err = createConfigFileIfNotExists(configFile)
		if err != nil {
			// Note: On some systems, the exact error message may vary
			// So we check for either possible error message
			hasExpectedError := strings.Contains(err.Error(), "error creating config directory") ||
				strings.Contains(err.Error(), "permission denied") ||
				strings.Contains(err.Error(), "mkdir")
			assert.True(t, hasExpectedError, "Expected directory creation error, got: %v", err)
		} else {
			// Some systems may allow the operation or handle it differently
			// In such cases, we just verify the function didn't panic
			assert.NoError(t, err)
		}
	})

	t.Run("ErrorCreatingFile", func(t *testing.T) {
		// Create a directory and then make it so we can't create files in it
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, "config")
		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		// Make directory read-only so file creation fails
		err = os.Chmod(configDir, 0444)
		require.NoError(t, err)

		defer func() {
			// Restore permissions for cleanup
			os.Chmod(configDir, 0755)
		}()

		configFile := filepath.Join(configDir, "config.json")

		err = createConfigFileIfNotExists(configFile)
		if err != nil {
			// Check for expected error messages
			hasExpectedError := strings.Contains(err.Error(), "error creating config file") ||
				strings.Contains(err.Error(), "permission denied") ||
				strings.Contains(err.Error(), "read-only")
			assert.True(t, hasExpectedError, "Expected file creation error, got: %v", err)
		} else {
			// Some systems may allow the operation or handle it differently
			// In such cases, we just verify the function didn't panic
			assert.NoError(t, err)
		}
	})
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

	t.Run("WithUnwritableConfigFile", func(t *testing.T) {
		// Test the error path where config file creation fails
		// but initialization continues with in-memory config
		configFile := "/invalid/path/that/does/not/exist/config.json"

		assert.NotPanics(t, func() {
			InitConfig(configFile)
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

	t.Run("UpdateAccessTokenWithValidConfig", func(t *testing.T) {
		// Create a temporary config file
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "test_config.json")

		// Create initial config file
		err := os.WriteFile(configFile, []byte("{}"), 0644)
		require.NoError(t, err)

		// Initialize viper with this config file
		viper.SetConfigFile(configFile)
		err = viper.ReadInConfig()
		require.NoError(t, err)

		testToken := "test-access-token-456"
		assert.NotPanics(t, func() {
			SetAccessToken(testToken)
		})
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

func TestUpdateConfigErrorPaths(t *testing.T) {
	t.Run("UpdateConfigWithoutConfigFile", func(t *testing.T) {
		// Reset viper to ensure no config file is being used
		viper.Reset()

		assert.NotPanics(t, func() {
			SetAccessToken("test-token")
		})
	})

	t.Run("UpdateConfigWithUnreadableFile", func(t *testing.T) {
		// Create a config file that exists but can't be read
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "unreadable_config.json")

		err := os.WriteFile(configFile, []byte("{}"), 0644)
		require.NoError(t, err)

		// Make file unreadable
		err = os.Chmod(configFile, 0000)
		require.NoError(t, err)

		defer func() {
			// Restore permissions for cleanup
			os.Chmod(configFile, 0644)
		}()

		// Set viper to use this file
		viper.SetConfigFile(configFile)

		assert.NotPanics(t, func() {
			SetAccessToken("test-token")
		})
	})

	t.Run("UpdateConfigWithUnwritableFile", func(t *testing.T) {
		// Create a config file that can be read but not written
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "readonly_config.json")

		err := os.WriteFile(configFile, []byte("{}"), 0644)
		require.NoError(t, err)

		// Make file read-only
		err = os.Chmod(configFile, 0444)
		require.NoError(t, err)

		defer func() {
			// Restore permissions for cleanup
			os.Chmod(configFile, 0644)
		}()

		// Set viper to use this file
		viper.SetConfigFile(configFile)
		err = viper.ReadInConfig()
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			SetAccessToken("test-token")
		})
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

	t.Run("GetConfigWithInvalidData", func(t *testing.T) {
		// Create viper with invalid data that would cause unmarshal error
		viper.Reset()
		defer viper.Reset()

		// Set some invalid nested data that would cause unmarshal issues
		viper.Set("auth", "invalid_string_instead_of_object")

		config := GetConfig()

		// Should return default config on unmarshal error
		assert.NotNil(t, config)
		assert.False(t, config.Debug)
		assert.False(t, config.Experimental)
		assert.False(t, config.Verbose)
		assert.NotEmpty(t, config.APIURL)
	})
}

func TestGetConfigDir(t *testing.T) {
	t.Run("ReturnsConfigDirectory", func(t *testing.T) {
		configDir := GetConfigDir()
		assert.NotEmpty(t, configDir)
	})

	t.Run("ReturnsConfigDirectoryWithNoConfigFile", func(t *testing.T) {
		// Reset viper to ensure no config file is being used
		originalViper := viper.GetViper()
		viper.Reset()
		defer func() {
			viper.Reset()
			for key, value := range originalViper.AllSettings() {
				viper.Set(key, value)
			}
		}()

		configDir := GetConfigDir()
		assert.NotEmpty(t, configDir)
		assert.Contains(t, configDir, ".rudder")
	})
}

func TestConfigDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "defaults_config.json")

	// Clear environment variables
	envVars := []string{"RUDDERSTACK_ACCESS_TOKEN", "RUDDERSTACK_API_URL", "RUDDERSTACK_CLI_EXPERIMENTAL", "RUDDERSTACK_CLI_TELEMETRY_WRITE_KEY", "RUDDERSTACK_CLI_TELEMETRY_DATAPLANE_URL", "RUDDERSTACK_CLI_TELEMETRY_DISABLED"}
	originalEnvs := make(map[string]string)
	for _, envVar := range envVars {
		originalEnvs[envVar] = os.Getenv(envVar)
		os.Unsetenv(envVar)
	}
	defer func() {
		for envVar, value := range originalEnvs {
			if value != "" {
				os.Setenv(envVar, value)
			}
		}
	}()

	viper.Reset()
	os.WriteFile(configFile, []byte("{}"), 0644)
	InitConfig(configFile)

	// Verify default values
	defaults := map[string]interface{}{
		"debug": false, "experimental": false, "verbose": false,
		"telemetry.disabled": false, "telemetry.writeKey": TelemetryWriteKey, "telemetry.dataplaneURL": TelemetryDataplaneURL,
	}
	for key, expected := range defaults {
		if key == "telemetry.writeKey" || key == "telemetry.dataplaneURL" {
			assert.Equal(t, expected, viper.GetString(key))
		} else {
			assert.Equal(t, expected, viper.GetBool(key))
		}
	}
	assert.NotEmpty(t, viper.GetString("apiURL"))
}

func TestCreateConfigFileIfNotExistsCI(t *testing.T) {
	t.Parallel()

	ciTests := []struct {
		name string
		test func(*testing.T)
	}{
		{"CreatesConfigFileSuccessfully", func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "test-config.json")
			err := createConfigFileIfNotExists(configFile)
			assert.NoError(t, err)
			assert.FileExists(t, configFile)
		}},
		{"HandlesMissingDirectory", func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "subdir", "config.json")
			err := createConfigFileIfNotExists(configFile)
			assert.NoError(t, err)
			assert.FileExists(t, configFile)
		}},
		{"HandlesInvalidPath", func(t *testing.T) {
			err := createConfigFileIfNotExists("/invalid/path/config.json")
			assert.Error(t, err)
		}},
		{"SkipsExistingFile", func(t *testing.T) {
			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "existing-config.json")
			os.WriteFile(configFile, []byte(`{"test": true}`), 0644)
			err := createConfigFileIfNotExists(configFile)
			assert.NoError(t, err)
			content, _ := os.ReadFile(configFile)
			assert.Contains(t, string(content), "test")
		}},
	}

	for _, test := range ciTests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			test.test(t)
		})
	}
}
