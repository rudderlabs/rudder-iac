package telemetry

import (
	"encoding/json"
	"maps"
	"os"
	"strings"
	"sync/atomic"

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

// TrackCommand is a variable so tests can observe what a command reports.
// Callers report from a deferred call in RunE, so RunE needs a named error
// return: an unnamed return is copied before the defer runs.
var TrackCommand = trackCommand

// reported is set once any command event is emitted, so the fallback in
// TrackUnreportedFailure does not count a failure twice.
var reported atomic.Bool

// telemetryReady is a variable so tests can stand in for loaded config.
var telemetryReady = telemetry.Ready

// CommandName is the command path below the root. Every tracking site derives
// its name from here, so RunE and PreRunE report the same name by construction.
func CommandName(cmd *cobra.Command) string {
	return strings.TrimPrefix(cmd.CommandPath(), cmd.Root().Name()+" ")
}

// TrackUnreportedFailure reports a failure that no hook reported. cobra rejects
// stray arguments, missing required flags and flag groups outside PreRunE and
// RunE, so Execute is the only place that sees them.
//
// It reports nothing for flag-parse errors or an unknown command: cobra fails
// those before the OnInitialize hooks load config and telemetry, and without
// the config the opt-out setting cannot be honoured.
func TrackUnreportedFailure(root *cobra.Command, args []string, err error) {
	if err == nil || reported.Load() || !telemetryReady() {
		return
	}

	target, _, findErr := root.Find(args)
	if findErr != nil || target == nil {
		return
	}

	TrackCommand(CommandName(target), err, KV{K: "stage", V: "unreported"})
}

func trackCommand(command string, err error, extras ...KV) {
	reported.Store(true)

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
// loading) across the command tree under the same name RunE uses. Call it once,
// after every command is registered: a second call wraps each PreRunE again and
// reports every failure twice.
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
			TrackCommand(CommandName(c), err, KV{K: "stage", V: "pre_run"})
		}
		return err
	}
}
