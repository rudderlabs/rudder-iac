package schema

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdSchema(t *testing.T) {
	t.Parallel()

	t.Run("CommandCreation", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdSchema()

		assert.NotNil(t, cmd)
		assert.Equal(t, "schema", cmd.Name())
		assert.Equal(t, "Manage event schemas and data catalog resources", cmd.Short)
		assert.Contains(t, cmd.Long, "Manage the lifecycle of event schemas and data catalog resources")
		assert.Contains(t, cmd.Example, "rudder-cli schema fetch")
		assert.Contains(t, cmd.Example, "rudder-cli schema unflatten")
		assert.Contains(t, cmd.Example, "rudder-cli schema convert")
	})

	t.Run("HasExpectedSubcommands", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdSchema()

		expectedSubcommands := []string{"fetch", "unflatten", "convert"}
		actualSubcommands := make([]string, 0)

		for _, subCmd := range cmd.Commands() {
			actualSubcommands = append(actualSubcommands, subCmd.Name())
		}

		for _, expected := range expectedSubcommands {
			assert.Contains(t, actualSubcommands, expected, "Expected subcommand %s not found", expected)
		}

		assert.Len(t, cmd.Commands(), 3, "Expected exactly 3 subcommands")
	})

	t.Run("CommandStructure", func(t *testing.T) {
		t.Parallel()

		cmd := NewCmdSchema()

		assert.Equal(t, "schema <command>", cmd.Use)
		assert.NotNil(t, cmd.PersistentPreRunE)

		// Verify each subcommand is properly configured
		commands := cmd.Commands()
		assert.Len(t, commands, 3, "Expected exactly 3 subcommands")

		// Check that all expected commands exist (order may vary)
		commandNames := make([]string, len(commands))
		for i, cmd := range commands {
			commandNames[i] = cmd.Name()
		}

		assert.Contains(t, commandNames, "fetch")
		assert.Contains(t, commandNames, "unflatten")
		assert.Contains(t, commandNames, "convert")
	})
}

func TestSchemaPersistentPreRunE(t *testing.T) {
	t.Run("ExperimentalModeEnabled", func(t *testing.T) {
		// Save original viper state
		originalExperimental := viper.GetBool("experimental")
		defer viper.Set("experimental", originalExperimental)

		// Enable experimental mode
		viper.Set("experimental", true)

		cmd := NewCmdSchema()
		require.NotNil(t, cmd.PersistentPreRunE)

		err := cmd.PersistentPreRunE(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("ExperimentalModeDisabled", func(t *testing.T) {
		// Save original viper state
		originalExperimental := viper.GetBool("experimental")
		defer viper.Set("experimental", originalExperimental)

		// Disable experimental mode
		viper.Set("experimental", false)

		cmd := NewCmdSchema()
		require.NotNil(t, cmd.PersistentPreRunE)

		err := cmd.PersistentPreRunE(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema commands require experimental mode")
		assert.Contains(t, err.Error(), "RUDDERSTACK_CLI_EXPERIMENTAL=true")
		assert.Contains(t, err.Error(), "experimental\": true")
	})

	t.Run("ExperimentalModeDefault", func(t *testing.T) {
		// Reset viper to test default behavior
		viper.Reset()
		defer viper.Reset()

		cmd := NewCmdSchema()
		require.NotNil(t, cmd.PersistentPreRunE)

		// By default, experimental should be false
		err := cmd.PersistentPreRunE(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "schema commands require experimental mode")
	})
}

func TestSchemaCommandWithArguments(t *testing.T) {
	t.Run("PersistentPreRunWithArguments", func(t *testing.T) {
		// Test that PersistentPreRunE works with various arguments
		originalExperimental := viper.GetBool("experimental")
		defer viper.Set("experimental", originalExperimental)

		viper.Set("experimental", true)

		cmd := NewCmdSchema()
		require.NotNil(t, cmd.PersistentPreRunE)

		// Test with various argument combinations
		testArgs := [][]string{
			{},
			{"fetch"},
			{"fetch", "output.json"},
			{"unflatten", "input.json", "output.json"},
			{"convert", "input.json", "output/"},
		}

		for _, args := range testArgs {
			err := cmd.PersistentPreRunE(cmd, args)
			assert.NoError(t, err, "PersistentPreRunE should not fail with args: %v", args)
		}
	})
}

func TestSchemaCommandIntegration(t *testing.T) {
	t.Run("SubcommandExecution", func(t *testing.T) {
		// Test that subcommands can be retrieved and have the expected properties
		cmd := NewCmdSchema()

		// Find and test fetch command
		var fetchCmd *cobra.Command
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "fetch" {
				fetchCmd = subCmd
				break
			}
		}
		require.NotNil(t, fetchCmd, "fetch command should exist")
		assert.Equal(t, "Fetch event schemas from the API", fetchCmd.Short)

		// Find and test unflatten command
		var unflattenCmd *cobra.Command
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "unflatten" {
				unflattenCmd = subCmd
				break
			}
		}
		require.NotNil(t, unflattenCmd, "unflatten command should exist")
		assert.Equal(t, "Unflatten schema JSON files with optional JSONPath extraction", unflattenCmd.Short)

		// Find and test convert command
		var convertCmd *cobra.Command
		for _, subCmd := range cmd.Commands() {
			if subCmd.Name() == "convert" {
				convertCmd = subCmd
				break
			}
		}
		require.NotNil(t, convertCmd, "convert command should exist")
		assert.Equal(t, "Convert unflattened schemas to YAML files", convertCmd.Short)
	})
}
