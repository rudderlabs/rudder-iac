package telemetry

import (
	"encoding/json"
	"maps"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rudderlabs/rudder-iac/cli/internal/config"
	"github.com/rudderlabs/rudder-iac/cli/internal/telemetry"
)

const (
	CommandExecutedEvent = "CLI Command Executed"
)

type KV struct {
	K string
	V interface{}
}

// getCIExecutionContext returns the execution context of the CLI if running in CI.
func getCIExecutionContext() map[string]interface{} {
	executionContext := make(map[string]interface{})
	envMap := map[string]string{
		"RUDDERSTACK_CLI_WORKFLOW_VERSION": "cli_workflow_version",
		"RUDDERSTACK_CLI_CI_PLATFORM":      "ci_platform",
	}

	for envVar, key := range envMap {
		envValue := os.Getenv(envVar)
		if envValue != "" {
			executionContext[key] = envValue
		}
	}

	return executionContext
}

// TrackCommand is a variable so tests can observe what a command reports
// without sending events.
var TrackCommand = trackCommand

func trackCommand(command string, err error, extras ...KV) {
	props := map[string]interface{}{
		"command": command,
		"errored": err != nil,
	}

	for _, extra := range extras {
		props[extra.K] = extra.V
	}

	// Automatically add experimental flags
	cfg := config.GetConfig()
	experimentalData, _ := json.Marshal(cfg.ExperimentalFlags)
	var experimental map[string]interface{}
	json.Unmarshal(experimentalData, &experimental)
	props["experimental"] = experimental

	// Automatically add execution context (CI)
	maps.Copy(props, getCIExecutionContext())

	if err := telemetry.TrackEvent(CommandExecutedEvent, props); err != nil {
		log.Error("failed to track command", "error", err)
	}
}

// TrackPreRunFailures reports PreRunE failures (auth, workspace lookup, spec
// loading) across the command tree. Commands track themselves from RunE,
// which cobra skips when PreRunE fails, so there is no double counting.
//
// ponytail: the command name is the path below the root, which matches the
// name every current PreRunE command passes to TrackCommand from RunE; a new
// command tracking under a different name would split into two funnel steps.
func TrackPreRunFailures(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		TrackPreRunFailures(sub)
	}

	preRunE := cmd.PreRunE
	if preRunE == nil {
		return
	}

	cmd.PreRunE = func(c *cobra.Command, args []string) error {
		err := preRunE(c, args)
		if err != nil {
			command := strings.TrimPrefix(c.CommandPath(), c.Root().Name()+" ")
			TrackCommand(command, err, KV{K: "stage", V: "pre_run"})
		}
		return err
	}
}
