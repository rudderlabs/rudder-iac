package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdImportFromSource(t *testing.T) {
	t.Parallel()

	t.Run("CommandCreation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		assert.NotNil(t, cmd)
		assert.Equal(t, "importFromSource", cmd.Name())
		assert.Equal(t, "Import schemas from source and convert them to Data Catalog YAML files", cmd.Short)
		assert.Contains(t, cmd.Long, "Import schemas from source using an optimized in-memory workflow")
		assert.Contains(t, cmd.Example, "rudder-cli importFromSource output/")
	})

	t.Run("HasExpectedFlags", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Check that all expected flags are present
		assert.NotNil(t, cmd.Flags().Lookup("write-key"))
		assert.NotNil(t, cmd.Flags().Lookup("config"))
		assert.NotNil(t, cmd.Flags().Lookup("dry-run"))
		assert.NotNil(t, cmd.Flags().Lookup("verbose"))

		// Check flag types and defaults
		writeKeyFlag := cmd.Flags().Lookup("write-key")
		assert.Equal(t, "string", writeKeyFlag.Value.Type())
		assert.Equal(t, "", writeKeyFlag.DefValue)

		configFlag := cmd.Flags().Lookup("config")
		assert.Equal(t, "string", configFlag.Value.Type())
		assert.Equal(t, "", configFlag.DefValue)

		dryRunFlag := cmd.Flags().Lookup("dry-run")
		assert.Equal(t, "bool", dryRunFlag.Value.Type())
		assert.Equal(t, "false", dryRunFlag.DefValue)

		verboseFlag := cmd.Flags().Lookup("verbose")
		assert.Equal(t, "bool", verboseFlag.Value.Type())
		assert.Equal(t, "false", verboseFlag.DefValue)
	})

	t.Run("CommandStructure", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Check command structure
		assert.NotNil(t, cmd.Args)
		assert.NotNil(t, cmd.RunE)
		assert.NotNil(t, cmd.PersistentPreRunE)
	})
}

func TestImportFromSourcePersistentPreRunE(t *testing.T) {
	t.Parallel()

	t.Run("ExperimentalModeEnabled", func(t *testing.T) {
		t.Parallel()

		// Store original value
		originalExperimental := viper.GetBool("experimental")
		defer viper.Set("experimental", originalExperimental)

		viper.Set("experimental", true)

		cmd := NewCmdImportFromSource()
		require.NotNil(t, cmd.PersistentPreRunE)

		err := cmd.PersistentPreRunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("ExperimentalModeDisabled", func(t *testing.T) {
		t.Parallel()

		// Store original value
		originalExperimental := viper.GetBool("experimental")
		defer viper.Set("experimental", originalExperimental)

		viper.Set("experimental", false)

		cmd := NewCmdImportFromSource()
		require.NotNil(t, cmd.PersistentPreRunE)

		err := cmd.PersistentPreRunE(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "importFromSource command requires experimental mode")
	})
}

func TestImportFromSourceCommandWithArguments(t *testing.T) {
	t.Run("PersistentPreRunWithArguments", func(t *testing.T) {
		// Test that PersistentPreRunE works with various arguments
		originalExperimental := viper.GetBool("experimental")
		defer viper.Set("experimental", originalExperimental)

		viper.Set("experimental", true)

		cmd := NewCmdImportFromSource()
		require.NotNil(t, cmd.PersistentPreRunE)

		// Test with various argument combinations
		testArgs := [][]string{
			{"output/"},
			{"my-output-dir/"},
			{"/tmp/output"},
		}

		for _, args := range testArgs {
			err := cmd.PersistentPreRunE(cmd, args)
			assert.NoError(t, err, "PersistentPreRunE should not fail with args: %v", args)
		}
	})
}

func TestImportFromSourceIntegration(t *testing.T) {
	t.Run("CommandExecution", func(t *testing.T) {
		// Test that the command can be retrieved and has the expected properties
		cmd := NewCmdImportFromSource()

		assert.Equal(t, "importFromSource", cmd.Name())
		assert.NotNil(t, cmd.RunE)

		// Test flag parsing
		cmd.SetArgs([]string{"output/", "--write-key=test-key", "--config=config.yaml", "--verbose", "--dry-run"})
		err := cmd.ParseFlags([]string{"output/", "--write-key=test-key", "--config=config.yaml", "--verbose", "--dry-run"})
		assert.NoError(t, err)

		// Check flag values
		writeKey, err := cmd.Flags().GetString("write-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-key", writeKey)

		config, err := cmd.Flags().GetString("config")
		assert.NoError(t, err)
		assert.Equal(t, "config.yaml", config)

		verbose, err := cmd.Flags().GetBool("verbose")
		assert.NoError(t, err)
		assert.True(t, verbose)

		dryRun, err := cmd.Flags().GetBool("dry-run")
		assert.NoError(t, err)
		assert.True(t, dryRun)
	})
}

func TestRunImportFromSourceConfigValidation(t *testing.T) {
	t.Parallel()

	t.Run("InvalidConfigFile", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		outputDir := filepath.Join(tempDir, "output")

		// Test with non-existent config file
		err := runImportFromSource(outputDir, "", "/nonexistent/config.yaml", true, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load event type configuration")
	})

	t.Run("ValidConfigFile", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		outputDir := filepath.Join(tempDir, "output")
		configFile := filepath.Join(tempDir, "config.yaml")

		// Create a valid config file
		configContent := `
event_mappings:
  identify: "$.traits"
  page: "$.context.traits"
  screen: "$.properties"
  group: "$.properties" 
  alias: "$.traits"
`
		err := os.WriteFile(configFile, []byte(configContent), 0644)
		require.NoError(t, err)

		// Test config loading (dry-run to avoid actual API calls)
		err = runImportFromSource(outputDir, "", configFile, true, false)
		// This will fail at the fetch step since we're not mocking the API,
		// but it should pass config validation
		if err != nil {
			// Expect failure at fetch step, not config validation
			assert.NotContains(t, err.Error(), "failed to load event type configuration")
		}
	})
}

func TestRunImportFromSourceDryRun(t *testing.T) {
	t.Parallel()

	t.Run("DryRunMode", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		outputDir := filepath.Join(tempDir, "output")

		// Test dry-run mode (should not create actual files)
		err := runImportFromSource(outputDir, "", "", true, false)
		// This will likely fail at fetch step since we're not mocking API,
		// but dry-run should be handled properly
		if err != nil {
			// Check that it's not a config error
			assert.NotContains(t, err.Error(), "failed to load event type configuration")
		}

		// Output directory should not be created in dry-run
		assert.NoFileExists(t, outputDir)
	})
}

func TestImportFromSourceCommandHelp(t *testing.T) {
	t.Parallel()

	t.Run("HelpText", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Test that help contains expected information
		assert.Contains(t, cmd.Long, "Configuration File Format")
		assert.Contains(t, cmd.Long, "event_mappings:")
		assert.Contains(t, cmd.Long, "$.traits")
		assert.Contains(t, cmd.Long, "$.context.traits")
		assert.Contains(t, cmd.Long, "$.properties")
		assert.Contains(t, cmd.Long, "Default Behavior:")
		assert.Contains(t, cmd.Long, "track events always use \"$.properties\"")

		// Test examples
		assert.Contains(t, cmd.Example, "rudder-cli importFromSource output/")
		assert.Contains(t, cmd.Example, "--write-key=YOUR_WRITE_KEY")
		assert.Contains(t, cmd.Example, "--config=event-mappings.yaml")
		assert.Contains(t, cmd.Example, "--verbose")
	})
}

func TestImportFromSourceFlagValidation(t *testing.T) {
	t.Parallel()

	t.Run("FlagDefaults", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Test default flag values
		writeKey, err := cmd.Flags().GetString("write-key")
		assert.NoError(t, err)
		assert.Equal(t, "", writeKey)

		config, err := cmd.Flags().GetString("config")
		assert.NoError(t, err)
		assert.Equal(t, "", config)

		dryRun, err := cmd.Flags().GetBool("dry-run")
		assert.NoError(t, err)
		assert.False(t, dryRun)

		verbose, err := cmd.Flags().GetBool("verbose")
		assert.NoError(t, err)
		assert.False(t, verbose)
	})

	t.Run("FlagSetting", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Test setting flag values
		err := cmd.Flags().Set("write-key", "test-write-key")
		assert.NoError(t, err)

		err = cmd.Flags().Set("config", "test-config.yaml")
		assert.NoError(t, err)

		err = cmd.Flags().Set("dry-run", "true")
		assert.NoError(t, err)

		err = cmd.Flags().Set("verbose", "true")
		assert.NoError(t, err)

		// Verify values
		writeKey, err := cmd.Flags().GetString("write-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-write-key", writeKey)

		config, err := cmd.Flags().GetString("config")
		assert.NoError(t, err)
		assert.Equal(t, "test-config.yaml", config)

		dryRun, err := cmd.Flags().GetBool("dry-run")
		assert.NoError(t, err)
		assert.True(t, dryRun)

		verbose, err := cmd.Flags().GetBool("verbose")
		assert.NoError(t, err)
		assert.True(t, verbose)
	})
}

func TestImportFromSourceUsage(t *testing.T) {
	t.Parallel()

	t.Run("CorrectUsage", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()
		assert.Equal(t, "importFromSource [output-dir]", cmd.Use)
	})

	t.Run("ArgumentValidation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Test that command expects exactly 1 argument by checking Args function
		assert.NotNil(t, cmd.Args, "Args function should be set")

		// Test with correct number of args - should not error
		err := cmd.Args(cmd, []string{"output/"})
		assert.NoError(t, err)

		// Test with wrong number of args - should error
		err = cmd.Args(cmd, []string{})
		assert.Error(t, err)

		err = cmd.Args(cmd, []string{"output1/", "output2/"})
		assert.Error(t, err)
	})
}

func TestImportFromSourceOptimizedWorkflow(t *testing.T) {
	t.Parallel()

	t.Run("InMemoryProcessing", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()
		assert.Contains(t, cmd.Long, "in-memory workflow")
		assert.Contains(t, cmd.Long, "optimized in-memory workflow")
	})

	t.Run("NoTemporaryFiles", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Should mention no temporary file I/O but not creating temporary files
		assert.Contains(t, cmd.Long, "No temporary file I/O")

		// Should mention in-memory processing
		assert.Contains(t, cmd.Long, "memory")
	})

	t.Run("PerformanceBenefits", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdImportFromSource()

		// Should mention performance benefits
		assert.Contains(t, cmd.Long, "Performance Benefits")
		assert.Contains(t, cmd.Long, "efficiency")
	})
}
