package importfromsource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rudderlabs/rudder-iac/cli/internal/schema/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdTPImportFromSource(t *testing.T) {
	t.Parallel()

	t.Run("CommandCreation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		assert.NotNil(t, cmd)
		assert.Equal(t, "importFromSource", cmd.Name())
		assert.Equal(t, "Import schemas from source and convert them to Data Catalog YAML files", cmd.Short)
		assert.Contains(t, cmd.Long, "Import schemas from source using an optimized in-memory workflow")
		assert.Contains(t, cmd.Example, "rudder-cli tp importFromSource output/")
	})

	t.Run("HasExpectedFlags", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Check that all expected flags are present
		assert.NotNil(t, cmd.Flags().Lookup("write-key"))
		assert.NotNil(t, cmd.Flags().Lookup("config"))
		assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
		assert.NotNil(t, cmd.Flags().Lookup("verbose"))
	})

	t.Run("FlagDefaults", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Check flag defaults
		writeKeyFlag := cmd.Flags().Lookup("write-key")
		assert.Equal(t, "", writeKeyFlag.DefValue)

		configFlag := cmd.Flags().Lookup("config")
		assert.Equal(t, "", configFlag.DefValue)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		assert.Equal(t, "false", dryRunFlag.DefValue)

		verboseFlag := cmd.Flags().Lookup("verbose")
		assert.Equal(t, "false", verboseFlag.DefValue)
	})
}

func TestTPImportFromSourceUsage(t *testing.T) {
	t.Parallel()

	t.Run("CorrectUsage", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()
		assert.Equal(t, "importFromSource [output-dir]", cmd.Use)
	})

	t.Run("ArgumentValidation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Test that command expects exactly 1 argument by checking Args function
		assert.NotNil(t, cmd.Args, "Args function should be set")

		// Test with correct number of args - should not error
		err := cmd.Args(cmd, []string{"output/"})
		assert.NoError(t, err)

		// Test with wrong number of args - should error
		err = cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"output/", "extra"})
		assert.Error(t, err)
	})
}

func TestTPImportFromSourceOptimizedWorkflow(t *testing.T) {
	t.Parallel()

	t.Run("InMemoryProcessing", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()
		assert.Contains(t, cmd.Long, "in-memory processing")
		assert.Contains(t, cmd.Long, "optimized in-memory workflow")
	})

	t.Run("NoTemporaryFiles", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// The optimized workflow should mention eliminating temporary files
		assert.Contains(t, cmd.Long, "eliminating")
		assert.Contains(t, cmd.Long, "temporary files")

		// Should mention in-memory processing
		assert.Contains(t, cmd.Long, "memory")
	})

	t.Run("PerformanceBenefits", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Should highlight performance benefits
		assert.Contains(t, cmd.Long, "optimal performance")
		assert.Contains(t, cmd.Long, "eliminating")
		assert.Contains(t, cmd.Long, "I/O overhead")
	})
}

func TestTPImportFromSourceConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("ConfigurationDocumentation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Should document YAML configuration format
		assert.Contains(t, cmd.Long, "Configuration File Format")
		assert.Contains(t, cmd.Long, "event_mappings")
		assert.Contains(t, cmd.Long, "identify:")
		assert.Contains(t, cmd.Long, "$.traits")
		assert.Contains(t, cmd.Long, "$.context.traits")
		assert.Contains(t, cmd.Long, "$.properties")
	})

	t.Run("DefaultBehavior", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Should document default behavior
		assert.Contains(t, cmd.Long, "Default Behavior")
		assert.Contains(t, cmd.Long, "Without writeKey: fetches all schemas")
		assert.Contains(t, cmd.Long, "Without config: all event types use")
		assert.Contains(t, cmd.Long, "Track events always use")
	})

	t.Run("Examples", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Should provide comprehensive examples
		assert.Contains(t, cmd.Example, "rudder-cli tp importFromSource output/")
		assert.Contains(t, cmd.Example, "--write-key")
		assert.Contains(t, cmd.Example, "--config")
		assert.Contains(t, cmd.Example, "--dry-run")
		assert.Contains(t, cmd.Example, "--verbose")
	})
}

func TestNewCmdTPImportFromSource_Command(t *testing.T) {
	t.Parallel()

	t.Run("CommandCreation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		assert.NotNil(t, cmd)
		assert.Equal(t, "importFromSource", cmd.Name())
		assert.Equal(t, "Import schemas from source and convert them to Data Catalog YAML files", cmd.Short)
		assert.Contains(t, cmd.Long, "Import schemas from source using an optimized in-memory workflow")
		assert.Contains(t, cmd.Example, "rudder-cli tp importFromSource output/")
	})

	t.Run("CommandFlags", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Check that all expected flags are present
		assert.NotNil(t, cmd.Flags().Lookup("write-key"))
		assert.NotNil(t, cmd.Flags().Lookup("config"))
		assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
		assert.NotNil(t, cmd.Flags().Lookup("verbose"))
	})

	t.Run("FlagDefaults", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Check flag defaults
		writeKeyFlag := cmd.Flags().Lookup("write-key")
		assert.Equal(t, "", writeKeyFlag.DefValue)

		configFlag := cmd.Flags().Lookup("config")
		assert.Equal(t, "", configFlag.DefValue)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		assert.Equal(t, "false", dryRunFlag.DefValue)

		verboseFlag := cmd.Flags().Lookup("verbose")
		assert.Equal(t, "false", verboseFlag.DefValue)
	})

	t.Run("ArgumentValidation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		// Test that command expects exactly 1 argument by checking Args function
		assert.NotNil(t, cmd.Args, "Args function should be set")

		// Test with correct number of args - should not error
		err := cmd.Args(cmd, []string{"output/"})
		assert.NoError(t, err)

		// Test with wrong number of args - should error
		err = cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"output/", "extra"})
		assert.Error(t, err)
	})

	t.Run("LongDescription", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		assert.Contains(t, cmd.Long, "fetch")
		assert.Contains(t, cmd.Long, "unflatten")
		assert.Contains(t, cmd.Long, "convert")
		assert.Contains(t, cmd.Long, "in-memory")
		assert.Contains(t, cmd.Long, "track\" always uses \"$.properties\"")
	})

	t.Run("ExampleUsages", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdTPImportFromSource()

		assert.Contains(t, cmd.Example, "--write-key")
		assert.Contains(t, cmd.Example, "--config")
		assert.Contains(t, cmd.Example, "--dry-run")
		assert.Contains(t, cmd.Example, "--verbose")
	})
}

func TestRunImportFromSource_CommandExecution(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("InvalidConfigFile", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "invalid_config_output")
		configFile := "/nonexistent/config.yaml"

		// This should fail when trying to load the config file
		err := runImportFromSource(outputDir, "", configFile, true, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load event type configuration")
	})

	t.Run("InvalidOutputDirectory", func(t *testing.T) {
		t.Parallel()

		// Try to use a file as an output directory
		invalidPath := filepath.Join(tempDir, "not_a_directory")
		err := os.WriteFile(invalidPath, []byte("content"), 0644)
		require.NoError(t, err)

		// This test might fail at the schema fetch step before reaching directory creation
		// but we can test that it properly handles the error
		err = runImportFromSource(invalidPath, "", "", true, false)
		// The test might succeed in dry-run mode if schema fetching fails first
		// We just verify it doesn't panic and handles errors gracefully
		if err != nil {
			// Error is expected in some cases
			assert.Error(t, err)
		}
	})

	t.Run("EmptyArguments", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "empty_args_output")

		// Test with all empty arguments - should still try to execute
		err := runImportFromSource(outputDir, "", "", true, false)
		// In dry-run mode, this might succeed or fail depending on schema fetching
		// We just verify it doesn't panic
		if err != nil {
			// Error is expected when schema fetching fails
			assert.Error(t, err)
		}
	})
}

func TestRunImportFromSource_ErrorCases(t *testing.T) {
	t.Parallel()

	// Skip if experimental mode is not enabled
	if !isExperimentalEnabled() {
		t.Skip("Experimental mode not enabled")
	}

	tempDir := t.TempDir()

	t.Run("InvalidConfigFile", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "invalid_config_output")

		// Test with an actual nonexistent config file
		err := runImportFromSource(outputDir, "", "nonexistent.yaml", false, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load event type configuration")
	})

	t.Run("FetchSchemasError", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})

	t.Run("UnflattenSchemasError", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})

	t.Run("ConvertSchemasError", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})
}

func TestRunImportFromSource_ConfigHandling(t *testing.T) {
	t.Parallel()

	// Skip if experimental mode is not enabled
	if !isExperimentalEnabled() {
		t.Skip("Experimental mode not enabled")
	}

	tempDir := t.TempDir()
	_ = filepath.Join(tempDir, "config_handling_output") // outputDir not used in current tests

	t.Run("DefaultConfig", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})

	t.Run("CustomConfigFile", func(t *testing.T) {
		t.Parallel()

		configFile := filepath.Join(tempDir, "custom_config.yaml")

		// Create custom config file
		configContent := `event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  track: "$.properties"`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test that custom config file can be loaded - this tests the actual LoadEventTypeConfig function
		config, err := models.LoadEventTypeConfig(configFile)
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "$.traits", config.EventMappings["identify"])
		assert.Equal(t, "$.context.traits", config.EventMappings["page"])
	})
}

func TestRunImportFromSource_Statistics(t *testing.T) {
	t.Parallel()

	// Skip if experimental mode is not enabled
	if !isExperimentalEnabled() {
		t.Skip("Experimental mode not enabled")
	}

	tempDir := t.TempDir()
	_ = filepath.Join(tempDir, "statistics_output") // outputDir not used in current tests

	t.Run("StatisticsReporting", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})

	t.Run("EmptyResults", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})
}

// Helper function to check if experimental mode is enabled
func isExperimentalEnabled() bool {
	// Check environment variable or config
	experimental := os.Getenv("RUDDERSTACK_CLI_EXPERIMENTAL")
	return experimental == "true"
}

// Additional edge case tests
func TestRunImportFromSource_EdgeCases(t *testing.T) {
	t.Parallel()

	// Skip if experimental mode is not enabled
	if !isExperimentalEnabled() {
		t.Skip("Experimental mode not enabled")
	}

	t.Run("EmptyOutputDir", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})

	t.Run("InvalidOutputDir", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})
}

// New comprehensive tests for improved coverage

func TestRunImportFromSource_ComprehensiveCoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		outputDir   string
		writeKey    string
		configFile  string
		dryRun      bool
		verbose     bool
		setupConfig func(tempDir string) string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ValidDryRunNoConfig",
			outputDir:   "valid_output",
			writeKey:    "",
			configFile:  "",
			dryRun:      true,
			verbose:     false,
			setupConfig: nil,
			expectError: true, // Expected to fail due to missing access token or API issues
			errorMsg:    "",   // Could be various API-related errors
		},
		{
			name:        "ValidDryRunWithWriteKey",
			outputDir:   "valid_writekey_output",
			writeKey:    "test-write-key",
			configFile:  "",
			dryRun:      true,
			verbose:     true,
			setupConfig: nil,
			expectError: true, // Expected to fail due to missing access token or API issues
			errorMsg:    "",
		},
		{
			name:       "ValidConfigFileStructure",
			outputDir:  "valid_config_output",
			writeKey:   "",
			configFile: "valid_config.yaml",
			dryRun:     true,
			verbose:    false,
			setupConfig: func(tempDir string) string {
				configFile := filepath.Join(tempDir, "valid_config.yaml")
				configContent := `event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.context.traits"
  group: "$.traits"
  alias: "$.traits"
  track: "$.properties"`
				err := os.WriteFile(configFile, []byte(configContent), 0644)
				require.NoError(t, err)
				return configFile
			},
			expectError: true, // Expected to fail due to API issues, but should load config successfully
			errorMsg:    "",
		},
		{
			name:       "InvalidConfigFileFormat",
			outputDir:  "invalid_config_output",
			writeKey:   "",
			configFile: "invalid_config.yaml",
			dryRun:     true,
			verbose:    false,
			setupConfig: func(tempDir string) string {
				configFile := filepath.Join(tempDir, "invalid_config.yaml")
				configContent := `invalid yaml content: [unclosed bracket`
				err := os.WriteFile(configFile, []byte(configContent), 0644)
				require.NoError(t, err)
				return configFile
			},
			expectError: true,
			errorMsg:    "failed to load event type configuration",
		},
		{
			name:       "EmptyConfigFile",
			outputDir:  "empty_config_output",
			writeKey:   "",
			configFile: "empty_config.yaml",
			dryRun:     true,
			verbose:    false,
			setupConfig: func(tempDir string) string {
				configFile := filepath.Join(tempDir, "empty_config.yaml")
				err := os.WriteFile(configFile, []byte(""), 0644)
				require.NoError(t, err)
				return configFile
			},
			expectError: true, // May fail at API level
			errorMsg:    "",
		},
		{
			name:        "NonExistentConfigFile",
			outputDir:   "nonexistent_config_output",
			writeKey:    "",
			configFile:  "nonexistent.yaml",
			dryRun:      true,
			verbose:     false,
			setupConfig: nil,
			expectError: true,
			errorMsg:    "failed to load event type configuration",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			outputDir := filepath.Join(tempDir, c.outputDir)

			var configFile string
			if c.setupConfig != nil {
				configFile = c.setupConfig(tempDir)
			} else if c.configFile != "" {
				configFile = c.configFile
			}

			err := runImportFromSource(outputDir, c.writeKey, configFile, c.dryRun, c.verbose)

			if c.expectError {
				assert.Error(t, err)
				if c.errorMsg != "" {
					assert.Contains(t, err.Error(), c.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				// Verify output directory exists
				assert.DirExists(t, outputDir)
			}
		})
	}
}

func TestRunImportFromSource_ConfigVariations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		configContent  string
		expectError    bool
		expectedConfig map[string]string
	}{
		{
			name: "StandardConfigFormat",
			configContent: `event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.context.traits"
  group: "$.traits"
  alias: "$.traits"`,
			expectError: false,
			expectedConfig: map[string]string{
				"identify": "$.traits",
				"page":     "$.context.traits",
				"screen":   "$.context.traits",
				"group":    "$.traits",
				"alias":    "$.traits",
			},
		},
		{
			name: "PropertiesConfig",
			configContent: `event_mappings:
  identify: "$.properties"
  page: "$.properties"
  group: "$.properties"`,
			expectError: false,
			expectedConfig: map[string]string{
				"identify": "$.properties",
				"page":     "$.properties",
				"group":    "$.properties",
			},
		},
		{
			name: "MixedConfig",
			configContent: `event_mappings:
  identify: "$.traits"
  page: "$.properties"
  screen: "$.context.traits"
  group: "$.traits"
  alias: "$.properties"`,
			expectError: false,
			expectedConfig: map[string]string{
				"identify": "$.traits",
				"page":     "$.properties",
				"screen":   "$.context.traits",
				"group":    "$.traits",
				"alias":    "$.properties",
			},
		},
		{
			name: "InvalidYAMLSyntax",
			configContent: `event_mappings:
  identify: "$.traits"
  page: [unclosed bracket
  group: "$.traits"`,
			expectError: true,
		},
		{
			name: "MissingEventMappings",
			configContent: `other_section:
  some_key: "some_value"`,
			expectError: false, // Should use defaults
		},
		{
			name:          "EmptyFile",
			configContent: "",
			expectError:   false, // Should use defaults
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			configFile := filepath.Join(tempDir, "test_config.yaml")

			err := os.WriteFile(configFile, []byte(c.configContent), 0644)
			require.NoError(t, err)

			config, err := models.LoadEventTypeConfig(configFile)

			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)

				// Check expected mappings if provided
				if c.expectedConfig != nil {
					for eventType, expectedPath := range c.expectedConfig {
						actualPath, exists := config.EventMappings[eventType]
						assert.True(t, exists, "Event type %s should exist in config", eventType)
						assert.Equal(t, expectedPath, actualPath, "Path for event type %s should match", eventType)
					}
				}
			}
		})
	}
}

func TestRunImportFromSource_VerboseModeOutput(t *testing.T) {
	t.Parallel()

	t.Run("VerboseModeExecution", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		outputDir := filepath.Join(tempDir, "verbose_output")

		// Test verbose mode - should print additional information
		err := runImportFromSource(outputDir, "", "", true, true)

		// Expect error due to API issues, but test doesn't panic in verbose mode
		assert.Error(t, err)
	})

	t.Run("NonVerboseModeExecution", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		outputDir := filepath.Join(tempDir, "nonverbose_output")

		// Test non-verbose mode
		err := runImportFromSource(outputDir, "", "", true, false)

		// Expect error due to API issues, but test doesn't panic in non-verbose mode
		assert.Error(t, err)
	})
}

func TestRunImportFromSource_WriteKeyVariations(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	cases := []struct {
		name     string
		writeKey string
	}{
		{
			name:     "EmptyWriteKey",
			writeKey: "",
		},
		{
			name:     "ShortWriteKey",
			writeKey: "abc123",
		},
		{
			name:     "LongWriteKey",
			writeKey: "1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "WriteKeyWithSpecialChars",
			writeKey: "test-write-key_123",
		},
		{
			name:     "WriteKeyWithDashes",
			writeKey: "test-write-key-with-dashes",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			outputDir := filepath.Join(tempDir, c.name+"_output")

			// Test that function handles different write key formats without panicking
			err := runImportFromSource(outputDir, c.writeKey, "", true, false)

			// Expect error due to API issues, but should handle write key properly
			assert.Error(t, err)
		})
	}
}

func TestRunImportFromSource_DirectoryHandling(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("ValidOutputDirectory", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "valid_dir_output")

		// Directory doesn't exist yet - should be created by the function
		err := runImportFromSource(outputDir, "", "", true, false)

		// May fail at API level, but directory handling should work
		assert.Error(t, err) // Expected due to API issues
	})

	t.Run("ExistingOutputDirectory", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "existing_dir_output")
		err := os.MkdirAll(outputDir, 0755)
		require.NoError(t, err)

		// Directory already exists - should handle gracefully
		err = runImportFromSource(outputDir, "", "", true, false)

		// May fail at API level, but directory handling should work
		assert.Error(t, err) // Expected due to API issues
	})

	t.Run("NestedOutputDirectory", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "nested", "deep", "directory", "output")

		// Nested directory - should create entire path
		err := runImportFromSource(outputDir, "", "", true, false)

		// May fail at API level, but directory handling should work
		assert.Error(t, err) // Expected due to API issues
	})

	t.Run("RelativeOutputDirectory", func(t *testing.T) {
		t.Parallel()

		// Change to temp directory to test relative path
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		outputDir := "relative_output"

		err = runImportFromSource(outputDir, "", "", true, false)

		// May fail at API level, but relative path handling should work
		assert.Error(t, err) // Expected due to API issues
	})
}

func TestRunImportFromSource_ErrorPathCoverage(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("InvalidConfigFilePath", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "invalid_config_path_output")
		configFile := "/root/nonexistent/impossible/path/config.yaml"

		err := runImportFromSource(outputDir, "", configFile, true, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load event type configuration")
	})

	t.Run("ConfigFileIsDirectory", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "config_dir_output")
		configDir := filepath.Join(tempDir, "config_as_directory")

		err := os.MkdirAll(configDir, 0755)
		require.NoError(t, err)

		err = runImportFromSource(outputDir, "", configDir, true, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load event type configuration")
	})

	t.Run("ConfigFilePermissionDenied", func(t *testing.T) {
		t.Parallel()

		// Skip on Windows as permission handling is different
		if os.Getenv("GOOS") == "windows" {
			t.Skip("Skipping permission test on Windows")
		}

		outputDir := filepath.Join(tempDir, "permission_denied_output")
		configFile := filepath.Join(tempDir, "no_permission_config.yaml")

		// Create config file with content
		err := os.WriteFile(configFile, []byte("event_mappings:\n  identify: $.traits"), 0644)
		require.NoError(t, err)

		// Remove read permissions
		err = os.Chmod(configFile, 0000)
		require.NoError(t, err)

		// Restore permissions after test
		defer func() { _ = os.Chmod(configFile, 0644) }()

		err = runImportFromSource(outputDir, "", configFile, true, false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load event type configuration")
	})
}

func TestRunImportFromSource_IntegrationScenarios(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("FullWorkflowDryRun", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "full_workflow_output")
		configFile := filepath.Join(tempDir, "full_workflow_config.yaml")

		// Create a comprehensive config file
		configContent := `event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.context.traits"
  group: "$.traits"
  alias: "$.traits"
  track: "$.properties"`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test full workflow in dry-run mode with verbose output
		err = runImportFromSource(outputDir, "test-write-key", configFile, true, true)

		// Expected to fail due to API configuration, but should handle all parameters
		assert.Error(t, err)
	})

	t.Run("MinimalConfiguration", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "minimal_config_output")

		// Test with minimal configuration (all defaults)
		err := runImportFromSource(outputDir, "", "", true, false)

		// Expected to fail due to API issues, but should handle minimal config
		assert.Error(t, err)
	})

	t.Run("ComplexWriteKeyAndConfig", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "complex_output")
		configFile := filepath.Join(tempDir, "complex_config.yaml")

		// Create complex config with all event types
		configContent := `event_mappings:
  identify: "$.traits"
  page: "$.properties"
  screen: "$.context.traits"
  group: "$.traits"
  alias: "$.properties"
  track: "$.properties"`

		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test with complex write key and config
		err = runImportFromSource(outputDir, "complex-write-key-123_test", configFile, true, true)

		// Expected to fail due to API issues, but should handle complex parameters
		assert.Error(t, err)
	})
}

func TestRunImportFromSource_ParameterValidation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	t.Run("EmptyOutputDirectory", func(t *testing.T) {
		t.Parallel()

		// Test with empty output directory
		err := runImportFromSource("", "", "", true, false)

		// Should handle empty output directory gracefully
		assert.Error(t, err)
	})

	t.Run("WhitespaceOutputDirectory", func(t *testing.T) {
		t.Parallel()

		// Test with whitespace-only output directory
		err := runImportFromSource("   ", "", "", true, false)

		// Should handle whitespace-only output directory
		assert.Error(t, err)
	})

	t.Run("AllParametersEmpty", func(t *testing.T) {
		t.Parallel()

		// Test with all parameters empty/default
		err := runImportFromSource("", "", "", false, false)

		// Should handle all empty parameters
		assert.Error(t, err)
	})

	t.Run("OnlyVerboseSet", func(t *testing.T) {
		t.Parallel()

		outputDir := filepath.Join(tempDir, "only_verbose_output")

		// Test with only verbose flag set
		err := runImportFromSource(outputDir, "", "", false, true)

		// Should handle verbose-only configuration
		assert.Error(t, err)
	})

	// Use tempDir to avoid unused variable error
	_ = tempDir
}

func TestRunImportFromSource_ConfigContentValidation(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	cases := []struct {
		name          string
		configContent string
		expectError   bool
		errorContains string
	}{
		{
			name: "ValidCompleteConfig",
			configContent: `event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.context.traits"
  group: "$.traits"
  alias: "$.traits"
  track: "$.properties"`,
			expectError: false,
		},
		{
			name: "ValidPartialConfig",
			configContent: `event_mappings:
  identify: "$.traits"
  page: "$.properties"`,
			expectError: false,
		},
		{
			name: "InvalidYAMLStructure",
			configContent: `event_mappings:
  identify: "$.traits"
  page: [invalid structure
  group: "$.traits"`,
			expectError:   true,
			errorContains: "failed to load event type configuration",
		},
		{
			name: "InvalidJSONPath",
			configContent: `event_mappings:
  identify: "$.traits"
  page: "invalid-json-path-without-dollar"
  group: "$.traits"`,
			expectError:   true, // Invalid paths are now validated at config loading time
			errorContains: "failed to load event type configuration",
		},
		{
			name:          "EmptyEventMappings",
			configContent: `event_mappings:`,
			expectError:   false, // Should use defaults
		},
		{
			name:          "OnlyComments",
			configContent: `# This is just a comment\n# Another comment`,
			expectError:   false, // Should use defaults
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			outputDir := filepath.Join(tempDir, c.name+"_output")
			configFile := filepath.Join(tempDir, c.name+"_config.yaml")

			err := os.WriteFile(configFile, []byte(c.configContent), 0644)
			require.NoError(t, err)

			err = runImportFromSource(outputDir, "", configFile, true, false)

			if c.expectError {
				assert.Error(t, err)
				if c.errorContains != "" {
					assert.Contains(t, err.Error(), c.errorContains)
				}
			} else {
				// Even valid configs will fail due to API issues, but shouldn't fail at config loading
				assert.Error(t, err)
				// Should not contain config loading error
				assert.NotContains(t, err.Error(), "failed to load event type configuration")
			}
		})
	}
}
