package telemetry

import (
	"encoding/json"
	"maps"
	"os"

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
	envs := []string{"RUDDERSTACK_CLI_VERSION", "RUDDERSTACK_CLI_RUNNER_VERSION", "CI_PLATFORM"}
	executionContext := make(map[string]interface{})
	isCI := false

	for _, envVar := range envs {
		if os.Getenv(envVar) != "" {
			isCI = true
			executionContext[envVar] = os.Getenv(envVar)
		}
	}

	executionContext["IS_CI"] = isCI
	return executionContext
}

func TrackCommand(command string, err error, extras ...KV) {

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
