package importfromsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
