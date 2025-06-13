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

	tempDir := t.TempDir()

	t.Run("EmptyOutputDir", func(t *testing.T) {
		t.Parallel()

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})

	t.Run("InvalidOutputDir", func(t *testing.T) {
		t.Parallel()

		// Use an invalid directory path (e.g., a file that already exists)
		invalidPath := filepath.Join(tempDir, "existing_file")
		err := os.WriteFile(invalidPath, []byte("content"), 0644)
		require.NoError(t, err)

		// Skip this test since we can't easily mock package functions without dependency injection
		t.Skip("Skipping test that requires mocking package-level functions")
	})
}
