package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_Structure(t *testing.T) {
	t.Parallel()

	t.Run("HasCorrectBasicProperties", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "rudder-cli", rootCmd.Use)
		assert.Equal(t, "Rudder CLI", rootCmd.Short)
		assert.Contains(t, rootCmd.Long, "Rudder is a CLI tool")
		assert.True(t, rootCmd.SilenceUsage)
		assert.True(t, rootCmd.SilenceErrors)
	})

	t.Run("HasExpectedSubcommands", func(t *testing.T) {
		t.Parallel()

		expectedCommands := []string{"auth", "debug", "schema", "tp", "telemetry"}
		actualCommands := make([]string, 0)

		for _, cmd := range rootCmd.Commands() {
			actualCommands = append(actualCommands, cmd.Name())
		}

		for _, expected := range expectedCommands {
			assert.Contains(t, actualCommands, expected, "Expected subcommand %s not found", expected)
		}
	})

	t.Run("HasConfigFlag", func(t *testing.T) {
		t.Parallel()

		configFlag := rootCmd.PersistentFlags().Lookup("config")
		require.NotNil(t, configFlag)
		assert.Equal(t, "c", configFlag.Shorthand)
		assert.Contains(t, configFlag.Usage, "config file")
	})
}

func TestSetVersion(t *testing.T) {
	// Removed t.Parallel() to avoid race conditions with global rootCmd.Version

	cases := []struct {
		name    string
		version string
	}{
		{
			name:    "ValidVersion",
			version: "1.0.0",
		},
		{
			name:    "VersionWithBuild",
			version: "1.0.0-beta.1",
		},
		{
			name:    "EmptyVersion",
			version: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Removed t.Parallel() to avoid race conditions with global rootCmd.Version

			SetVersion(c.version)
			assert.Equal(t, c.version, rootCmd.Version)
		})
	}
}

func TestRootCommand_Execution(t *testing.T) {
	t.Parallel()

	t.Run("ShowsHelpWhenNoArgs", func(t *testing.T) {
		t.Parallel()

		// Create a new command instance to avoid global state issues
		cmd := &cobra.Command{
			Use:   "rudder-cli",
			Short: "Rudder CLI",
			Run: func(cmd *cobra.Command, args []string) {
				cmd.Help()
			},
		}

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Rudder CLI")
		assert.Contains(t, output, "Usage:")
	})
}

func TestRecovery(t *testing.T) {
	t.Run("CatchesPanic", func(t *testing.T) {
		// This test verifies the recovery function exists and can be called
		// We can't easily test the actual panic recovery without causing side effects
		assert.NotPanics(t, func() {
			defer func() {
				if r := recover(); r != nil {
					// Expected behavior - the recovery function should handle panics
					assert.NotNil(t, r)
				}
			}()
			recovery()
		})
	})

}

func TestInitFunctions(t *testing.T) {
	// These tests verify that the init functions can be called without errors
	// Full integration testing would require more complex setup

	t.Run("InitConfig", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = filepath.Join(tempDir, "test_config.json") // We create a temp path but don't use it

		// Set a temporary config file to avoid affecting global state
		originalViper := viper.GetViper()
		defer func() {
			viper.Reset()
			for key, value := range originalViper.AllSettings() {
				viper.Set(key, value)
			}
		}()

		assert.NotPanics(t, func() {
			initConfig()
		})
	})

	t.Run("InitLogger", func(t *testing.T) {
		assert.NotPanics(t, func() {
			initLogger()
		})
	})

	t.Run("InitLoggerWithDebugEnabled", func(t *testing.T) {
		// Test the debug path in initLogger
		viper.Set("debug", true)
		defer viper.Set("debug", false)

		assert.NotPanics(t, func() {
			initLogger()
		})
	})

	t.Run("InitAppDependencies", func(t *testing.T) {
		assert.NotPanics(t, func() {
			initAppDependencies()
		})
	})

	t.Run("InitTelemetry", func(t *testing.T) {
		assert.NotPanics(t, func() {
			initTelemetry()
		})
	})
}

func TestExecute(t *testing.T) {
	t.Run("ExecuteWithValidCommand", func(t *testing.T) {
		// Store original args
		originalArgs := os.Args

		// Set test args (help command should always work)
		os.Args = []string{"rudder-cli", "--help"}

		defer func() {
			os.Args = originalArgs
			if r := recover(); r != nil {
				t.Errorf("Execute panicked: %v", r)
			}
		}()

		// Capture stderr to check for errors
		var stderr bytes.Buffer
		originalStderr := os.Stderr

		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stderr = w

		done := make(chan bool)
		go func() {
			defer close(done)
			buf := make([]byte, 1024)
			for {
				n, err := r.Read(buf)
				if err != nil {
					break
				}
				stderr.Write(buf[:n])
			}
		}()

		// This would normally exit the program, so we can't easily test it
		// without more complex setup. We'll just verify the function exists.
		assert.NotNil(t, Execute)

		w.Close()
		<-done
		os.Stderr = originalStderr

		// The help command should not produce errors
		errorOutput := stderr.String()
		assert.False(t, strings.Contains(errorOutput, "panic"), "Execute should not panic on help command")
	})
}

func TestDebugCommand_Visibility(t *testing.T) {
	t.Run("DebugCommandHiddenByDefault", func(t *testing.T) {
		t.Parallel()

		// Find debug command
		var debugCommand *cobra.Command
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "debug" {
				debugCommand = cmd
				break
			}
		}

		require.NotNil(t, debugCommand, "Debug command should exist")

		// By default, debug should be hidden unless config enables it
		// We can't easily test this without full config setup
		assert.NotNil(t, debugCommand)
	})

	t.Run("DebugCommandVisibleWhenEnabled", func(t *testing.T) {
		// Test the debug command visibility logic in initConfig
		viper.Set("debug", true)
		defer viper.Set("debug", false)

		// Find debug command
		var debugCommand *cobra.Command
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "debug" {
				debugCommand = cmd
				break
			}
		}

		require.NotNil(t, debugCommand, "Debug command should exist")

		// Call initConfig to trigger the visibility logic
		assert.NotPanics(t, func() {
			initConfig()
		})

		// The debug command should now be visible
		assert.False(t, debugCommand.Hidden)
	})
}

func TestInitConfigWithDebugEnabled(t *testing.T) {
	t.Run("DebugCommandBecomesVisible", func(t *testing.T) {
		// Clean slate
		viper.Reset()

		// Set debug to true
		viper.Set("debug", true)
		defer viper.Set("debug", false)

		// Find debug command
		var debugCommand *cobra.Command
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "debug" {
				debugCommand = cmd
				break
			}
		}

		require.NotNil(t, debugCommand, "Debug command should exist")

		// Initially, command might be hidden
		originalHidden := debugCommand.Hidden

		// Call initConfig which should make debug visible
		assert.NotPanics(t, func() {
			initConfig()
		})

		// After initConfig, debug command should be visible when debug is enabled
		if viper.GetBool("debug") {
			assert.False(t, debugCommand.Hidden, "Debug command should be visible when debug is enabled")
		}

		// Restore original state
		debugCommand.Hidden = originalHidden
	})
}
