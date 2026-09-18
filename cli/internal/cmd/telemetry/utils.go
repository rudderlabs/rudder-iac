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

// TrackCommand is a variable so tests can observe what a command reports
// without sending events.
//
// Callers report from a deferred call in RunE, which is why those RunE
// signatures use a named error return: an unnamed return is copied before the
// defer runs, so the defer would otherwise see nil.
var TrackCommand = trackCommand

// reported records whether this invocation already emitted a command event, so
// TrackUnreportedFailure does not double count one a hook already reported. One
// invocation is one process, which is why package state is the right scope.
// Tests that replace TrackCommand bypass it; only Execute reads it.
var reported atomic.Bool

// TrackUnreportedFailure reports a failure no command reported itself.
//
// cobra rejects several classes of failure outside the two hooks this package
// wraps: ValidateArgs and PersistentPreRunE run before PreRunE, and
// ValidateRequiredFlags and ValidateFlagGroups sit between PreRunE and RunE. A
// missing required flag, a stray argument, a flag parse error and a mistyped
// command therefore emitted nothing at all — and those are precisely the
// drop-off points an adoption funnel needs to see. Execute is the only place
// outside every hook, so the fallback lives there; args is passed so the
// command resolves the way cobra resolved it.
func TrackUnreportedFailure(root *cobra.Command, args []string, err error) {
	if err == nil || reported.Load() {
		return
	}

	// An unresolvable path is a mistyped or unknown command, which is its own
	// funnel step rather than something to attribute to a command.
	command := "unknown"
	if target, _, findErr := root.Find(args); findErr == nil && target != nil {
		command = strings.TrimPrefix(target.CommandPath(), target.Root().Name()+" ")
	}

	TrackCommand(command, err, KV{K: "stage", V: "unreported"})
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
// loading) across the command tree. Commands track themselves from RunE,
// which cobra skips when PreRunE fails, so there is no double counting.
//
// The name is the command path below the root, which has to match the name the
// same command passes to TrackCommand from RunE: a command reporting under two
// names splits into two funnel steps. TestTrackedNamesMatchCommandPaths pins
// every RunE literal to its path for that reason.
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
